package entity

// AppConfig 表示应用的核心配置领域对象。
// 不绑定任何框架特定的配置结构，保持纯领域表达。
type AppConfig struct {
	DataDir         string // 本地数据存储根目录
	DefaultModel    string // 默认使用的模型标识
	Language        string // 界面语言偏好
	EnableCloud     bool   // 是否允许云端模型调用
	EnableAnalytics bool   // 是否允许匿名使用数据统计
}

// Validate 校验配置合法性，返回领域错误。
func (c *AppConfig) Validate() error {
	if c.DataDir == "" {
		return ErrInvalidConfig
	}
	return nil
}
