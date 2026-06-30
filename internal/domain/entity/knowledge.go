package entity

import "time"

// KnowledgeSourceType 表示知识文档的来源格式。
type KnowledgeSourceType string

const (
	KnowledgeSourceMarkdown KnowledgeSourceType = "markdown"
	KnowledgeSourceJSONL    KnowledgeSourceType = "jsonl"
)

// KnowledgeDocument 表示导入知识库的一篇文档。
type KnowledgeDocument struct {
	DocumentID   string
	Title        string
	SourceType   KnowledgeSourceType
	Citation     string
	URL          string
	Language     string
	Checksum     string
	MetadataJSON string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// KnowledgeChunk 表示文档拆分后的片段。
type KnowledgeChunk struct {
	ChunkID      string
	DocumentID   string
	ChunkIndex   int
	Content      string
	TokenCount   int
	MetadataJSON string
	CreatedAt    time.Time
}

// KnowledgeCitation 表示聊天回答中引用知识库的条目。
type KnowledgeCitation struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Source  string  `json:"source"`
	URL     string  `json:"url,omitempty"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

// KnowledgeImportStatus 表示导入作业状态。
type KnowledgeImportStatus string

const (
	KnowledgeImportPending           KnowledgeImportStatus = "pending"
	KnowledgeImportIndexed           KnowledgeImportStatus = "indexed"
	KnowledgeImportError             KnowledgeImportStatus = "error"
	KnowledgeImportVectorUnavailable KnowledgeImportStatus = "indexed_vector_unavailable"
)

// KnowledgeImportJob 表示一次知识导入任务。
type KnowledgeImportJob struct {
	JobID        string
	Status       KnowledgeImportStatus
	Total        int
	Processed    int
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
