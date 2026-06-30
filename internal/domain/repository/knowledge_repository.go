package repository

import (
	"context"

	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// KnowledgeRepository 定义知识库持久化接口。
type KnowledgeRepository interface {
	// SaveDocument 保存或更新文档（基于 checksum 去重）。
	SaveDocument(ctx context.Context, doc *entity.KnowledgeDocument) error
	// SaveChunks 批量保存或更新文档片段。
	SaveChunks(ctx context.Context, chunks []*entity.KnowledgeChunk) error
	// FindDocument 按 ID 查询文档。
	FindDocument(ctx context.Context, id string) (*entity.KnowledgeDocument, error)
	// ListDocuments 列出全部文档。
	ListDocuments(ctx context.Context) ([]*entity.KnowledgeDocument, error)
	// DeleteDocument 删除文档及其关联的片段、词项、向量。
	DeleteDocument(ctx context.Context, id string) error
	// FindDocumentByChecksum 按 checksum 查询文档，用于去重。
	FindDocumentByChecksum(ctx context.Context, checksum string) (*entity.KnowledgeDocument, error)
	// CountChunks 返回知识片段总数。
	CountChunks(ctx context.Context) (int, error)
	// CountTermDF 返回某词项的文档频率（出现该词的 distinct chunk 数）。
	CountTermDF(ctx context.Context, term string) (int, error)
	// AverageChunkTokenCount 返回所有片段的平均 token 数。
	AverageChunkTokenCount(ctx context.Context) (float64, error)
	// GetChunkTokenCount 返回指定片段的 token_count。
	GetChunkTokenCount(ctx context.Context, chunkID string) (int, error)
	// SaveTerms 保存片段的词项频率。
	SaveTerms(ctx context.Context, chunkID, documentID string, termFreq map[string]int) error
	// SearchKeyword 基于 BM25-like 评分执行关键词检索，返回按得分降序排列的结果。
	SearchKeyword(ctx context.Context, terms []string, limit int) ([]*KnowledgeSearchResult, error)
	// SaveEmbedding 保存片段向量。
	SaveEmbedding(ctx context.Context, chunkID, modelVersion string, dimension int, embedding []float32) error
	// SearchVector 基于向量相似度检索。
	SearchVector(ctx context.Context, embedding []float32, limit int) ([]*KnowledgeSearchResult, error)
	// SaveImportJob 保存导入任务。
	SaveImportJob(ctx context.Context, job *entity.KnowledgeImportJob) error
	// GetImportJob 按 ID 查询导入任务。
	GetImportJob(ctx context.Context, id string) (*entity.KnowledgeImportJob, error)
}

// KnowledgeSearchResult 表示一次知识检索结果。
type KnowledgeSearchResult struct {
	ChunkID    string
	DocumentID string
	Content    string
	Score      float64
	SourceType string // keyword / vector / hybrid
}
