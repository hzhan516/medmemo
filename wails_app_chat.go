package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hzhan516/medmemo/internal/application/stream"
	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// SendMessageRequest 前端发送消息请求。
type SendMessageRequest struct {
	ConversationID string           `json:"conversation_id"`
	Messages       []models.Message `json:"messages"`
	Model          string           `json:"model"`
	ProviderID     string           `json:"provider_id"`
	AIMessageID    string           `json:"ai_message_id"`
}

// SendMessageResponse 发送消息响应。
type SendMessageResponse struct {
	Reply            string                 `json:"reply"`
	ConfidenceResult map[string]interface{} `json:"confidence_result"`
	Warnings         []string               `json:"warnings"`
}

const (
	defaultChatTimeout   = 30 * time.Second
	defaultStreamTimeout = 5 * time.Minute
	minStreamTimeout     = 120 * time.Second
)

// SendMessage 发送对话消息，编排完整对话流程（非流式）。
func (a *WailsApp) SendMessage(req SendMessageRequest) (*SendMessageResponse, error) {
	ctx, cancel := context.WithTimeout(a.ctx, defaultChatTimeout)
	defer cancel()

	chatReq := usecase.ChatRequest{
		ConversationID:       models.ConversationID(req.ConversationID),
		Messages:             req.Messages,
		Model:                models.ProviderType(req.Model),
		ProviderID:           req.ProviderID,
		DesensitizationLevel: a.config.DesensitizationLevel,
	}

	a.maybeAutoCompress(ctx, &chatReq)

	resp, err := a.chatOrchestrator.Execute(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	confJSON := map[string]interface{}{}
	if resp.ConfidenceResult != nil {
		confJSON = confidenceResultToMap(resp.ConfidenceResult)
	}
	// 保存用户消息（非流式路径）
	if len(req.Messages) > 0 {
		lastUser := req.Messages[len(req.Messages)-1]
		if lastUser.Role == models.RoleUser {
			a.saveUserMessage(ctx, req.ConversationID, lastUser)
		}
	}
	// 保存 AI 回复
	a.saveMessages(ctx, req.ConversationID, req.Messages, resp.Reply, nil, resp.ConfidenceResult, req.ProviderID, req.AIMessageID)

	return &SendMessageResponse{
		Reply:            resp.Reply,
		ConfidenceResult: confJSON,
		Warnings:         resp.Warnings,
	}, nil
}

// SendMessageStream 发送流式对话请求，通过 Wails Events 实时推送结构化 StreamChunk。
func (a *WailsApp) SendMessageStream(req SendMessageRequest) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[panic] SendMessageStream: %v\n", r)
			err = fmt.Errorf("stream internal error: %v", r)
		}
	}()
	streamTimeoutVal := a.streamTimeout(req.ProviderID)

	ctx, cancel := context.WithTimeout(a.ctx, streamTimeoutVal)

	a.streamMu.Lock()
	a.activeStreams[req.ConversationID] = cancel
	a.streamMu.Unlock()

	defer func() {
		a.streamMu.Lock()
		delete(a.activeStreams, req.ConversationID)
		a.streamMu.Unlock()
		cancel()
	}()

	chatReq := usecase.ChatRequest{
		ConversationID:       models.ConversationID(req.ConversationID),
		Messages:             req.Messages,
		Model:                models.ProviderType(req.Model),
		ProviderID:           req.ProviderID,
		DesensitizationLevel: a.config.DesensitizationLevel,
	}

	a.maybeAutoCompress(ctx, &chatReq)

	// 统一流式处理层：将原始 callback 包装为结构化 StreamChunk 序列
	broker := stream.NewBroker(req.Model, "", func(chunk models.StreamChunk) {
		chunk.Metadata.ConversationID = req.ConversationID
		runtime.EventsEmit(a.ctx, "chat:stream_chunk", chunk)
	})
	broker.Start()

	// 立即保存用户消息，确保切换会话时可见（不阻塞流式生成）
	if len(req.Messages) > 0 {
		lastUser := req.Messages[len(req.Messages)-1]
		if lastUser.Role == models.RoleUser {
			// 使用应用生命周期 context 异步保存，避免占用 stream 超时 budget
			go a.saveUserMessage(a.ctx, req.ConversationID, lastUser)
		}
	}

	// 收集 AI 完整回复用于持久化
	var fullReply strings.Builder

	usage, confidenceResult, finalContent, err := a.chatOrchestrator.StreamExecute(ctx, chatReq, func(chunk string) {
		fullReply.WriteString(chunk)
		broker.Content(chunk)
	})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			broker.Error("生成已中断")
			// 保存已生成的部分内容（无置信度，token 为 0）
			a.saveMessages(ctx, req.ConversationID, req.Messages, fullReply.String(), nil, nil, req.ProviderID, req.AIMessageID)
			return nil
		}
		broker.Error(err.Error())
		return fmt.Errorf("stream failed: %w", err)
	}

	// 若最终内容与流式过程中展示的不同（脱敏还原或合规替换），通知前端替换
	if finalContent != fullReply.String() {
		runtime.EventsEmit(a.ctx, "chat:stream:replace", map[string]any{
			"conversation_id": req.ConversationID,
			"content":         finalContent,
		})
	}

	// 保存用户消息和 AI 回复（携带 token 与置信度）
	a.saveMessages(ctx, req.ConversationID, req.Messages, finalContent, usage, confidenceResult, req.ProviderID, req.AIMessageID)

	// 流式结束后对完整内容做一次合规检测（MVP 简化策略）
	compResult, compErr := a.chatOrchestrator.CheckCompliance(ctx, finalContent)
	if compErr == nil && compResult.Level != "L4_NORMAL" {
		payload := map[string]any{
			"conversation_id": req.ConversationID,
			"level":           compResult.Level,
			"warning":         compResult.Warning,
			"notice":          compResult.Notice,
			"replacedTerms":   compResult.ReplacedTerms,
			"matchedRule":     compResult.MatchedRule,
		}
		runtime.EventsEmit(a.ctx, "chat:stream:compliance", payload)
	}

	// 推送置信度与 Token 用量事件
	if confidenceResult != nil {
		confidencePayload := map[string]any{
			"conversation_id":   req.ConversationID,
			"confidence":        confidenceResultToMap(confidenceResult),
			"prompt_tokens":     0,
			"completion_tokens": 0,
			"total_tokens":      0,
			"truncated":         false,
		}
		if usage != nil {
			confidencePayload["prompt_tokens"] = usage.PromptTokens
			confidencePayload["completion_tokens"] = usage.CompletionTokens
			confidencePayload["total_tokens"] = usage.TotalTokens
			confidencePayload["truncated"] = usage.FinishReason == "length"
		}
		runtime.EventsEmit(a.ctx, "chat:stream:confidence", confidencePayload)
	}

	broker.Done(usage)
	return nil
}

