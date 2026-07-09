package repository

import (
	"context"
	"testing"

	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSecretStore 实现 secret.Store 接口，用于测试。
type accuracyMockSecretStore struct {
	data map[string][]byte
}

func (m *accuracyMockSecretStore) Get(key string) ([]byte, error) {
	if v, ok := m.data[key]; ok {
		return v, nil
	}
	return nil, assert.AnError
}

func (m *accuracyMockSecretStore) Set(key string, value []byte) error {
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[key] = value
	return nil
}

func (m *accuracyMockSecretStore) Delete(_ string) error {
	return nil
}

func newTestAccuracyRepo(t *testing.T) (*AccuracyRepoSQLite, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	store := &accuracyMockSecretStore{data: make(map[string][]byte)}
	conn, err := database.NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, conn.Migrate(ctx))

	repo := NewAccuracyRepoSQLite(conn)
	return repo, func() { _ = conn.Close() }
}

func TestAccuracyRepoSQLite_GetAccuracy_EmptyReturnsDefault(t *testing.T) {
	repo, cleanup := newTestAccuracyRepo(t)
	defer cleanup()

	ctx := context.Background()
	acc, err := repo.GetAccuracy(ctx, "symptom_analysis")
	require.NoError(t, err)
	assert.InDelta(t, 0.75, acc, 0.0001)
}

func TestAccuracyRepoSQLite_RecordFeedback_RecomputesStats(t *testing.T) {
	repo, cleanup := newTestAccuracyRepo(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, repo.RecordFeedback(ctx, "msg-1", "symptom_analysis", "helpful"))
	require.NoError(t, repo.RecordFeedback(ctx, "msg-2", "symptom_analysis", "helpful"))
	require.NoError(t, repo.RecordFeedback(ctx, "msg-3", "symptom_analysis", "inaccurate"))

	acc, err := repo.GetAccuracy(ctx, "symptom_analysis")
	require.NoError(t, err)
	assert.InDelta(t, 2.0/3.0, acc, 0.0001)
}

func TestAccuracyRepoSQLite_RecordFeedback_IdempotentUpdate(t *testing.T) {
	repo, cleanup := newTestAccuracyRepo(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, repo.RecordFeedback(ctx, "msg-1", "symptom_analysis", "helpful"))
	require.NoError(t, repo.RecordFeedback(ctx, "msg-1", "symptom_analysis", "inaccurate"))

	acc, err := repo.GetAccuracy(ctx, "symptom_analysis")
	require.NoError(t, err)
	assert.InDelta(t, 0.0, acc, 0.0001)
}

func TestAccuracyRepoSQLite_RestartPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	store := &accuracyMockSecretStore{data: make(map[string][]byte)}

	ctx := context.Background()

	// 第一次打开并写入反馈
	conn1, err := database.NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	require.NoError(t, conn1.Migrate(ctx))
	repo1 := NewAccuracyRepoSQLite(conn1)
	require.NoError(t, repo1.RecordFeedback(ctx, "msg-1", "health_info", "helpful"))
	require.NoError(t, conn1.Close())

	// 重新打开，验证数据仍在
	conn2, err := database.NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	require.NoError(t, conn2.Migrate(ctx))
	repo2 := NewAccuracyRepoSQLite(conn2)
	defer func() { _ = conn2.Close() }()

	acc, err := repo2.GetAccuracy(ctx, "health_info")
	require.NoError(t, err)
	assert.InDelta(t, 1.0, acc, 0.0001)
}
