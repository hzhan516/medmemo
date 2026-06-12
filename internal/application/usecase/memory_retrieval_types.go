package usecase

import (
	"time"
)

// RetrievalPath 标识检索召回路径。
type RetrievalPath string

const (
	PathIntent  RetrievalPath = "intent"
	PathKeyword RetrievalPath = "keyword"
	PathVector  RetrievalPath = "vector"
	PathRecent  RetrievalPath = "recent"
)

// PathStatus 记录单条检索路径的完成状态与失败原因。
type PathStatus struct {
	Path    RetrievalPath
	Status  string // "success" / "failure" / "skipped"
	Reason  string // 失败或跳过原因
	Latency time.Duration
}

// RetrievalRequest 封装一次多路召回检索请求。
type RetrievalRequest struct {
	RawQuery      string
	SessionID     string
	Limit         int
	Subject       string // 事实主体，默认 "用户"
	Normalized    string // 规范化后的 query
	ExpandedQuery string // 用于向量检索的扩展 query（本地生成）
	Intent        *IntentResult
}

// RetrievalCandidate 统一的多路召回候选记忆。
type RetrievalCandidate struct {
	FactID     string
	Content    string // subject + predicate + object，用于注入 LLM 上下文
	Snippet    string // 截断 50 字符的短文本，用于诊断日志
	CreatedAt  time.Time
	Confidence float64

	// 召回路径追踪
	MatchedPaths []RetrievalPath

	// 多路评分
	IntentLevel      int     // 意图匹配级别：ConfidenceHigh→3, Medium→2, Low→1, 无→0
	KeywordScore     float64 // 关键词匹配分数 [0,1]
	VectorSimilarity float64 // 向量余弦相似度 [0,1]
	RecencyScore     float64 // 时效性分数（指数衰减后）[0,1]

	// 诊断信息
	Reasons      []string // 命中原因列表
	RejectReason string   // 被 rerank 或 token budget 剔除的原因
}

// RerankScore 封装排序用综合分数。
type RerankScore struct {
	IntentLevel      int
	KeywordScore     float64
	VectorSimilarity float64
	RecencyScore     float64
	Confidence       float64
}

// RetrievalDiagnostics 记录一次检索的完整诊断信息。
type RetrievalDiagnostics struct {
	DetectedIntent *IntentResult
	ExpandedQuery  string

	// 各路候选
	IntentCandidates  []RetrievalCandidate
	KeywordCandidates []RetrievalCandidate
	VectorCandidates  []RetrievalCandidate
	RecentCandidates  []RetrievalCandidate

	// 最终结果
	MergedCandidates   []RetrievalCandidate
	SelectedMemories   []RetrievalCandidate
	RejectedCandidates []RetrievalCandidate

	// 路径状态
	PathStatuses []PathStatus

	// 汇总
	TotalApprovedFacts int
	TotalRejected      int
}
