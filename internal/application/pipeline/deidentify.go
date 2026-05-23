// Package pipeline 实现脱敏流水线的应用层编排。
// 将 L1 规则引擎、L2 NER 模型、L3 关键词字典串联为完整处理管道。
package pipeline

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/pkg/desensitizer"
	"github.com/hzhan516/medmemo/pkg/models"
)

// DeidentifyPipeline 三级脱敏流水线编排器。
type DeidentifyPipeline struct {
	stages []DeidentifyStage
}

// DeidentifyStage 单个脱敏阶段接口。
type DeidentifyStage interface {
	Process(ctx context.Context, input PipelineInput) (PipelineOutput, error)
}

// PipelineInput / PipelineOutput 管道数据单元。
type PipelineInput struct {
	Text     string
	Metadata map[string]any
}

type PipelineOutput struct {
	Text     string
	Metadata map[string]any
}

// NewDeidentifyPipeline 创建流水线，顺序注入各阶段。
func NewDeidentifyPipeline(stages ...DeidentifyStage) *DeidentifyPipeline {
	return &DeidentifyPipeline{stages: stages}
}

// Execute 顺序执行各阶段，任一阶段出错即短路返回。
// 收集各阶段产生的实体和 P2 级占位符映射，供输出还原使用。
func (p *DeidentifyPipeline) Execute(ctx context.Context, raw string) (models.DeidentifyResult, error) {
	input := PipelineInput{Text: raw, Metadata: make(map[string]any)}
	var allEntities []models.SensitiveEntity
	allPlaceholders := make(map[string]string)

	for _, stage := range p.stages {
		output, err := stage.Process(ctx, input)
		if err != nil {
			return models.DeidentifyResult{}, fmt.Errorf("deidentify stage %T failed: %w", stage, err)
		}

		// 收集 L1 阶段的实体和 P2 占位符映射
		if l1Entities, ok := output.Metadata["l1_entities"].([]models.SensitiveEntity); ok {
			allEntities = append(allEntities, l1Entities...)
		}
		if l1Placeholders, ok := output.Metadata["l1_placeholders"].(map[string]string); ok {
			for k, v := range l1Placeholders {
				allPlaceholders[k] = v
			}
		}
		// 收集 L2 阶段的实体
		if l2Entities, ok := output.Metadata["l2_entities"].([]models.SensitiveEntity); ok {
			allEntities = append(allEntities, l2Entities...)
		}

		input = PipelineInput(output)
	}
	return models.DeidentifyResult{
		OriginalText: raw,
		SafeText:     input.Text,
		Entities:     allEntities,
		Placeholder:  allPlaceholders,
	}, nil
}

// L1RuleStage 一级规则脱敏阶段。
// 基于 regexp 匹配身份证号、手机号、银行卡、邮箱、URL，延迟 <1ms。
type L1RuleStage struct {
	engine *desensitizer.RuleEngine
}

// NewL1RuleStage 创建 L1 规则脱敏阶段。
func NewL1RuleStage() *L1RuleStage {
	return &L1RuleStage{engine: desensitizer.NewRuleEngine()}
}

func (s *L1RuleStage) Process(ctx context.Context, input PipelineInput) (PipelineOutput, error) {
	result, err := s.engine.Process(input.Text)
	if err != nil {
		return PipelineOutput{}, fmt.Errorf("L1 rule deidentify failed: %w", err)
	}
	if input.Metadata == nil {
		input.Metadata = make(map[string]any)
	}
	// 保存原始文本供 L2 在原始文本上做 NER（避免 L1 占位符干扰模型上下文）
	input.Metadata["original_text"] = input.Text
	input.Metadata["l1_entities"] = result.Entities
	input.Metadata["l1_placeholders"] = result.Placeholder
	return PipelineOutput{Text: result.SafeText, Metadata: input.Metadata}, nil
}

// L2NERStage 二级 NER 模型脱敏阶段。
// 基于 DistilBERT-ONNX 识别人名、地点、机构名，补充 L1 未覆盖的实体。
type L2NERStage struct {
	detector port.NERDetector
}

// NewL2NERStage 创建 L2 NER 脱敏阶段。
func NewL2NERStage(det port.NERDetector) *L2NERStage {
	return &L2NERStage{detector: det}
}

