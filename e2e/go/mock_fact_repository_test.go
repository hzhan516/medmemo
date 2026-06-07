//go:build e2e

package e2e

import (
	"context"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
)

type mockFactRepository struct{}

func (m *mockFactRepository) Save(ctx context.Context, f *entity.ExtractedFact) error { return nil }
func (m *mockFactRepository) GetByID(ctx context.Context, factID string) (*entity.ExtractedFact, error) {
	return nil, entity.ErrFactNotFound
}
func (m *mockFactRepository) ListByStatus(ctx context.Context, status entity.FactStatus, offset, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepository) ListPending(ctx context.Context, offset, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepository) UpdateStatus(ctx context.Context, factID string, status entity.FactStatus) error {
	return nil
}
func (m *mockFactRepository) Delete(ctx context.Context, factID string) error { return nil }
func (m *mockFactRepository) GetStats(ctx context.Context) (total, approved, rejected, pending int64, err error) {
	return 0, 0, 0, 0, nil
}
func (m *mockFactRepository) ListAllSubjects(ctx context.Context) ([]string, error) { return nil, nil }
func (m *mockFactRepository) FindBySubject(ctx context.Context, subject string) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepository) FindBySession(ctx context.Context, sessionID string) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepository) FindLatestApprovedByPredicates(ctx context.Context, subject string, predicates []string) (*entity.ExtractedFact, error) {
	return nil, entity.ErrFactNotFound
}

var _ repository.FactRepository = (*mockFactRepository)(nil)
