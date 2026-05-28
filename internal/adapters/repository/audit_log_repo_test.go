package repository

import (
	"context"
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuditLogTestDB(t *testing.T) (*AuditLogRepoSQLite, func()) {
	tmpDir := t.TempDir()
	connector, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	err = connector.Migrate(ctx)
	require.NoError(t, err)

	repo := NewAuditLogRepoSQLite(connector)
	cleanup := func() {
		connector.Close()
	}
	return repo, cleanup
}

func TestAuditLogRepo_SaveAndList(t *testing.T) {
	repo, cleanup := setupAuditLogTestDB(t)
	defer cleanup()
	ctx := context.Background()

	entry := entity.NewAuditLogEntry(entity.AuditActionApprove, "fact", "fact_001", "user")
	err := repo.Save(ctx, entry)
	require.NoError(t, err)

	logs, err := repo.ListByTarget(ctx, "fact", "fact_001", 10)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, entity.AuditActionApprove, logs[0].Action)
	assert.Equal(t, "fact_001", logs[0].TargetID)
	assert.Equal(t, "user", logs[0].Actor)
}

func TestAuditLogRepo_ListByTarget_Empty(t *testing.T) {
	repo, cleanup := setupAuditLogTestDB(t)
	defer cleanup()
	ctx := context.Background()

	logs, err := repo.ListByTarget(ctx, "fact", "nonexistent", 10)
	require.NoError(t, err)
	assert.Len(t, logs, 0)
}

func TestAuditLogRepo_MultipleActions(t *testing.T) {
	repo, cleanup := setupAuditLogTestDB(t)
	defer cleanup()
	ctx := context.Background()

	actions := []entity.AuditAction{
		entity.AuditActionCreate,
		entity.AuditActionApprove,
		entity.AuditActionDelete,
	}
	for i, action := range actions {
		entry := entity.NewAuditLogEntry(action, "fact", "fact_multi", "user")
		entry.ID = "audit_" + string(rune('0'+i))
		err := repo.Save(ctx, entry)
		require.NoError(t, err)
	}

	logs, err := repo.ListByTarget(ctx, "fact", "fact_multi", 10)
	require.NoError(t, err)
	assert.Len(t, logs, 3)
}

func TestAuditLogRepo_Limit(t *testing.T) {
	repo, cleanup := setupAuditLogTestDB(t)
	defer cleanup()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		entry := entity.NewAuditLogEntry(entity.AuditActionApprove, "fact", "fact_limit", "user")
		entry.ID = "audit_" + string(rune('0'+i))
		err := repo.Save(ctx, entry)
		require.NoError(t, err)
	}

	logs, err := repo.ListByTarget(ctx, "fact", "fact_limit", 2)
	require.NoError(t, err)
	assert.Len(t, logs, 2)
}
