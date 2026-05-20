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
 * 认证方式枚举。
 * 对应后端 models.AuthMethod。
 */
export type AuthMethod = 'api_key' | 'cli_token' | 'oauth_device' | 'service_account'

/**
 * 各认证方式特有的配置参数。
 */
export interface AuthParams {
  /** API Key 方式（api_key） */
  apiKey?: string
  /** CLI Token 方式（cli_token） */
  cliCredentialPath?: string
  /** OAuth Device Flow 方式（oauth_device） */
  oauthClientId?: string
  oauthAuthUrl?: string
  oauthTokenUrl?: string
  oauthRefreshToken?: string
  oauthAccessToken?: string
  oauthExpiresAt?: number
  /** Service Account 方式（service_account） */
  gcpProjectId?: string
  gcpRegion?: string
  saJson?: string
}

/**
 * CLI 检测结果，对应后端 auth.CLIDetectResult。
 */
export interface CLIDetectResult {
  /** 提供商类型：kimi | gemini */
  providerType: string
  /** CLI 凭证文件是否存在 */
  detected: boolean
  /** 检测到的凭证文件路径 */
  credentialPath: string
  /** 文件内容是否包含有效 token 信息 */
  loggedIn: boolean
  /** 检测过程中的错误提示 */
  error?: string
}

/**
 * 用户已添加的 Provider 配置。
 * 由 ProviderTemplate + 用户输入（认证配置、自定义参数）构成。
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
  /** API 密钥（api_key 方式下使用；内存中短暂存在，持久化由系统密钥环处理） */
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
  /** 认证方式，默认 api_key */
  authMethod: AuthMethod
  /** 认证方式特有参数 */
  authParams: AuthParams
  /** 标记 API Key 是否需补全（导入时留空则标记） */
  needsApiKey?: boolean
}
