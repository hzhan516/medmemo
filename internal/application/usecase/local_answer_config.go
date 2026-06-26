package usecase

// LocalAnswerConfig 聚合本地回答所需的业务配置，把模板、人称、事实 subject 从代码中抽离。
type LocalAnswerConfig struct {
	Subject     string                  // 查询事实三元组时使用的 subject
	UserSubject string                  // 模板中的人称占位符替换值
	Templates   map[MemoryIntent]string // intent -> 模板字符串
}

// NewLocalAnswerConfig 创建默认配置，供 Wire 注入。
func NewLocalAnswerConfig() *LocalAnswerConfig {
	return DefaultLocalAnswerConfig()
}

// DefaultLocalAnswerConfig 返回内置默认配置。
func DefaultLocalAnswerConfig() *LocalAnswerConfig {
	return &LocalAnswerConfig{
		Subject:     "用户",
		UserSubject: "你",
		Templates: map[MemoryIntent]string{
			IntentPersonalWeight:    "记录中显示，{subject}当前体重为 {object}。",
			IntentPersonalHeight:    "记录中显示，{subject}当前身高为 {object}。",
			IntentPersonalAge:       "记录中显示，{subject}当前年龄为 {object}。",
			IntentAllergyHistory:    "记录中显示，{subject}的过敏相关信息为：{object}。",
			IntentMedicationHistory: "记录中显示，{subject}的用药相关信息为：{object}。",
		},
	}
}
