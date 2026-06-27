package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// KnowledgeDocumentDTO 知识库文档列表项。
type KnowledgeDocumentDTO struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Source    string `json:"source"`
	Citation  string `json:"citation"`
	URL       string `json:"url"`
	Language  string `json:"language"`
	Checksum  string `json:"checksum"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// ImportKnowledgeResponse 知识导入响应。
type ImportKnowledgeResponse struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Total   int    `json:"total"`
	Processed int  `json:"processed"`
	Error   string `json:"error,omitempty"`
}

// SelectKnowledgeFile 打开系统文件选择对话框，返回所选文件路径。
// 前端不直接操作本地路径，由后端统一处理。
func (a *WailsApp) SelectKnowledgeFile() (string, error) {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择知识库文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "Markdown", Pattern: "*.md;*.markdown"},
			{DisplayName: "JSON Lines", Pattern: "*.jsonl"},
			{DisplayName: "所有文件", Pattern: "*"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to open file dialog: %w", err)
	}
	if selection == "" {
		return "", fmt.Errorf("no file selected")
	}
	return selection, nil
}

// ImportKnowledgeFile 导入指定路径的知识库文件。
func (a *WailsApp) ImportKnowledgeFile(filePath string) (*ImportKnowledgeResponse, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read knowledge file: %w", err)
	}

	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Minute)
	defer cancel()

	job, err := a.knowledgeImporter.ImportFile(ctx, filePath, content, false)
	if err != nil {
		return nil, fmt.Errorf("failed to import knowledge file: %w", err)
	}

	return &ImportKnowledgeResponse{
		JobID:     job.JobID,
		Status:    string(job.Status),
		Total:     job.Total,
		Processed: job.Processed,
		Error:     job.ErrorMessage,
	}, nil
}

// ListKnowledgeDocuments 列出所有已导入的知识库文档。
func (a *WailsApp) ListKnowledgeDocuments() ([]KnowledgeDocumentDTO, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	docs, err := a.knowledgeRepo.ListDocuments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list knowledge documents: %w", err)
	}

	result := make([]KnowledgeDocumentDTO, 0, len(docs))
	for _, d := range docs {
		result = append(result, KnowledgeDocumentDTO{
			ID:        d.DocumentID,
			Title:     d.Title,
			Source:    string(d.SourceType),
			Citation:  d.Citation,
			URL:       d.URL,
			Language:  d.Language,
			Checksum:  d.Checksum,
			CreatedAt: d.CreatedAt.UnixMilli(),
			UpdatedAt: d.UpdatedAt.UnixMilli(),
		})
	}
	return result, nil
}

// DeleteKnowledgeDocument 删除指定知识库文档及其索引。
func (a *WailsApp) DeleteKnowledgeDocument(id string) error {
	if id == "" {
		return fmt.Errorf("document id is required")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	if err := a.knowledgeRepo.DeleteDocument(ctx, id); err != nil {
		return fmt.Errorf("failed to delete knowledge document: %w", err)
	}
	return nil
}

// GetKnowledgeImportJob 查询导入任务状态。
func (a *WailsApp) GetKnowledgeImportJob(jobID string) (*ImportKnowledgeResponse, error) {
	if jobID == "" {
		return nil, fmt.Errorf("job id is required")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	job, err := a.knowledgeRepo.GetImportJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get import job: %w", err)
	}
	return &ImportKnowledgeResponse{
		JobID:     job.JobID,
		Status:    string(job.Status),
		Total:     job.Total,
		Processed: job.Processed,
		Error:     job.ErrorMessage,
	}, nil
}

