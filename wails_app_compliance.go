package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hzhan516/medmemo/internal/application"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// EmergencyResult 紧急症状检测结果。
type EmergencyResult struct {
	Level   string `json:"level"`   // A, B, none
	Message string `json:"message"` // 提示信息
	Action  string `json:"action"`  // 建议操作
}

// CheckEmergency 检查文本是否包含紧急症状（AGENTS.md 7.3）。
// 委托 application 层的 EvaluateEmergency 执行本地关键词匹配，延迟 <5ms，独立于 AI 回复流程。
func (a *WailsApp) CheckEmergency(text string) (*EmergencyResult, error) {
	result := application.EvaluateEmergency(text)
	return &EmergencyResult{
		Level:   string(result.Level),
		Message: result.Message,
		Action:  result.Action,
	}, nil
}

// DisclaimerStatus 返回当前免责声明状态，供前端在启动时检测是否需要展示。
type DisclaimerStatus struct {
	Required bool   `json:"required"`
	Text     string `json:"text"`
	Version  string `json:"version"`
}

// GetDisclaimerStatus 查询用户是否需要同意当前版本的免责声明。
func (a *WailsApp) GetDisclaimerStatus() (*DisclaimerStatus, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	rec, err := a.disclaimerRepo.GetAcceptance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get disclaimer status: %w", err)
	}

	// 若从未同意，或已同意版本低于当前版本，均需重新展示
	if rec == nil || rec.Version != entity.CurrentDisclaimerVersion {
		return &DisclaimerStatus{
			Required: true,
			Text:     entity.DisclaimerText,
			Version:  entity.CurrentDisclaimerVersion,
		}, nil
	}

	return &DisclaimerStatus{
		Required: false,
		Text:     entity.DisclaimerText,
		Version:  entity.CurrentDisclaimerVersion,
	}, nil
}

// AcceptDisclaimer 记录用户同意当前版本的免责声明。
func (a *WailsApp) AcceptDisclaimer(version string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	if version != entity.CurrentDisclaimerVersion {
		return fmt.Errorf("disclaimer version mismatch: expected %s, got %s", entity.CurrentDisclaimerVersion, version)
	}

	rec := &entity.DisclaimerAcceptance{
		Version:    version,
		AcceptedAt: time.Now(),
		TextHash:   "", // 当前阶段无需哈希校验，预留字段
	}
	if err := a.disclaimerRepo.SaveAcceptance(ctx, rec); err != nil {
		return fmt.Errorf("failed to save disclaimer acceptance: %w", err)
	}
	return nil
}

// DeclineDisclaimer 用户不同意免责声明，退出应用。
func (a *WailsApp) DeclineDisclaimer() {
	runtime.Quit(a.ctx)
}

// ShowEmergencyDialog 触发紧急症状弹窗（供前端调用）。
func (a *WailsApp) ShowEmergencyDialog(title, message string) {
	_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.WarningDialog,
		Title:   title,
		Message: message,
	})
}

// ReportComplianceFeedback 接收前端提交的合规误判反馈。
func (a *WailsApp) ReportComplianceFeedback(ruleID string, originalText string) error {
	logger := application.NewComplianceLogger("data")
	if err := logger.LogFeedback(a.ctx, ruleID, originalText, "false_positive"); err != nil {
		return fmt.Errorf("failed to log compliance feedback: %w", err)
	}
	return nil
}
