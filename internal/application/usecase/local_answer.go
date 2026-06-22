package usecase

import (
	"strings"

	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// FactView 是 LocalAnswerService 对事实对象的最小只读视图。
type FactView interface {
	Object() string
}

type factAdapter struct {
	fact *entity.ExtractedFact
}

func (fa factAdapter) Object() string {
	if fa.fact == nil {
		return ""
	}
	return fa.fact.Object
}

// NilFact 是 Null Object，调用方不应再向 Format 传入 nil。
var NilFact FactView = factAdapter{}

// AsFactView 把 *entity.ExtractedFact 包装为 FactView；nil 时返回 NilFact。
func AsFactView(f *entity.ExtractedFact) FactView {
	return factAdapter{fact: f}
}

// LocalAnswerService 负责将已审批事实格式化为本地模板回答。
// 只复述已记录事实，不输出任何医疗建议。
type LocalAnswerService struct {
	cfg *LocalAnswerConfig
}

// NewLocalAnswerService 创建新的本地回答服务。
func NewLocalAnswerService(cfg *LocalAnswerConfig) *LocalAnswerService {
	if cfg == nil {
		cfg = DefaultLocalAnswerConfig()
	}
	return &LocalAnswerService{cfg: cfg}
}

// Subject 返回配置中用于事实查询的 subject。
func (s *LocalAnswerService) Subject() string {
	return s.cfg.Subject
}

// Format 根据意图和事实内容返回本地模板回答。
// 未配置的意图或事实对象为空时返回空字符串。
func (s *LocalAnswerService) Format(intent MemoryIntent, fact FactView) string {
	tmpl, ok := s.cfg.Templates[intent]
	if !ok {
		return ""
	}

	object := fact.Object()
	if object == "" {
		return ""
	}

	out := strings.ReplaceAll(tmpl, "{subject}", s.cfg.UserSubject)
	out = strings.ReplaceAll(out, "{object}", object)
	return out
}
