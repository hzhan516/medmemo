package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
)

// stubFactRepository 用于 WailsApp 测试的简单 stub
type wailsStubFactRepo struct {
	facts       map[string]*entity.ExtractedFact
	pendingList []*entity.ExtractedFact
}

func (s *wailsStubFactRepo) Save(ctx context.Context, f *entity.ExtractedFact) error { return nil }
func (s *wailsStubFactRepo) GetByID(ctx context.Context, factID string) (*entity.ExtractedFact, error) {
	f, ok := s.facts[factID]
	if !ok {
		return nil, entity.ErrFactNotFound
	}
	return f, nil
}
func (s *wailsStubFactRepo) ListByStatus(ctx context.Context, status entity.FactStatus, offset, limit int) ([]*entity.ExtractedFact, error) {
	var result []*entity.ExtractedFact
	for _, f := range s.facts {
		if f.Status == status {
			result = append(result, f)
		}
	}
	if offset >= len(result) {
		return nil, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}
func (s *wailsStubFactRepo) ListPending(ctx context.Context, offset, limit int) ([]*entity.ExtractedFact, error) {
	if offset >= len(s.pendingList) {
		return nil, nil
	}
	end := offset + limit
	if end > len(s.pendingList) {
		end = len(s.pendingList)
	}
	return s.pendingList[offset:end], nil
}
func (s *wailsStubFactRepo) UpdateStatus(ctx context.Context, factID string, status entity.FactStatus) error {
	f, ok := s.facts[factID]
	if !ok {
		return entity.ErrFactNotFound
	}
	f.Status = status
	return nil
}
func (s *wailsStubFactRepo) Delete(ctx context.Context, factID string) error {
	delete(s.facts, factID)
	return nil
}
func (s *wailsStubFactRepo) GetStats(ctx context.Context) (total, approved, rejected, pending int64, err error) {
	return 0, 0, 0, 0, nil
}

var _ repository.FactRepository = (*wailsStubFactRepo)(nil)

func setupMemoryWailsApp() *WailsApp {
	now := time.Now().UTC()
	facts := map[string]*entity.ExtractedFact{
		"fact_1": {
			FactID: "fact_1", Subject: "用户", Predicate: "患有", Object: "高血压",
			Confidence: 0.9, Status: entity.FactStatusApproved, CreatedAt: now,
		},
		"fact_2": {
			FactID: "fact_2", Subject: "用户", Predicate: "服用", Object: "降压药",
			Confidence: 0.85, Status: entity.FactStatusApproved, CreatedAt: now,
		},
		"fact_3": {
			FactID: "fact_3", Subject: "用户", Predicate: "感觉", Object: "头晕",
			Confidence: 0.7, Status: entity.FactStatusPending, CreatedAt: now,
		},
	}
	pending := []*entity.ExtractedFact{facts["fact_3"]}

	return &WailsApp{
		factRepo: &wailsStubFactRepo{facts: facts, pendingList: pending},
	}
}

func TestWailsApp_GetMemories(t *testing.T) {
	app := setupMemoryWailsApp()
	memories, err := app.GetMemories(10, 0)
	require.NoError(t, err)
	require.Len(t, memories, 2) // 只返回 approved
	assert.Equal(t, "用户", memories[0].Subject)
}

func TestWailsApp_GetMemories_Pagination(t *testing.T) {
	app := setupMemoryWailsApp()
	memories, err := app.GetMemories(1, 0)
	require.NoError(t, err)
	require.Len(t, memories, 1)

	memories, err = app.GetMemories(1, 1)
	require.NoError(t, err)
	require.Len(t, memories, 1)

	memories, err = app.GetMemories(1, 2)
	require.NoError(t, err)
	require.Len(t, memories, 0)
}

func TestWailsApp_GetMemoryByID(t *testing.T) {
	app := setupMemoryWailsApp()
	mem, err := app.GetMemoryByID("fact_1")
	require.NoError(t, err)
	assert.Equal(t, "用户", mem.Subject)
	assert.Equal(t, "患有", mem.Predicate)
	assert.Equal(t, "高血压", mem.Object)

	_, err = app.GetMemoryByID("nonexistent")
	assert.Error(t, err)
}

func TestWailsApp_DeleteMemory(t *testing.T) {
	app := setupMemoryWailsApp()
	err := app.DeleteMemory("fact_1")
	require.NoError(t, err)

	_, err = app.GetMemoryByID("fact_1")
	assert.Error(t, err)
}

func TestWailsApp_SearchMemories(t *testing.T) {
	app := setupMemoryWailsApp()
	results, err := app.SearchMemories("高血压")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "高血压", results[0].Object)

	results, err = app.SearchMemories("用药")
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestWailsApp_GetPendingReviews(t *testing.T) {
	app := setupMemoryWailsApp()
	pending, err := app.GetPendingReviews(10, 0)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, "头晕", pending[0].Object)
}

func TestWailsApp_ApproveFact(t *testing.T) {
	app := setupMemoryWailsApp()
	err := app.ApproveFact("fact_3")
	require.NoError(t, err)

	mem, err := app.GetMemoryByID("fact_3")
	require.NoError(t, err)
	assert.Equal(t, string(entity.FactStatusApproved), mem.Status)
}

func TestWailsApp_RejectFact(t *testing.T) {
	app := setupMemoryWailsApp()
	err := app.RejectFact("fact_3")
	require.NoError(t, err)

	mem, err := app.GetMemoryByID("fact_3")
	require.NoError(t, err)
	assert.Equal(t, string(entity.FactStatusRejected), mem.Status)
}

func TestWailsApp_ApproveFact_NotFound(t *testing.T) {
	app := setupMemoryWailsApp()
	err := app.ApproveFact("nonexistent")
	assert.Error(t, err)
}