func (s *L2NERStage) Process(ctx context.Context, input PipelineInput) (PipelineOutput, error) {
	// 1. 可用性检查：NER 不可用时降级透传，不阻断流水线
	if s.detector == nil || !s.detector.IsAvailable() {
		return PipelineOutput(input), nil
	}

	// 2. 获取原始文本（L1 存入 Metadata）
	originalText, _ := input.Metadata["original_text"].(string)
	if originalText == "" {
		originalText = input.Text
	}

	// 3. 在原始文本上执行 NER 推理，避免 L1 占位符干扰模型上下文
	entities, err := s.detector.Predict(ctx, originalText)
	if err != nil {
		// 降级：推理失败时不阻断流水线，直接透传 L1 结果
		return PipelineOutput(input), nil
	}
	if len(entities) == 0 {
		return PipelineOutput(input), nil
	}

	// 4. 获取 L1 实体，过滤与 L1 区域重叠的 NER 结果（L1 优先）
	l1Entities, _ := input.Metadata["l1_entities"].([]models.SensitiveEntity)
	entities = filterOverlappingEntities(entities, l1Entities)
	if len(entities) == 0 {
		return PipelineOutput(input), nil
	}

	// 5. 偏移量映射：将原始文本中的 NER 位置映射到 L1 脱敏文本中的对应位置
	for i := range entities {
		entities[i].StartPos = mapOriginalToDeidPos(entities[i].StartPos, l1Entities)
		entities[i].EndPos = mapOriginalToDeidPos(entities[i].EndPos, l1Entities)
	}

	// 6. 按 StartPos 降序排序，从后向前替换（避免偏移量混乱）
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].StartPos > entities[j].StartPos
	})

	text := input.Text
	for i, e := range entities {
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(e.Text+fmt.Sprintf("_l2_%d", i))))[:8]
		prefix := mapTypeToPlaceholderPrefix(e.Type)
		placeholder := fmt.Sprintf("{{%s_%s}}", prefix, hash)

		// 边界安全检查
		if e.StartPos < 0 || e.EndPos > len(text) || e.StartPos >= e.EndPos {
			continue
		}
		text = text[:e.StartPos] + placeholder + text[e.EndPos:]
	}

	// 7. 将 L2 实体记录存入 Metadata，供后续阶段或日志使用
	input.Metadata["l2_entities"] = entities
	return PipelineOutput{Text: text, Metadata: input.Metadata}, nil
}

// L3KeywordStage 三级关键词字典脱敏阶段。
type L3KeywordStage struct{}

// NewL3KeywordStage 创建 L3 关键词字典脱敏阶段。
func NewL3KeywordStage() *L3KeywordStage {
	return &L3KeywordStage{}
}

func (s *L3KeywordStage) Process(ctx context.Context, input PipelineInput) (PipelineOutput, error) {
	// TODO(作者): 接入 Trie 树前缀匹配字典 [Issue#007]
	return PipelineOutput(input), nil
}

// --- 辅助函数 ---

// filterOverlappingEntities 过滤掉与 L1 实体区域重叠的 NER 结果。
// L1 规则引擎对身份证/手机号等有精确 regexp 验证，误报率低于 NER，
// 因此重叠时 L1 优先，避免同一文本被重复替换。
func filterOverlappingEntities(nerEntities, l1Entities []models.SensitiveEntity) []models.SensitiveEntity {
	if len(l1Entities) == 0 {
		return nerEntities
	}
	var filtered []models.SensitiveEntity
	for _, ne := range nerEntities {
		overlap := false
		for _, le := range l1Entities {
			if ne.StartPos < le.EndPos && ne.EndPos > le.StartPos {
				overlap = true
				break
			}
		}
		if !overlap {
			filtered = append(filtered, ne)
		}
	}
	return filtered
}

// mapOriginalToDeidPos 将原始文本中的位置映射到 L1 脱敏文本中的对应位置。
// 遍历 L1 实体（按 StartPos 升序），维护累积偏移量 delta，
// 即每个 L1 实体替换为占位符后带来的长度变化总和。
func mapOriginalToDeidPos(origPos int, l1Entities []models.SensitiveEntity) int {
	if len(l1Entities) == 0 {
		return origPos
	}
	// 按 StartPos 升序排序，确保 delta 累积正确
	sorted := make([]models.SensitiveEntity, len(l1Entities))
	copy(sorted, l1Entities)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartPos < sorted[j].StartPos
	})

	delta := 0
	for _, e := range sorted {
		if origPos <= e.StartPos {
			break
		}
		if e.Placeholder != "" {
			delta += len(e.Placeholder) - (e.EndPos - e.StartPos)
		}
	}
	return origPos + delta
}

// mapTypeToPlaceholderPrefix 将实体类型映射为占位符前缀。
func mapTypeToPlaceholderPrefix(entityType string) string {
	switch entityType {
	case "姓名":
		return "per"
	case "地点":
		return "loc"
	case "机构名":
		return "org"
	default:
		return "ent"
	}
}

// NewDefaultDeidentifyPipeline 创建默认的三级脱敏流水线（L1→L2→L3），
// 供 Wire 注入使用，避免变参接口带来的多绑定问题。
func NewDefaultDeidentifyPipeline(l1 *L1RuleStage, l2 *L2NERStage, l3 *L3KeywordStage) *DeidentifyPipeline {
	return NewDeidentifyPipeline(l1, l2, l3)
}

// PipelineSet 供 Wire 使用的 ProviderSet。
var PipelineSet = wire.NewSet(
	NewDefaultDeidentifyPipeline,
	NewL1RuleStage,
	NewL2NERStage,
	NewL3KeywordStage,
)
