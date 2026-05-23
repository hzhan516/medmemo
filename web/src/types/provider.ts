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
  authMethods: AuthMethod[]
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
 * 认证降级事件，对应后端 runtime.EventsEmit("auth:degraded", ...)。
 */
export interface AuthDegradedEvent {
  /** 触发降级的 Provider ID */
  providerID: string
  /** 降级原因 */
  reason: string
}

/**
 * OAuth Device Flow 启动响应，对应后端 DeviceFlowStartResponse。
 */
export interface DeviceFlowStartResponse {
  /** 用户码，供前端展示 */
  userCode: string
  /** 验证 URL，供用户浏览器访问 */
  verificationURI: string
  /** 设备码，后端轮询用 */
  deviceCode: string
  /** 设备码有效期（秒） */
  expiresIn: number
  /** 建议轮询间隔（秒） */
  interval: number
  /** 本地回调服务器地址（可选，用于同设备授权后的自动通知） */
  redirectURI?: string
}

/**
 * OAuth Device Flow 状态响应，对应后端 DeviceFlowStatusResponse。
 */
export interface DeviceFlowStatusResponse {
  /** 设备码 */
  deviceCode: string
  /** 厂商类型 */
  providerType: string
  /** 当前状态：pending / slow_down / success / error / cancelled */
  status: string
  /** 错误信息 */
  error?: string
  /** 授权成功后生成的 Provider ID */
  providerID?: string
  /** 授权成功后生成的 Provider 名称 */
  providerName?: string
}

/**
 * OAuth Device Flow 支持的厂商信息，对应后端 OAuthDeviceFlowProviderInfo。
 */
export interface OAuthDeviceFlowProviderInfo {
  /** 厂商类型 */
  providerType: string
  /** 显示名称 */
  name: string
  /** 是否可用（环境变量已配置） */
  available: boolean
  /** 环境变量是否已配置 */
  configured: boolean
  /** 状态描述 */
  detail: string
}

/**
 * OAuth Device Flow 成功事件，对应后端 runtime.EventsEmit("oauth:success", ...)。
 */
export interface OAuthSuccessEvent {
  /** 设备码 */
  deviceCode: string
  /** 厂商类型 */
  providerType: string
  /** 新创建的 Provider ID */
  providerID: string
  /** Provider 显示名称 */
  providerName: string
}

/**
 * Ollama 环境检测结果，对应后端 OllamaDetectResult。
 */
export interface OllamaDetectResult {
  /** ollama 命令是否存在于 PATH */
  installed: boolean
  /** 11434 端口是否响应 */
  running: boolean
  /** smollm2:135m 模型是否已下载 */
  has_smollm2: boolean
  /** 未安装时返回的安装引导文本 */
  install_guide?: string
  /** 正在后台启动服务 */
  server_starting?: boolean
  /** 模型下载进度文本 */
  pull_progress?: string
}

/**
 * 单种认证方式的检测结果，对应后端 AuthMethodDetectStatus。
 */
export interface AuthMethodDetectStatus {
  /** 认证方式：cli_token | oauth_device | api_key | local */
  method: string
  /** 该方式是否可用 */
  available: boolean
  /** 是否已连接/认证成功 */
  connected: boolean
  /** Tier 优先级 1-4 */
  tier: number
  /** 检测到的厂商类型 */
  provider_type?: string
  /** 状态描述文本 */
  detail?: string
  /** 不可用原因 */
  error?: string
}

/**
 * 认证方式统一检测结果，对应后端 AuthDetectResult。
 */
export interface AuthDetectResult {
  /** 各认证方式检测状态列表 */
  results: AuthMethodDetectStatus[]
  /** 推荐的方法 */
  recommended: string
  /** 是否全部不可用 */
  all_unavailable: boolean
}

/**
 * 服务商下的单个模型配置。
 */
export interface ProviderModel {
  /** 模型ID */
  id: string
  /** 显示名称 */
  name: string
  /** 是否启用 */
  enabled: boolean
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
  /** 当前选用的默认模型ID（向后兼容） */
  modelId: string
  /** 该服务商下的模型列表 */
  models?: ProviderModel[]
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
