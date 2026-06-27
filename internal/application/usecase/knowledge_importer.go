package usecase

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
)

// KnowledgeImporter 编排知识库文件导入流程。
type KnowledgeImporter struct {
	repo         repository.KnowledgeRepository
	chunker      *KnowledgeChunker
	tokenizer    *KnowledgeTokenizer
	embeddingSvc port.EmbeddingService
}

// NewKnowledgeImporter 构造函数。
func NewKnowledgeImporter(repo repository.KnowledgeRepository, chunker *KnowledgeChunker, tokenizer *KnowledgeTokenizer, embeddingSvc port.EmbeddingService) *KnowledgeImporter {
	return &KnowledgeImporter{repo: repo, chunker: chunker, tokenizer: tokenizer, embeddingSvc: embeddingSvc}
}

// ImportFile 导入单个知识文件。
// 若 checksum 已存在且 force=false，则返回现有文档对应的任务作为幂等结果。
func (i *KnowledgeImporter) ImportFile(ctx context.Context, filePath string, content []byte, force bool) (*entity.KnowledgeImportJob, error) {
	checksum := fmt.Sprintf("%x", sha256.Sum256(content))

	// 幂等：checksum 已存在且非强制时直接返回旧任务
	if !force {
		existing, err := i.repo.FindDocumentByChecksum(ctx, checksum)
		if err == nil && existing != nil {
			job := &entity.KnowledgeImportJob{
				JobID:     uuid.New().String(),
				Status:    entity.KnowledgeImportIndexed,
				Total:     1,
				Processed: 1,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			_ = i.repo.SaveImportJob(ctx, job)
			return job, nil
		}
	}

	job := &entity.KnowledgeImportJob{
		JobID:     uuid.New().String(),
		Status:    entity.KnowledgeImportPending,
		Total:     1,
		Processed: 0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := i.repo.SaveImportJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create import job: %w", err)
	}

	doc, chunks, err := i.parseAndChunk(filePath, content)
	if err != nil {
		job.Status = entity.KnowledgeImportError
		job.ErrorMessage = err.Error()
		job.UpdatedAt = time.Now()
		_ = i.repo.SaveImportJob(ctx, job)
		return job, fmt.Errorf("failed to parse knowledge file: %w", err)
	}

	doc.Checksum = checksum
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()
	if err := i.repo.SaveDocument(ctx, doc); err != nil {
		job.Status = entity.KnowledgeImportError
		job.ErrorMessage = err.Error()
		job.UpdatedAt = time.Now()
		_ = i.repo.SaveImportJob(ctx, job)
		return job, fmt.Errorf("failed to save knowledge document: %w", err)
	}

	// 填充 document_id 到 chunks
	for _, c := range chunks {
		c.DocumentID = doc.DocumentID
		c.ChunkID = fmt.Sprintf("%s-%d", doc.DocumentID, c.ChunkIndex)
		c.CreatedAt = time.Now()
	}
	if err := i.repo.SaveChunks(ctx, chunks); err != nil {
		job.Status = entity.KnowledgeImportError
		job.ErrorMessage = err.Error()
		job.UpdatedAt = time.Now()
		_ = i.repo.SaveImportJob(ctx, job)
		return job, fmt.Errorf("failed to save knowledge chunks: %w", err)
	}

	// 为每个片段建立关键词索引
	for _, c := range chunks {
		termFreq := i.tokenizer.Tokenize(c.Content)
		if err := i.repo.SaveTerms(ctx, c.ChunkID, doc.DocumentID, termFreq); err != nil {
			job.Status = entity.KnowledgeImportError
			job.ErrorMessage = err.Error()
			job.UpdatedAt = time.Now()
			_ = i.repo.SaveImportJob(ctx, job)
			return job, fmt.Errorf("failed to save knowledge terms: %w", err)
		}
	}

	// 可选：生成向量嵌入
	status := entity.KnowledgeImportIndexed
	if i.embeddingSvc != nil && i.embeddingSvc.IsAvailable() {
		texts := make([]string, len(chunks))
		for idx, c := range chunks {
			texts[idx] = c.Content
		}
		vectors, err := i.embeddingSvc.Embed(ctx, texts)
		if err != nil {
			status = entity.KnowledgeImportVectorUnavailable
		} else {
			for idx, c := range chunks {
				if idx < len(vectors) {
					vector := vectors[idx]
					if err := i.repo.SaveEmbedding(ctx, c.ChunkID, i.embeddingSvc.ModelVersion(), len(vector), vector); err != nil {
						status = entity.KnowledgeImportVectorUnavailable
						break
					}
				}
			}
		}
	}

	job.Status = status
	job.Processed = 1
	job.Total = len(chunks)
	job.UpdatedAt = time.Now()
	if err := i.repo.SaveImportJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to save indexed import job: %w", err)
	}
	return job, nil
}

// parseAndChunk 根据扩展名解析文件并切分。
func (i *KnowledgeImporter) parseAndChunk(filePath string, content []byte) (*entity.KnowledgeDocument, []*entity.KnowledgeChunk, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	base := filepath.Base(filePath)
	title := strings.TrimSuffix(base, filepath.Ext(base))

	switch ext {
	case ".md", ".markdown":
		chunks := i.chunker.ChunkMarkdown(title, content)
		return &entity.KnowledgeDocument{
			DocumentID: uuid.New().String(),
			Title:      title,
			SourceType: entity.KnowledgeSourceMarkdown,
			Citation:   base,
			Language:   detectLanguage(string(content)),
		}, chunks, nil
	case ".jsonl":
		chunks, err := i.chunker.ChunkJSONL(content)
		if err != nil {
			return nil, nil, err
		}
		return &entity.KnowledgeDocument{
			DocumentID: uuid.New().String(),
			Title:      title,
			SourceType: entity.KnowledgeSourceJSONL,
			Citation:   base,
			Language:   detectLanguage(string(content)),
		}, chunks, nil
	default:
		return nil, nil, fmt.Errorf("unsupported knowledge file format: %s", ext)
	}
}

// detectLanguage 简单根据字符判断语言。
func detectLanguage(s string) string {
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			return "zh"
		}
	}
	return "en"
}
