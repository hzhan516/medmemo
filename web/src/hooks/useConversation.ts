import { useCallback, useEffect, useState } from 'react'
import { logger } from '@/lib/logger'
import { useChatStore } from '@/stores/chatStore'
import { useSettingsStore } from '@/stores/settingsStore'
import { useProviderStore } from '@/stores/providerStore'
import { useWails } from './useWails'
import { EventsOn } from '@wails/runtime/runtime'
import type { ConfidenceResult, ConfidenceLevel } from '@/components/confidence/types'

// 合法置信度等级集合，用于数据校验
const validConfidenceLevels: Set<ConfidenceLevel> = new Set(['A', 'B', 'C', 'D', 'E'])

function normalizeConfidenceLevel(level: unknown): ConfidenceLevel {
  return level && validConfidenceLevels.has(level as ConfidenceLevel) ? (level as ConfidenceLevel) : 'E'
}

/**
 * 会话管理 Hook，封装消息发送、流式输出与状态更新。
 */
export function useConversation() {
  const wails = useWails()
  const {
    messages,
    isStreaming,
    currentConversationId,
    emergencyAlert,
    emergencyWarningAcknowledged,
    addMessageForConversation,
    appendToLastMessageForConversation,
    setLastMessageErrorForConversation,
    setLastMessageWarningsForConversation,
    setLastMessageReplacedTermsForConversation,
    setLastMessageTokenUsageForConversation,
    setLastMessageConfidenceForConversation,
    setLastMessageTruncatedForConversation,
    replaceLastMessageForConversation,
    setStreamingForConversation,
    addConversation,
    selectConversation,
    updateConversation,
    setMessages,
    setEmergencyAlert,
    acknowledgeEmergencyWarning,
  } = useChatStore()

  // 当前活跃 Provider 及健康状态
  const activeProviderId = useSettingsStore((s) => s.activeProviderId)
  const activeModelId = useSettingsStore((s) => s.activeModelId)
  const healthStatus = useSettingsStore((s) => s.providerHealthStatus)
  const setActiveProviderId = useSettingsStore((s) => s.setActiveProviderId)
  const setActiveModelId = useSettingsStore((s) => s.setActiveModelId)
  const providers = useProviderStore((s) => s.providers)

  const activeProvider = providers.find((p) => p.id === activeProviderId)
  const activeStatus = activeProvider ? (healthStatus[activeProvider.id] ?? 'unknown') : 'unknown'

  // 获取第一个可用的 green Provider（用于回退）
  const fallbackProvider = providers.find(
    (p) => p.enabled && (healthStatus[p.id] ?? 'unknown') === 'green'
  )

  const [error, setError] = useState<string | null>(null)

  // 注册 Wails 流式事件监听
  // 所有事件按 conversation_id 路由到对应会话，消除 currentConversationId 闭包依赖
  useEffect(() => {
    const removeStreamChunk = EventsOn('chat:stream_chunk', (chunk: { type: 'start' | 'content' | 'done' | 'error'; payload: string; metadata?: { conversation_id?: string; model?: string; provider_id?: string; latency_ms?: number; token_count?: number; prompt_tokens?: number; completion_tokens?: number } }) => {
      const convId = chunk.metadata?.conversation_id
      if (!convId) return
      switch (chunk.type) {
        case 'start':
          // start chunk 仅携带 metadata，无需 UI 操作
          break
        case 'content':
          appendToLastMessageForConversation(convId, chunk.payload)
          break
        case 'done': {
          setStreamingForConversation(convId, false)
          // 流式结束后更新对应会话的预览和时间
          const convMsgs = useChatStore.getState().messagesMap[convId] || []
          const lastAssistant = [...convMsgs].reverse().find((m) => m.role === 'assistant')
          if (lastAssistant) {
            updateConversation(convId, {
              preview: lastAssistant.content.slice(0, 60),
              updatedAt: Date.now(),
            })
          }
          // 写入 token 用量统计
          if (chunk.metadata?.prompt_tokens !== undefined && chunk.metadata?.completion_tokens !== undefined) {
            setLastMessageTokenUsageForConversation(
              convId,
              chunk.metadata.prompt_tokens,
              chunk.metadata.completion_tokens,
              (chunk.metadata.prompt_tokens ?? 0) + (chunk.metadata.completion_tokens ?? 0),
            )
          }
          break
        }
        case 'error':
          setLastMessageErrorForConversation(convId, chunk.payload)
          setStreamingForConversation(convId, false)
          break
      }
    })
    const removeCompliance = EventsOn('chat:stream:compliance', (payload: { conversation_id?: string; level: string; warning: string; notice: string; replacedTerms?: string[]; matchedRule?: string }) => {
      const convId = payload.conversation_id
      if (!convId) return
      const warnings: string[] = [payload.level]
      if (payload.warning) warnings.push(`WARNING:${payload.warning}`)
      if (payload.notice) warnings.push(`NOTICE:${payload.notice}`)
      if (payload.matchedRule) warnings.push(`RULE:${payload.matchedRule}`)
      setLastMessageWarningsForConversation(convId, warnings)
      if (payload.replacedTerms && payload.replacedTerms.length > 0) {
        setLastMessageReplacedTermsForConversation(convId, payload.replacedTerms)
      }
    })
    const removeTitle = EventsOn('chat:title:generated', (payload: { conv_id: string; title: string }) => {
      updateConversation(payload.conv_id, { title: payload.title })
    })
    const removeReplace = EventsOn('chat:stream:replace', (payload: { conversation_id: string; content: string }) => {
      replaceLastMessageForConversation(payload.conversation_id, payload.content)
    })
    const removeConfidence = EventsOn('chat:stream:confidence', (payload: {
      conversation_id?: string
      confidence?: Record<string, unknown>
      prompt_tokens?: number
      completion_tokens?: number
      total_tokens?: number
      truncated?: boolean
    }) => {
      const convId = payload.conversation_id
      if (!convId) return
      // 更新 token 用量（作为 done chunk 的兜底）
      if (payload.prompt_tokens !== undefined && payload.completion_tokens !== undefined) {
        setLastMessageTokenUsageForConversation(
          convId,
          payload.prompt_tokens,
          payload.completion_tokens,
          payload.total_tokens ?? (payload.prompt_tokens + payload.completion_tokens),
        )
      }
      // 更新置信度
      if (payload.confidence) {
        const raw = payload.confidence
        const confidence: ConfidenceResult = {
          overallScore: (raw.overall_score as number) ?? 0,
          level: normalizeConfidenceLevel(raw.level),
          breakdown: {
            knowledge_source: ((raw.breakdown as Record<string, number>)?.knowledge_source) ?? 0,
            reasoning: ((raw.breakdown as Record<string, number>)?.reasoning) ?? 0,
            context: ((raw.breakdown as Record<string, number>)?.context) ?? 0,
            history: ((raw.breakdown as Record<string, number>)?.history) ?? 0,
            uncertainty: ((raw.breakdown as Record<string, number>)?.uncertainty) ?? 0,
          },
          explanation: (raw.explanation as string) ?? '',
          suggestion: (raw.suggestion as string) ?? '',
          missingInfo: (raw.missing_info as string[]) ?? [],
        }
        setLastMessageConfidenceForConversation(convId, confidence)
      }
      if (payload.truncated) {
        setLastMessageTruncatedForConversation(convId, true)
      }
    })

    return () => {
      removeStreamChunk()
      removeCompliance()
      removeTitle()
      removeReplace()
      removeConfidence()
    }
  }, [appendToLastMessageForConversation, setLastMessageErrorForConversation, setLastMessageWarningsForConversation, setLastMessageReplacedTermsForConversation, setLastMessageTokenUsageForConversation, setLastMessageConfidenceForConversation, setLastMessageTruncatedForConversation, replaceLastMessageForConversation, setStreamingForConversation, updateConversation])

  const sendMessage = useCallback(
    async (content: string) => {
      // 按会话维度检测流式状态，避免会话 A 流式中阻断会话 B 的发送
      const convStreamingIds = useChatStore.getState().streamingIds
      const currentConvId = useChatStore.getState().currentConversationId
      if (!content.trim() || (currentConvId && convStreamingIds.has(currentConvId))) return

      setError(null)

      // 确保当前会话存在
      let convId = currentConvId
      if (!convId) {
        try {
          convId = await wails.createConversation()
          const newConv = {
            id: convId,
            title: '新对话',
            updatedAt: Date.now(),
            unread: 0,
          }
          addConversation(newConv)
          selectConversation(convId)
        } catch {
          setError('创建会话失败')
          return
        }
      }

      // 紧急症状检测（独立于 AI 回复流程）
      const emergency = await wails.checkEmergency(content.trim())
      if (emergency.level === 'A') {
        setEmergencyAlert({
          level: 'A',
          message: emergency.message,
          action: emergency.action,
        })
        // A 级：不发送消息到 LLM，也不添加用户消息到列表
        // 用户需在全屏弹窗中选择操作
        return
      }
      if (emergency.level === 'B') {
        setEmergencyAlert({
          level: 'B',
          message: emergency.message,
          action: emergency.action,
        })
        // B 级：仍添加用户消息，但阻断后续 LLM 调用直到用户确认
      }

      // 在添加新消息前，先快照当前会话的历史消息（排除 AI 占位符），
      // 避免将空的 assistant 消息和重复的 user 消息发送到后端
      const snapshotMessages = useChatStore.getState().messagesMap[convId] || useChatStore.getState().messages
      const history = snapshotMessages
        .filter((m) => !(m.role === 'assistant' && m.isStreaming))
        .map((m) => ({
          role: m.role,
          content: m.content,
        }))

      // 添加用户消息（使用 ForConversation 变体确保跨会话隔离）
      const userMsg = {
        id: `msg_${Date.now()}_user`,
        role: 'user' as const,
        content: content.trim(),
        timestamp: Date.now(),
        conversationId: convId,
      }
      addMessageForConversation(convId, userMsg)

      // 更新会话 preview
      updateConversation(convId, {
        preview: content.trim().slice(0, 60),
        updatedAt: Date.now(),
      })

      // B 级未确认时，不继续发送到 LLM
      if (emergency.level === 'B' && !emergencyWarningAcknowledged) {
        return
      }

      // 添加空的 AI 消息占位（使用 ForConversation 变体确保跨会话隔离）
      const aiMsgId = `msg_${Date.now()}_ai`
      addMessageForConversation(convId, {
        id: aiMsgId,
        role: 'assistant',
        content: '',
        timestamp: Date.now(),
        isStreaming: true,
        conversationId: convId,
      })
      setStreamingForConversation(convId, true)

      // 活跃 Provider 不可用时的回退逻辑
      let targetProvider = activeProvider
      let targetModelId = activeModelId
      if (!targetProvider || activeStatus === 'red') {
        if (fallbackProvider) {
          targetProvider = fallbackProvider
          targetModelId = fallbackProvider.models?.find((m) => m.enabled)?.id || fallbackProvider.modelId
          setActiveProviderId(fallbackProvider.id)
          setActiveModelId(targetModelId)
        }
        // 若仍无可用 Provider，继续使用默认模型（向后兼容）
      }

      try {
        // 使用已快照的历史消息 + 当前用户消息构造请求，不重复读取 store
        await wails.sendMessageStream({
          conversation_id: convId,
          messages: [...history, { role: 'user', content: content.trim() }],
          model: targetModelId || targetProvider?.modelId || 'kimi-lite',
          provider_id: targetProvider?.id || '',
        } as Parameters<typeof wails.sendMessageStream>[0])

        // 首条用户消息后异步生成标题（不阻塞流式输出）
        if (history.filter((m) => m.role === 'user').length === 0) {
          wails.generateTitle(convId, content.trim()).catch(() => {
            // 标题生成失败静默处理，不影响对话流程
          })
        }
      } catch (e) {
        setLastMessageErrorForConversation(convId, String(e))
        setStreamingForConversation(convId, false)
        setError(String(e))
      }
    },
    [
      emergencyWarningAcknowledged,
      addMessageForConversation,
      setLastMessageErrorForConversation,
      setStreamingForConversation,
      addConversation,
      selectConversation,
      updateConversation,
      setEmergencyAlert,
      wails,
      activeModelId,
      setActiveModelId,
      activeProvider,
      activeStatus,
      fallbackProvider,
      setActiveProviderId,
    ]
  )

  const stopGeneration = useCallback(async () => {
    try {
      await wails.stopGeneration()
    } catch (e) {
      logger.error('停止生成失败:', e)
    }
  }, [wails])

  const retryMessage = useCallback(
    async (messageId: string) => {
      // 从 store 实时读取消息，避免闭包中 messages 陈旧
      const currentState = useChatStore.getState()
      const currentMsgs = currentState.messages
      const userMessages = currentMsgs.filter((m) => m.role === 'user')
      const lastUserMsg = userMessages[userMessages.length - 1]
      if (lastUserMsg) {
        // 移除后续的 assistant 消息（错误/中断的那条）
        const msgIndex = currentMsgs.findIndex((m) => m.id === messageId)
        if (msgIndex >= 0) {
          const truncated = currentMsgs.slice(0, msgIndex)
          const convId = currentState.currentConversationId
          useChatStore.setState((state) => ({
            messages: truncated,
            messagesMap: convId
              ? { ...state.messagesMap, [convId]: truncated }
              : state.messagesMap,
          }))
        }
        await sendMessage(lastUserMsg.content)
      }
    },
    [sendMessage]
  )

  const startNewConversation = useCallback(async () => {
    try {
      const id = await wails.createConversation()
      const newConv = {
        id,
        title: '新对话',
        updatedAt: Date.now(),
        unread: 0,
      }
      addConversation(newConv)
      selectConversation(id)
      setMessages([])
      setError(null)
      // 新建会话时清除紧急症状状态
      setEmergencyAlert(null)
    } catch {
      setError('创建新会话失败')
    }
  }, [wails, addConversation, selectConversation, setMessages, setEmergencyAlert])

  const loadConversationMessages = useCallback(
    async (convID: string) => {
      try {
        const response = await wails.getConversationMessages(convID)
        const mappedMessages = response.map((msg) => ({
          id: msg.id,
          role: msg.role as 'user' | 'assistant' | 'system',
          content: msg.content,
          timestamp: Number(msg.timestamp),
          promptTokens: msg.prompt_tokens,
          completionTokens: msg.completion_tokens,
          totalTokens: msg.total_tokens,
          confidence: msg.confidence
            ? ({
                overallScore: (msg.confidence as Record<string, unknown>).overall_score as number,
                level: normalizeConfidenceLevel((msg.confidence as Record<string, unknown>).level),
                breakdown: {
                  knowledge_source: ((msg.confidence as Record<string, unknown>).breakdown as Record<string, number>)?.knowledge_source ?? 0,
                  reasoning: ((msg.confidence as Record<string, unknown>).breakdown as Record<string, number>)?.reasoning ?? 0,
                  context: ((msg.confidence as Record<string, unknown>).breakdown as Record<string, number>)?.context ?? 0,
                  history: ((msg.confidence as Record<string, unknown>).breakdown as Record<string, number>)?.history ?? 0,
                  uncertainty: ((msg.confidence as Record<string, unknown>).breakdown as Record<string, number>)?.uncertainty ?? 0,
                },
                explanation: (msg.confidence as Record<string, unknown>).explanation as string,
                suggestion: (msg.confidence as Record<string, unknown>).suggestion as string,
                missingInfo: ((msg.confidence as Record<string, unknown>).missing_info as string[]) ?? [],
              } as ConfidenceResult)
            : undefined,
        }))
        setMessages(mappedMessages)
      } catch (e) {
        logger.error('加载会话消息失败:', e)
        setMessages([])
      }
    },
    [wails, setMessages]
  )

  // 紧急症状弹窗操作回调
  const handleEmergencyContinue = useCallback(() => {
    setEmergencyAlert(null)
  }, [setEmergencyAlert])

  const handleEmergencyNotEmergency = useCallback(() => {
    // 误判反馈：记录到控制台，关闭弹窗/横幅
    logger.warn('[Emergency Feedback] User reported false positive:', emergencyAlert)
    setEmergencyAlert(null)
  }, [emergencyAlert, setEmergencyAlert])

  const handleAcknowledgeWarning = useCallback(() => {
    acknowledgeEmergencyWarning()
    // 确认后自动触发重发最后一条用户消息（从 store 实时读取）
    const currentMsgs = useChatStore.getState().messages
    const userMessages = currentMsgs.filter((m) => m.role === 'user')
    const lastUserMsg = userMessages[userMessages.length - 1]
    if (lastUserMsg) {
      // 清除 B 级 alert 后重发
      setEmergencyAlert(null)
      sendMessage(lastUserMsg.content)
    } else {
      setEmergencyAlert(null)
    }
  }, [acknowledgeEmergencyWarning, setEmergencyAlert, sendMessage])

  return {
    messages,
    isStreaming,
    currentConversationId,
    emergencyAlert,
    sendMessage,
    stopGeneration,
    retryMessage,
    startNewConversation,
    loadConversationMessages,
    handleEmergencyContinue,
    handleEmergencyNotEmergency,
    handleAcknowledgeWarning,
    error,
    reportComplianceFeedback: async (messageId: string, ruleID: string) => {
      const msg = messages.find((m) => m.id === messageId)
      if (!msg) return
      try {
        await wails.reportComplianceFeedback(ruleID, msg.content)
        useChatStore.setState((state) => ({
          messages: state.messages.map((m) =>
            m.id === messageId ? { ...m, complianceFeedback: 'submitted' as const } : m
          ),
        }))
      } catch (e) {
        logger.error('提交合规反馈失败:', e)
      }
    },
  }
}