// maybeAutoCompress 在发送前估算上下文用量比例；若达到自动压缩阈值，
// 则调用 compressionService 压缩会话消息并替换 chatReq.Messages。
// 组装（记忆/知识检索 + 脱敏）在此只做一次，未触发压缩时透传给 Execute/StreamExecute 复用，
// 避免发送路径重复组装（C5-3）。
func (a *WailsApp) maybeAutoCompress(ctx context.Context, chatReq *usecase.ChatRequest) {
	if a.compressionService == nil || a.contextEstimator == nil || a.chatOrchestrator == nil {
		return
	}

	// 组装一次真实 prompt（记忆/知识检索 + 脱敏），供估算与发送复用。
	prepared := a.chatOrchestrator.PreparePrompt(ctx, *chatReq)

	est, err := a.contextEstimator.Estimate(ctx, usecase.EstimatorInput{
		Messages:        chatReq.Messages,
		AssembledPrompt: prepared.Messages,
		ProviderID:      chatReq.ProviderID,
		ModelID:         string(chatReq.Model),
	})
	if err != nil {
		fmt.Printf("[auto-compress] estimate failed: %v\n", err)
		return
	}

	if est.Ratio < models.AutoCompressionThreshold {
		// 未触发压缩：消息未变，复用已组装结果，避免 Execute/StreamExecute 二次组装。
		p := prepared
		chatReq.Prepared = &p
		return
	}

	cfg, providerID, modelID := a.buildCompressionConfig(chatReq.ProviderID, string(chatReq.Model))

	res, err := a.compressionService.CompressMessages(ctx, chatReq.Messages, providerID, modelID, cfg)
	if err != nil {
		fmt.Printf("[auto-compress] failed, proceeding uncompressed: %v\n", err)
		// 压缩失败：消息未变，仍可复用已组装结果。
		p := prepared
		chatReq.Prepared = &p
		return
	}

	// 压缩改变了消息集合 -> 已组装结果失效，Execute/StreamExecute 需基于压缩后消息重新组装。
	chatReq.Messages = res.Messages
	chatReq.Prepared = nil
	runtime.EventsEmit(a.ctx, "context:auto_compressed", map[string]any{
		"conversation_id": string(chatReq.ConversationID),
		"used_after":      res.UsedAfter,
		"fallback":        res.FallbackOccurred,
	})
}

