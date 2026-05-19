package entity

import "github.com/medmemo/medmemo/pkg/models"

// AppConfig 表示应用的核心配置领域对象。
// 不绑定任何框架特定的配置结构，保持纯领域表达。
type AppConfig struct {
	DataDir              string               // 本地数据存储根目录
	DefaultModel         string               // 默认使用的模型标识
	Language             string               // 界面语言偏好
	EnableCloud          bool                 // 是否允许云端模型调用
	EnableAnalytics      bool                 // 是否允许匿名使用数据统计
	ProviderType         models.ProviderType  // LLM 提供商类型
	APIEndpoint          string               // 自定义 API 端点（可选，留空则使用提供商默认地址）
	ModelDir             string               // NER 模型目录路径（默认 resources/models/distilbert-ner）
	UpdateCheckEnabled   bool                 // 是否启用自动更新检测
	UpdateChannel        UpdateChannel        // 更新通道（stable / beta）
	DesensitizationLevel DesensitizationLevel // 脱敏级别（standard / strict / off）
	DataRetentionDays    int                  // 本地数据留存天数，0 表示永久保留
}

// Validate 校验配置合法性，返回领域错误。
func (c *AppConfig) Validate() error {
	if c.DataDir == "" {
		return ErrInvalidConfig
	}
	return nil
}
