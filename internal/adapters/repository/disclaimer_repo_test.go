package repository

import (
	"context"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDisclaimerTestDB(t *testing.T) (*DisclaimerRepoSQLite, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	connector, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	err = connector.Migrate(ctx)
	require.NoError(t, err)

	repo := NewDisclaimerRepoSQLite(connector)
	cleanup := func() {
		connector.Close()
	}
	return repo, cleanup
}

func TestDisclaimerRepo_GetAcceptance_Empty(t *testing.T) {
	repo, cleanup := setupDisclaimerTestDB(t)
	defer cleanup()
	ctx := context.Background()

	rec, err := repo.GetAcceptance(ctx)
	require.NoError(t, err)
	assert.Nil(t, rec)
}

func TestDisclaimerRepo_SaveAndGet(t *testing.T) {
	repo, cleanup := setupDisclaimerTestDB(t)
	defer cleanup()
	ctx := context.Background()

	record := &entity.DisclaimerAcceptance{
		Version:    "1.0",
		AcceptedAt: time.Now().UTC(),
		TextHash:   "abc123",
	}
	err := repo.SaveAcceptance(ctx, record)
	require.NoError(t, err)

	got, err := repo.GetAcceptance(ctx)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "1.0", got.Version)
	assert.Equal(t, "abc123", got.TextHash)
}
