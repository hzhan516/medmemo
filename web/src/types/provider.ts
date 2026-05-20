/**
 * Provider 模板类型定义。
 * 对应 assets/provider-templates.json 中的静态模板数据。
 */
export interface ProviderTemplate {
  id: string
  name: string
  apiHost: string
  defaultModel: string
  models: string[]
  description: string
  docsUrl: string
  type: 'cloud' | 'local'
}

/**
 * 用户已添加的 Provider 配置。
 * 由 ProviderTemplate + 用户输入（API Key、自定义参数）构成。
 */
export interface ProviderConfig {
  /** 实例唯一ID（模板ID + 时间戳后缀，或自定义UUID） */
  id: string
  /** 来源模板ID */
  templateId: string
  /** 显示名称 */
  name: string
  /** API 基础地址 */
  apiHost: string
  /** API 密钥（内存中短暂存在，持久化由系统密钥环处理） */
  apiKey: string
  /** 当前选用的模型ID */
  modelId: string
  /** 温度参数，范围 0-2 */
  temperature: number
  /** 请求超时毫秒数 */
  timeoutMs: number
  /** 最大重试次数 */
  maxRetries: number
  /** 分组名称 */
  group: string
  /** 是否启用 */
  enabled: boolean
  /** 排序权重 */
  sortOrder: number
  /** 创建时间戳 */
  createdAt: number
  /** 更新时间戳 */
  updatedAt: number
}
