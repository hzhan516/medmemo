// Package pipeline 实现脱敏流水线的应用层编排。
// 将 L1 规则引擎、L2 NER 模型串联为完整处理管道。
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
	Process(ctx context.Context, input Input) (Output, error)
}

// Input / Output 管道数据单元。
type Input struct {
	Text     string
	Metadata map[string]any
	// Level 为本次脱敏的分级策略，贯穿各阶段，供严格级专属阶段判定是否激活。
	Level models.DesensitizationLevel
}

type Output struct {
	Text     string
	Metadata map[string]any
}

// NewDeidentifyPipeline 创建流水线，顺序注入各阶段。
func NewDeidentifyPipeline(stages ...DeidentifyStage) *DeidentifyPipeline {
	return &DeidentifyPipeline{stages: stages}
}

// Execute 顺序执行各阶段，任一阶段出错即短路返回。
// 收集各阶段产生的实体和 P2 级占位符映射，供输出还原使用。
// level 贯穿各阶段（Output 不携带 Level，循环内以传入的 level 重建 Input）。
func (p *DeidentifyPipeline) Execute(ctx context.Context, raw string, level models.DesensitizationLevel) (models.DeidentifyResult, error) {
	input := Input{Text: raw, Level: level, Metadata: make(map[string]any)}
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

		// Output 不携带 Level，用传入的 level 重建 Input 以贯穿后续阶段。
		input = Input{Text: output.Text, Metadata: output.Metadata, Level: level}
	}
	localRestore := make(map[string]string)
	for _, e := range allEntities {
		if e.Placeholder != "" {
			localRestore[e.Placeholder] = e.Text
		}
	}

	return models.DeidentifyResult{
		OriginalText: raw,
		SafeText:     input.Text,
		Entities:     allEntities,
		Placeholder:  allPlaceholders,
		LocalRestore: localRestore,
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

func (s *L1RuleStage) Process(_ context.Context, input Input) (Output, error) {
	result, err := s.engine.Process(input.Text)
	if err != nil {
		return Output{}, fmt.Errorf("l1 rule deidentify failed: %w", err)
	}
	if input.Metadata == nil {
		input.Metadata = make(map[string]any)
	}
	// 保存原始文本供 L2 在原始文本上做 NER（避免 L1 占位符干扰模型上下文）
	input.Metadata["original_text"] = input.Text
	input.Metadata["l1_entities"] = result.Entities
	input.Metadata["l1_placeholders"] = result.Placeholder
	return Output{Text: result.SafeText, Metadata: input.Metadata}, nil
}

// passthroughOutput 将 Input 原样透传为 Output（丢弃仅用于阶段间的 Level 字段）。
func passthroughOutput(input Input) Output {
	return Output{Text: input.Text, Metadata: input.Metadata}
}

// L2NERStage 二级 NER 模型脱敏阶段。
// 基于 DistilBERT-ONNX 识别人名、地点、机构名，补充 L1 未覆盖的实体。
// 持有标准级与严格级两个检测器，按 input.Level 选择：严格级使用更低置信度阈值以提升召回。
type L2NERStage struct {
	standard port.NERDetector
	strict   port.NERDetector
}

// NewL2NERStage 创建 L2 NER 脱敏阶段。
// standard 用于标准级（默认阈值），strict 用于严格级（更低阈值、更高召回）。
func NewL2NERStage(standard port.StandardNERDetector, strict port.StrictNERDetector) *L2NERStage {
	return &L2NERStage{standard: standard, strict: strict}
}

// detectorForLevel 按脱敏级别选择 NER 检测器。
func (s *L2NERStage) detectorForLevel(level models.DesensitizationLevel) port.NERDetector {
	if level == models.DesensitizationStrict {
		return s.strict
	}
	return s.standard
}

func (s *L2NERStage) Process(ctx context.Context, input Input) (Output, error) {
	detector := s.detectorForLevel(input.Level)

	// 1. 可用性检查：NER 不可用时降级透传，不阻断流水线
	if detector == nil || !detector.IsAvailable() {
		return passthroughOutput(input), nil
	}

	// 2. 获取原始文本（L1 存入 Metadata）
	originalText, _ := input.Metadata["original_text"].(string)
	if originalText == "" {
		originalText = input.Text
	}

	// 3. 在原始文本上执行 NER 推理，避免 L1 占位符干扰模型上下文
	entities, err := detector.Predict(ctx, originalText)
	if err != nil {
		// 降级：推理失败时不阻断流水线，直接透传 L1 结果
		return passthroughOutput(input), nil
	}
	if len(entities) == 0 {
		return passthroughOutput(input), nil
	}

	// 4. 获取 L1 实体，过滤与 L1 区域重叠的 NER 结果（L1 优先）
	l1Entities, _ := input.Metadata["l1_entities"].([]models.SensitiveEntity)
	entities = filterOverlappingEntities(entities, l1Entities)
	if len(entities) == 0 {
		return passthroughOutput(input), nil
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
		entities[i].Placeholder = placeholder
		text = text[:e.StartPos] + placeholder + text[e.EndPos:]
	}

	// 7. 将 L2 实体记录存入 Metadata，供后续阶段或日志使用
	input.Metadata["l2_entities"] = entities
	return Output{Text: text, Metadata: input.Metadata}, nil
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

// NewDefaultDeidentifyPipeline 创建默认的脱敏流水线，
// 顺序为 L1 规则 → L2 NER → L1.5 严格兜底（仅严格级激活），
// 供 Wire 注入使用，避免变参接口带来的多绑定问题。
// L1.5 置于最后：运行在 L1+L2 已处理的文本上，对残留可标识信息做兜底遮蔽，
// 从而不干扰 L2 基于原始文本的偏移映射逻辑。
func NewDefaultDeidentifyPipeline(l1 *L1RuleStage, l2 *L2NERStage, l1ext *L1ExtendedRuleStage) *DeidentifyPipeline {
	return NewDeidentifyPipeline(l1, l2, l1ext)
}

// Set 供 Wire 使用的 ProviderSet。
var Set = wire.NewSet(
	NewDefaultDeidentifyPipeline,
	NewL1RuleStage,
	NewL2NERStage,
	NewL1ExtendedRuleStage,
)
