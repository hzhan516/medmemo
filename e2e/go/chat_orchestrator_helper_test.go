//go:build e2e

package e2e

import (
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/application/usecase"
)

// newTestChatOrchestrator 创建用于 E2E 测试的 ChatOrchestrator，屏蔽 Deps 结构体细节。
func newTestChatOrchestrator(client port.LLMClient, checker usecase.ComplianceChecker, deid usecase.Deidentifier) *usecase.ChatOrchestrator {
	return usecase.NewChatOrchestrator(usecase.ChatOrchestratorDeps{
		LLMFactory:           &mockLLMClientFactory{client: client},
		ProviderStore:        newMockProviderStore(),
		MemoryRepo:           &mockMemoryRepository{},
		Detector:             &mockSensitiveDetector{},
		Compliance:           checker,
		DeidPipeline:         deid,
		MemoryRetriever:      &mockMemoryQuerier{},
		ConfidenceAggregator: usecase.NewConfidenceAggregator(),
		FactRepo:             &mockFactRepository{},
		IntentResolver:       usecase.NewIntentResolver(usecase.NewQueryExpansionService()),
		LocalAnswer:          usecase.NewLocalAnswerService(usecase.NewLocalAnswerConfig()),
	})
}
