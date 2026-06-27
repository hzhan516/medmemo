package repository

import (
	"context"
	"testing"

	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestKnowledgeRepo(t *testing.T) (*KnowledgeRepoSQLite, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	store := &accuracyMockSecretStore{data: make(map[string][]byte)}
	conn, err := database.NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, conn.Migrate(ctx))

	repo := NewKnowledgeRepoSQLite(conn)
	return repo, func() { _ = conn.Close() }
}

func TestKnowledgeRepoSQLite_ImportMarkdown(t *testing.T) {
	repo, cleanup := newTestKnowledgeRepo(t)
	defer cleanup()

	ctx := context.Background()
	chunker := usecase.NewKnowledgeChunker(512, 32)
	importer := usecase.NewKnowledgeImporter(repo, chunker)

	content := []byte("# 健康指南\n\n感冒通常由病毒引起。")
	job, err := importer.ImportFile(ctx, "guide.md", content, false)
	require.NoError(t, err)
	assert.Equal(t, entity.KnowledgeImportIndexed, job.Status)

	docs, err := repo.ListDocuments(ctx)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "guide", docs[0].Title)

	count, err := repo.CountChunks(ctx)
	require.NoError(t, err)
	assert.Greater(t, count, 0)
}

func TestKnowledgeRepoSQLite_ImportSameFileTwice(t *testing.T) {
	repo, cleanup := newTestKnowledgeRepo(t)
	defer cleanup()

	ctx := context.Background()
	chunker := usecase.NewKnowledgeChunker(512, 32)
	importer := usecase.NewKnowledgeImporter(repo, chunker)

	content := []byte("# 健康指南\n")
	job1, err := importer.ImportFile(ctx, "guide.md", content, false)
	require.NoError(t, err)
	job2, err := importer.ImportFile(ctx, "guide.md", content, false)
	require.NoError(t, err)

	assert.Equal(t, entity.KnowledgeImportIndexed, job1.Status)
	assert.Equal(t, entity.KnowledgeImportIndexed, job2.Status)

	docs, err := repo.ListDocuments(ctx)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
}

func TestKnowledgeRepoSQLite_DeleteDocument(t *testing.T) {
	repo, cleanup := newTestKnowledgeRepo(t)
	defer cleanup()

	ctx := context.Background()
	chunker := usecase.NewKnowledgeChunker(512, 32)
	importer := usecase.NewKnowledgeImporter(repo, chunker)

	content := []byte("# 健康指南\n\n感冒通常由病毒引起。")
	job, err := importer.ImportFile(ctx, "guide.md", content, false)
	require.NoError(t, err)
	require.Equal(t, entity.KnowledgeImportIndexed, job.Status)

	docs, err := repo.ListDocuments(ctx)
	require.NoError(t, err)
	require.Len(t, docs, 1)

	require.NoError(t, repo.DeleteDocument(ctx, docs[0].DocumentID))

	docs, err = repo.ListDocuments(ctx)
	require.NoError(t, err)
	assert.Len(t, docs, 0)

	count, err := repo.CountChunks(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