func (a *WailsApp) streamTimeout(providerID string) time.Duration {
	timeout := defaultStreamTimeout
	if a.providerStore == nil || providerID == "" {
		return timeout
	}

	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
	defer cancel()

	provider, err := a.providerStore.Get(ctx, providerID)
	if err != nil || provider == nil {
		return timeout
	}
	if provider.TimeoutMs <= 0 {
		return timeout
	}

	configured := time.Duration(provider.TimeoutMs) * time.Millisecond
	if configured < minStreamTimeout {
		return minStreamTimeout
	}
	return configured
}

// saveUserMessage 单独保存一条用户消息并更新会话时间戳，
// 用于流式生成启动前立即持久化，确保切换会话时可见。
// 错误通过 Wails Events 推送，确保异步保存失败可观测。
func (a *WailsApp) saveUserMessage(ctx context.Context, convID string, message models.Message) {
	if a.msgRepo == nil || convID == "" || message.Role != models.RoleUser {
		return
	}
	msgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	if err := a.msgRepo.Save(ctx, models.ConversationID(convID), &entity.Message{
		ID:        msgID,
		Role:      message.Role,
		Content:   message.Content,
		Timestamp: time.Now(),
	}); err != nil {
		a.safeEventsEmit("chat:save_error", map[string]any{
			"type":            "user_message",
			"conversation_id": convID,
			"error":           err.Error(),
			"timestamp":       time.Now().UnixMilli(),
		})
		fmt.Printf("[saveUserMessage] 保存用户消息失败: %v\n", err)
	}
	// 同步归档到 raw_dialogues，为事实提取提供源数据
	if a.dialogueRepo != nil {
		if err := a.dialogueRepo.Insert(ctx, entity.NewRawDialogue(convID, entity.RoleUser, message.Content, "")); err != nil {
			a.safeEventsEmit("chat:save_error", map[string]any{
				"type":            "raw_dialogue",
				"conversation_id": convID,
				"error":           err.Error(),
				"timestamp":       time.Now().UnixMilli(),
			})
		}
	}
	if a.convRepo != nil {
		if err := a.convRepo.UpdateTimestamp(ctx, models.ConversationID(convID), time.Now()); err != nil {
			a.safeEventsEmit("chat:save_error", map[string]any{
				"type":            "update_timestamp",
				"conversation_id": convID,
				"error":           err.Error(),
				"timestamp":       time.Now().UnixMilli(),
			})
			fmt.Printf("[saveUserMessage] 更新会话时间失败: %v\n", err)
		}
	}
}

// saveMessages 保存 AI 回复消息到数据库，可选携带 token 用量与置信度。
// 用户消息应由调用方通过 saveUserMessage 单独保存，避免重复。
// providerID 用于异步事实提取时创建对应的 LLM client。
// aiMessageID 由前端生成并传入，确保反馈统计与持久化消息使用同一 message_id。
func (a *WailsApp) saveMessages(ctx context.Context, convID string, messages []models.Message, aiReply string, usage *models.TokenUsage, confidence *entity.ConfidenceResult, providerID string, aiMessageID string) {
	if a.msgRepo == nil || convID == "" {
		return
	}
	// 保存 AI 回复
	if aiReply != "" {
		msgID := aiMessageID
		if msgID == "" {
			msgID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
		}
		msg := &entity.Message{
			ID:        msgID,
			Role:      models.RoleAssistant,
			Content:   aiReply,
			Timestamp: time.Now(),
		}
		if usage != nil {
			msg.PromptTokens = usage.PromptTokens
			msg.CompletionTokens = usage.CompletionTokens
		}
		if confidence != nil {
			msg.ConfidenceScore = confidence.OverallScore
			msg.ConfidenceLevel = string(confidence.Level)
			if confMap := confidenceResultToMap(confidence); confMap != nil {
				if jsonBytes, err := json.Marshal(confMap); err == nil {
					msg.ConfidenceJSON = string(jsonBytes)
				}
			}
		}
		if err := a.msgRepo.Save(ctx, models.ConversationID(convID), msg); err != nil {
			fmt.Printf("[saveMessages] 保存 AI 回复失败: %v\n", err)
		}
		// 同步归档 AI 回复到 raw_dialogues
		if a.dialogueRepo != nil {
			_ = a.dialogueRepo.Insert(ctx, entity.NewRawDialogue(convID, entity.RoleAssistant, aiReply, ""))
		}
		// 异步执行事实提取（不阻塞主流程）
		if a.chatOrchestrator != nil && providerID != "" {
			var userContent string
			if len(messages) > 0 {
				lastUser := messages[len(messages)-1]
				if lastUser.Role == models.RoleUser {
					userContent = lastUser.Content
				}
			}
			go a.extractFactsAsync(userContent, aiReply, providerID)
		}
	}
	// 更新会话时间
	if a.convRepo != nil {
		if err := a.convRepo.UpdateTimestamp(ctx, models.ConversationID(convID), time.Now()); err != nil {
			fmt.Printf("[saveMessages] 更新会话时间失败: %v\n", err)
		}
	}
}

