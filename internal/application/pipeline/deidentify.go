// Package pipeline 实现脱敏流水线的应用层编排。
// 将 L1 规则引擎、L2 NER 模型、L3 关键词字典串联为完整处理管道。
package pipeline

import (
	"context"
	"fmt"

	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/pkg/models"
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
func (p *DeidentifyPipeline) Execute(ctx context.Context, raw string) (models.DeidentifyResult, error) {
	input := PipelineInput{Text: raw, Metadata: make(map[string]any)}
	for _, stage := range p.stages {
		output, err := stage.Process(ctx, input)
		if err != nil {
			return models.DeidentifyResult{}, fmt.Errorf("deidentify stage %T failed: %w", stage, err)
		}
		input = PipelineInput{Text: output.Text, Metadata: output.Metadata}
	}
	return models.DeidentifyResult{
		OriginalText: raw,
		SafeText:     input.Text,
	}, nil
}

// L1RuleStage 一级规则脱敏阶段。
type L1RuleStage struct{}

func (s *L1RuleStage) Process(ctx context.Context, input PipelineInput) (PipelineOutput, error) {
	// TODO(作者): 接入 pkg/desensitizer 规则引擎 [Issue#005]
	return PipelineOutput{Text: input.Text, Metadata: input.Metadata}, nil
}

// L2NERStage 二级 NER 模型脱敏阶段。
type L2NERStage struct {
	detector port.SensitiveDetector
}

func NewL2NERStage(det port.SensitiveDetector) *L2NERStage {
	return &L2NERStage{detector: det}
}

func (s *L2NERStage) Process(ctx context.Context, input PipelineInput) (PipelineOutput, error) {
	// TODO(作者): 调用 ONNX NER 模型检测并替换实体 [Issue#006]
	return PipelineOutput{Text: input.Text, Metadata: input.Metadata}, nil
}

// L3KeywordStage 三级关键词字典脱敏阶段。
type L3KeywordStage struct{}

func (s *L3KeywordStage) Process(ctx context.Context, input PipelineInput) (PipelineOutput, error) {
	// TODO(作者): 接入 Trie 树前缀匹配字典 [Issue#007]
	return PipelineOutput{Text: input.Text, Metadata: input.Metadata}, nil
}

// PipelineSet 供 Wire 使用的 ProviderSet。
var PipelineSet = wire.NewSet(
	NewDeidentifyPipeline,
	NewL2NERStage,
)
