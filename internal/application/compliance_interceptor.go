// Package application 实现应用用例层，编排领域对象完成完整业务流程。
package application

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"sync"
	"time"
)

// RiskLevel 表示合规风险等级。
type RiskLevel int

const (
	L1Blocked RiskLevel = iota + 1 // L1-阻断级：诊断/处方/治疗
	L2Warning                      // L2-警告级：暗示性诊断/药物推荐
	L3Notice                       // L3-提示级：严重疾病科普
	L4Normal                       // L4-正常级：一般健康科普/生活方式
)

// String 返回风险等级的可读名称。
func (r RiskLevel) String() string {
	switch r {
	case L1Blocked:
		return "L1_BLOCKED"
	case L2Warning:
		return "L2_WARNING"
	case L3Notice:
		return "L3_NOTICE"
	case L4Normal:
		return "L4_NORMAL"
	default:
		return "UNKNOWN"
	}
}

// ComplianceResult 合规检查结果。
type ComplianceResult struct {
	Level       string // "L1_BLOCKED" | "L2_WARNING" | "L3_NOTICE" | "L4_NORMAL"
	Blocked     bool   // L1 为 true
	MatchedRule string // 命中的规则 ID
	SafeText    string // L1 时为替换文本；其他等级为原文
	Warning     string // L2 时的警告文案
	Notice      string // L3 时的提示文案
}

// ComplianceRule 单条合规规则定义。
type ComplianceRule struct {
	ID          string   `json:"id"`
	Level       string   `json:"level"`
	Name        string   `json:"name"`
	Patterns    []string `json:"patterns"`
	Action      string   `json:"action"`
	Replacement string   `json:"replacement,omitempty"`
	Warning     string   `json:"warning,omitempty"`
	Notice      string   `json:"notice,omitempty"`
}

// ComplianceRuleSet 规则库顶层结构。
type ComplianceRuleSet struct {
	Version     string           `json:"version"`
	UpdatedAt   string           `json:"updated_at"`
	Description string           `json:"description"`
	Rules       []ComplianceRule `json:"rules"`
}

// compiledRule 预编译后的规则，用于运行时高效匹配。
type compiledRule struct {
	ComplianceRule
	compiled []*regexp.Regexp
}

// ComplianceInterceptor 四层合规风险拦截引擎。
// 支持从本地 JSON 文件加载规则库，运行时热更新。
type ComplianceInterceptor struct {
	mu      sync.RWMutex
	rules   []compiledRule
	version string
	path    string
}

// NewComplianceInterceptor 从指定路径加载规则库并创建拦截器。
func NewComplianceInterceptor(rulesPath string) (*ComplianceInterceptor, error) {
	ci := &ComplianceInterceptor{path: rulesPath}
	if err := ci.load(); err != nil {
		// 故障降级：规则库加载失败时返回空规则拦截器，确保业务不中断
		return ci, fmt.Errorf("failed to load compliance rules from %s: %w", rulesPath, err)
	}
	return ci, nil
}

// load 从文件加载并编译规则库。
func (ci *ComplianceInterceptor) load() error {
	data, err := os.ReadFile(ci.path)
	if err != nil {
		return fmt.Errorf("read rules file failed: %w", err)
	}

	var rs ComplianceRuleSet
	if err := json.Unmarshal(data, &rs); err != nil {
		return fmt.Errorf("parse rules json failed: %w", err)
	}

	compiled := make([]compiledRule, 0, len(rs.Rules))
	for _, r := range rs.Rules {
		cr := compiledRule{ComplianceRule: r}
		for _, p := range r.Patterns {
			re, err := regexp.Compile(p)
			if err != nil {
				return fmt.Errorf("compile pattern %q for rule %s failed: %w", p, r.ID, err)
			}
			cr.compiled = append(cr.compiled, re)
		}
		compiled = append(compiled, cr)
	}

	// 按风险等级排序：L1 > L2 > L3，确保高优先级规则先匹配
	sort.SliceStable(compiled, func(i, j int) bool {
		return levelPriority(compiled[i].Level) < levelPriority(compiled[j].Level)
	})

	ci.mu.Lock()
	ci.rules = compiled
	ci.version = rs.Version
	ci.mu.Unlock()

	return nil
}

// levelPriority 返回风险等级的数值优先级（越小优先级越高）。
func levelPriority(level string) int {
	switch level {
	case "L1":
		return 1
	case "L2":
		return 2
	case "L3":
		return 3
	default:
		return 99
	}
}

// ReloadRules 运行时重新加载规则库，实现热更新。
func (ci *ComplianceInterceptor) ReloadRules() error {
	return ci.load()
}

// Version 返回当前加载的规则库版本号。
func (ci *ComplianceInterceptor) Version() string {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	return ci.version
}

// Evaluate 评估单条文本的风险等级，返回合规检查结果。
// 匹配策略：按 L1→L2→L3 优先级顺序匹配，一旦命中立即短路返回。
// 若无任何规则命中，返回 L4_NORMAL。
func (ci *ComplianceInterceptor) Evaluate(ctx context.Context, text string) (*ComplianceResult, error) {
	ci.mu.RLock()
	rules := ci.rules
	ci.mu.RUnlock()

	// 无规则时直接放行（降级策略）
	if len(rules) == 0 {
		return &ComplianceResult{Level: L4Normal.String(), SafeText: text}, nil
	}

	for _, r := range rules {
		if matched := ci.matchRule(text, r); matched {
			switch r.Level {
			case "L1":
				replacement := r.Replacement
				if replacement == "" {
					replacement = "我无法提供医疗诊断或治疗建议。如有健康疑虑，请咨询专业医生。"
				}
				return &ComplianceResult{
					Level:       L1Blocked.String(),
					Blocked:     true,
					MatchedRule: r.ID,
					SafeText:    replacement,
				}, nil
			case "L2":
				return &ComplianceResult{
					Level:       L2Warning.String(),
					Blocked:     false,
					MatchedRule: r.ID,
					SafeText:    text,
					Warning:     r.Warning,
				}, nil
			case "L3":
				return &ComplianceResult{
					Level:       L3Notice.String(),
					Blocked:     false,
					MatchedRule: r.ID,
					SafeText:    text,
					Notice:      r.Notice,
				}, nil
			}
		}
	}

	return &ComplianceResult{Level: L4Normal.String(), SafeText: text}, nil
}

// matchRule 检查文本是否命中某条规则的任一模式。
func (ci *ComplianceInterceptor) matchRule(text string, r compiledRule) bool {
	for _, re := range r.compiled {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

// EvaluateWithTimeout 带超时的合规评估，防止极端场景下检测挂起。
func (ci *ComplianceInterceptor) EvaluateWithTimeout(ctx context.Context, text string, timeout time.Duration) (*ComplianceResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		res *ComplianceResult
		err error
	}
	done := make(chan result, 1)

	go func() {
		res, err := ci.Evaluate(ctx, text)
		done <- result{res, err}
	}()

	select {
	case <-ctx.Done():
		return &ComplianceResult{Level: L4Normal.String(), SafeText: text}, nil
	case r := <-done:
		return r.res, r.err
	}
}