// extractFactsAsync 异步从完整对话（用户消息 + AI 回复）中提取事实并保存到 factRepo。
// 由 ChatOrchestrator 统一调度限流，避免与主对话竞争 API 配额触发 429。
func (a *WailsApp) extractFactsAsync(userContent, aiReply, providerID string) {
	ctx, cancel := context.WithTimeout(a.ctx, 60*time.Second)
	defer cancel()
	facts, err := a.chatOrchestrator.ExtractFactsFromReply(ctx, userContent, aiReply, providerID)
	if err != nil {
		// 429 限流时静默跳过，不记录错误（这是预期行为）
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "rate limited") {
			fmt.Printf("[extractFactsAsync] skipped due to rate limit (expected)\n")
			return
		}
		fmt.Printf("[extractFactsAsync] 事实提取失败: %v\n", err)
		return
	}
	for _, f := range facts {
		if err := a.factRepo.Save(ctx, f); err != nil {
			fmt.Printf("[extractFactsAsync] 保存事实失败 %s: %v\n", f.FactID, err)
			continue
		}
		// pending fact 不生成 embedding，embedding 只在审批通过后生成，
		// 避免未审核或后续被拒绝的事实污染向量召回空间。
	}
	fmt.Printf("[extractFactsAsync] 提取并保存 %d 条待审核事实\n", len(facts))
}

// StopGeneration 中断所有正在进行的流式生成。
func (a *WailsApp) StopGeneration() {
	a.streamMu.Lock()
	for _, cancel := range a.activeStreams {
		if cancel != nil {
			cancel()
		}
	}
	a.streamMu.Unlock()
}

// GenerateTitle 异步生成会话标题，通过 Wails Events 推送结果。
// 前端应在首条用户消息发送后调用此方法。
func (a *WailsApp) GenerateTitle(convID string, userMessage string) {
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
		defer cancel()

		title, err := a.titleGen.Generate(ctx, userMessage)
		if err != nil {
			// AI 生成失败或超时，降级到本地规则
			title = usecase.FallbackTitle(userMessage)
		}

		// 持久化到数据库
		if a.convRepo != nil {
			if err := a.convRepo.UpdateTitle(ctx, models.ConversationID(convID), title); err != nil {
				fmt.Printf("[GenerateTitle] 更新标题失败: %v\n", err)
			}
		}

		// 推送前端更新
		runtime.EventsEmit(a.ctx, "chat:title:generated", map[string]string{
			"conv_id": convID,
			"title":   title,
		})
	}()
}

// confidenceResultToMap 将 entity.ConfidenceResult 转为 map[string]interface{}，
// 供 Wails JSON 绑定序列化。
func confidenceResultToMap(r *entity.ConfidenceResult) map[string]interface{} {
	if r == nil {
		return nil
	}
	m := map[string]interface{}{
		"overall_score": r.OverallScore,
		"level":         string(r.Level),
		"explanation":   r.Explanation,
		"suggestion":    r.Suggestion,
		"missing_info":  r.MissingInfo,
		"citations":     r.Citations,
	}
	if r.Breakdown != nil {
		m["breakdown"] = r.Breakdown
	}
	return m
}
