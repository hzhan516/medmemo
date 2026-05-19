package desensitizer

import (
	"embed"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/medmemo/medmemo/pkg/models"
)

//go:embed rules/*.json
var rulesFS embed.FS

// RuleConfig 定义单条脱敏规则的配置。
type RuleConfig struct {
	Name        string   `json:"name"`
	EntityType  string   `json:"entity_type"`
	Patterns    []string `json:"patterns"`
	Keywords    []string `json:"keywords"`
	MinDigits   int      `json:"min_digits"`
	Level       string   `json:"level"`
	Placeholder string   `json:"placeholder"`
}

// compiledRule 是编译后的规则，包含预编译的 regexp。
type compiledRule struct {
	config   RuleConfig
	patterns []*regexp.Regexp
	level    models.SensitivityLevel
}

// 规则文件加载顺序，确保去重时高优先级规则先被处理（与原始硬编码顺序一致）。
var ruleFiles = []string{
	"id_card.json",
	"phone.json",
	"bank_card.json",
	"email.json",
	"url.json",
}

// loadDefaultRules 从嵌入的规则文件加载并编译所有规则。
func loadDefaultRules() ([]compiledRule, error) {
	var rules []compiledRule
	for _, name := range ruleFiles {
		data, err := rulesFS.ReadFile("rules/" + name)
		if err != nil {
			return nil, fmt.Errorf("read rule file %s: %w", name, err)
		}

		var rc RuleConfig
		if err := json.Unmarshal(data, &rc); err != nil {
			return nil, fmt.Errorf("parse rule file %s: %w", name, err)
		}

		if len(rc.Patterns) == 0 {
			return nil, fmt.Errorf("rule %s has no patterns", rc.Name)
		}

		cr := compiledRule{config: rc}
		for _, p := range rc.Patterns {
			re, err := regexp.Compile(p)
			if err != nil {
				return nil, fmt.Errorf("compile pattern %q for rule %s: %w", p, rc.Name, err)
			}
			cr.patterns = append(cr.patterns, re)
		}

		switch rc.Level {
		case "P2":
			cr.level = models.P2Internal
		case "P3":
			cr.level = models.P3Confidential
		default:
			cr.level = models.P3Confidential
		}

		rules = append(rules, cr)
	}

	return rules, nil
}
