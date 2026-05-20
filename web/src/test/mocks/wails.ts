/**
 * Wails 运行时全局 Mock。
 * 为 vitest + jsdom 环境模拟 window.go.main.WailsApp 和 @wails/runtime/runtime 的完整绑定。
 * 所有方法默认返回 resolved Promise，可通过 setMockHandlers 注入自定义行为。
 */

import type {
  SendMessageRequest,
  SendMessageResponse,
  ConversationSummary,
  ModelInfo,
  EmergencyResult,
  DisclaimerStatus,
  UpdateInfoResponse,
  DownloadUpdateRequest,
  UpdateSettingsResponse,
} from '@wails/go/main/WailsApp'

// --- 内部状态 ---

const listeners = new Map<string, Set<(data: any) => void>>()

// 默认的 mock 处理器，可被测试覆盖
let mockHandlers: Record<string, (...args: any[]) => any> = {}

// 会话内存存储（用于模拟数据库持久化）
let mockConversations: ConversationSummary[] = []
let mockMessages: Record<string, Array<{ role: string; content: string }>> = {}
let mockNextConvId = 1

// --- Events 模拟 ---

export function EventsOn(eventName: string, callback: (...data: any) => void): () => void {
  if (!listeners.has(eventName)) {
    listeners.set(eventName, new Set())
  }
  listeners.get(eventName)!.add(callback)
  return () => {
    listeners.get(eventName)?.delete(callback)
  }
}

export function EventsOff(eventName: string): void {
  listeners.delete(eventName)
}

export function EventsEmit(eventName: string, ...data: any): void {
  const cbs = listeners.get(eventName)
  if (cbs) {
    cbs.forEach((cb) => cb(...data))
  }
}

export function EventsOnce(eventName: string, callback: (...data: any) => void): () => void {
  const wrapped = (...data: any[]) => {
    callback(...data)
    listeners.get(eventName)?.delete(wrapped)
  }
  return EventsOn(eventName, wrapped)
}

// --- 窗口主题模拟 ---

export function WindowSetDarkTheme(): void {}
export function WindowSetLightTheme(): void {}
export function WindowSetSystemDefaultTheme(): void {}
export function WindowSetTitle(_title: string): void {}
export function BrowserOpenURL(_url: string): void {}
export function Quit(): void {}

// --- WailsApp 方法模拟 ---

function resolveHandler<T>(name: string, fallback: T): T {
  return (mockHandlers[name] as T) ?? fallback
}

const defaultSendMessage = async (_req: SendMessageRequest): Promise<SendMessageResponse> => {
  return {
    reply: '这是一个模拟的 AI 回复。',
    confidence: 0.92,
    warnings: [],
  }
}

const defaultSendMessageStream = async (req: SendMessageRequest): Promise<void> => {
  // 延迟确保前端 EventsOn listener 在 React effect 周期后已稳定注册
  await new Promise((r) => setTimeout(r, 150))
  const chunks = ['这是', '一个', '模拟的', '流式', '回复。']
  let accumulated = ''
  EventsEmit('chat:stream_chunk', { type: 'start', payload: '', metadata: { model: 'kimi-lite' } })
  for (const chunk of chunks) {
    accumulated += chunk
    await new Promise((r) => setTimeout(r, 10))
    EventsEmit('chat:stream_chunk', { type: 'content', payload: chunk })
  }
  if (!mockMessages[req.conversation_id]) {
    mockMessages[req.conversation_id] = []
  }
  mockMessages[req.conversation_id].push(
    { role: 'user', content: req.messages[req.messages.length - 1]?.content ?? '' },
    { role: 'assistant', content: accumulated }
  )
  EventsEmit('chat:stream_chunk', { type: 'done', payload: '', metadata: { latency_ms: 50 } })
}

const defaultStopGeneration = async (): Promise<void> => {
  EventsEmit('chat:stream_chunk', { type: 'error', payload: '生成已中断' })
}

const defaultGetConversations = async (): Promise<ConversationSummary[]> => {
  return [...mockConversations]
}

const defaultCreateConversation = async (): Promise<string> => {
  const id = `conv_${mockNextConvId++}_${Date.now()}`
  mockConversations.unshift({
    id,
    title: '新对话',
    updated_at: String(Date.now()),
  })
  mockMessages[id] = []
  return id
}

const defaultGetModels = async (): Promise<ModelInfo[]> => {
  return [
    { id: 'kimi-lite', name: 'Kimi Lite', provider: 'kimi' },
    { id: 'gpt-4o-mini', name: 'GPT-4o Mini', provider: 'openai' },
    { id: 'qwen-turbo', name: '通义千问 Turbo', provider: 'qwen' },
    { id: 'llama3.1-8b', name: 'Llama 3.1 8B (本地)', provider: 'ollama' },
  ]
}

const defaultCheckEmergency = async (text: string): Promise<EmergencyResult> => {
  const aLevelKeywords = ['胸痛', '呼吸困难', '意识丧失', '大出血', '严重过敏']
  const bLevelKeywords = ['持续高热', '剧烈腹痛', '血尿', '视力突然下降']

  if (aLevelKeywords.some((k) => text.includes(k))) {
    return {
      level: 'A',
      message: '检测到 A 级紧急症状，建议立即就医或拨打 120。',
      action: '立即就医',
    }
  }
  if (bLevelKeywords.some((k) => text.includes(k))) {
    return {
      level: 'B',
      message: '检测到 B 级症状，建议尽快就医。',
      action: '尽快就医',
    }
  }
  return { level: 'none', message: '', action: '' }
}

const defaultGenerateTitle = async (_convId: string, _userMessage: string): Promise<void> => {
  setTimeout(() => {
    EventsEmit('chat:title:generated', { conv_id: _convId, title: '模拟生成的标题' })
  }, 50)
}

const defaultShowEmergencyDialog = async (_title: string, _message: string): Promise<void> => {}

const defaultGetDisclaimerStatus = async (): Promise<DisclaimerStatus> => {
  return {
    required: false,
    text: '本产品提供的信息仅供参考，不构成医疗诊断或治疗建议。',
    version: '1.0.0',
  }
}

const defaultAcceptDisclaimer = async (_version: string): Promise<void> => {}
const defaultDeclineDisclaimer = async (): Promise<void> => {}

const defaultReportComplianceFeedback = async (_ruleID: string, _originalText: string): Promise<void> => {}

const defaultCheckUpdate = async (): Promise<UpdateInfoResponse | null> => {
  return null
}

const defaultDownloadUpdate = async (_req: DownloadUpdateRequest): Promise<string> => {
  return '/tmp/mock-update.tar.gz'
}

const defaultApplyUpdate = async (_path: string): Promise<void> => {}

const defaultGetUpdateSettings = async (): Promise<UpdateSettingsResponse> => {
  return { check_enabled: true, channel: 'beta', skip_version: '' }
}

const defaultSetUpdateSettings = async (_req: UpdateSettingsResponse): Promise<void> => {}

const defaultSkipUpdateVersion = async (_v: string): Promise<void> => {}

const defaultOpenDownloadURL = (_url: string): void => {}

const defaultSaveAPIKey = async (_provider: string, _apiKey: string): Promise<void> => {}

const defaultHasAPIKey = async (_provider: string): Promise<boolean> => {
  return false
}

const defaultTestAPIKey = async (_provider: string, _apiKey: string, _apiHost: string): Promise<any> => {
  return { valid: true, message: '验证通过', models: ['gpt-4o'] }
}

const defaultGetVersion = async (): Promise<string> => {
  return '0.5.0-test'
}

const defaultDetectAuthMethods = async (): Promise<any> => {
  return {
    results: [
      { method: 'cli_token', available: false, connected: false, tier: 1, detail: '未检测到 CLI 工具' },
      { method: 'oauth_device', available: true, connected: false, tier: 2, detail: '支持 OAuth Device Flow' },
      { method: 'api_key', available: true, connected: false, tier: 3, detail: '可手动输入 API Key' },
      { method: 'local', available: false, connected: false, tier: 4, detail: '未检测到 Ollama' },
    ],
    recommended: 'oauth_device',
    all_unavailable: false,
  }
}

const defaultDetectCLIToken = async (_providerType: string): Promise<any> => {
  return {
    provider_type: _providerType,
    detected: false,
    credential_path: '',
    logged_in: false,
  }
}

const defaultBuildCLIProvider = async (_providerType: string, _modelID: string): Promise<any> => {
  return {
    id: `${_providerType}_cli_${Date.now()}`,
    templateId: _providerType,
    name: `${_providerType} (CLI)`,
    apiHost: _providerType === 'kimi' ? 'https://api.moonshot.cn' : 'https://generativelanguage.googleapis.com',
    apiKey: '',
    modelId: _modelID,
    temperature: 0.7,
    timeoutMs: 30000,
    maxRetries: 3,
    group: 'CLI',
    enabled: true,
    sortOrder: 0,
    createdAt: Date.now(),
    updatedAt: Date.now(),
    authMethod: 'cli_token',
    authParams: {},
  }
}

const defaultStartOAuthDeviceFlow = async (_providerType: string): Promise<any> => {
  return {
    user_code: 'ABCD-EFGH',
    verification_uri: 'https://platform.moonshot.cn/device',
    device_code: 'mock-device-code',
    expires_in: 600,
    interval: 5,
  }
}

const defaultCancelOAuthDeviceFlow = async (_deviceCode: string): Promise<void> => {}

const defaultGetOAuthDeviceFlowStatus = async (_deviceCode: string): Promise<any> => {
  return {
    device_code: _deviceCode,
    provider_type: 'kimi',
    status: 'pending',
  }
}

const defaultDetectOllama = async (): Promise<any> => {
  return {
    installed: false,
    running: false,
    has_smollm2: false,
    install_guide: 'curl -fsSL https://ollama.com/install.sh | sh',
  }
}

const defaultStartOllamaServer = async (): Promise<void> => {}
const defaultPullOllamaModel = async (_modelName: string): Promise<void> => {}
const defaultEnsureOllamaAndSmolLM2 = async (): Promise<any> => {
  return {
    installed: false,
    running: false,
    has_smollm2: false,
    install_guide: 'curl -fsSL https://ollama.com/install.sh | sh',
  }
}
const defaultCreateOllamaProvider = async (): Promise<any> => {
  return {
    id: `ollama_local_${Date.now()}`,
    templateId: 'ollama',
    name: 'Ollama (本地)',
    apiHost: 'http://localhost:11434',
    apiKey: '',
    modelId: 'smollm2:135m',
    temperature: 0.7,
    timeoutMs: 30000,
    maxRetries: 3,
    group: '本地',
    enabled: true,
    sortOrder: 0,
    createdAt: Date.now(),
    updatedAt: Date.now(),
    authMethod: 'api_key',
    authParams: {},
  }
}

// --- window.go.main.WailsApp 聚合对象 ---

export const MockWailsApp = {
  SendMessage: (req: SendMessageRequest) => resolveHandler('SendMessage', defaultSendMessage)(req),
  SendMessageStream: (req: SendMessageRequest) => resolveHandler('SendMessageStream', defaultSendMessageStream)(req),
  StopGeneration: () => resolveHandler('StopGeneration', defaultStopGeneration)(),
  GetConversations: () => resolveHandler('GetConversations', defaultGetConversations)(),
  CreateConversation: () => resolveHandler('CreateConversation', defaultCreateConversation)(),
  GetModels: () => resolveHandler('GetModels', defaultGetModels)(),
  CheckEmergency: (text: string) => resolveHandler('CheckEmergency', defaultCheckEmergency)(text),
  GenerateTitle: (convId: string, msg: string) => resolveHandler('GenerateTitle', defaultGenerateTitle)(convId, msg),
  ShowEmergencyDialog: (title: string, message: string) => resolveHandler('ShowEmergencyDialog', defaultShowEmergencyDialog)(title, message),
  GetDisclaimerStatus: () => resolveHandler('GetDisclaimerStatus', defaultGetDisclaimerStatus)(),
  AcceptDisclaimer: (version: string) => resolveHandler('AcceptDisclaimer', defaultAcceptDisclaimer)(version),
  DeclineDisclaimer: () => resolveHandler('DeclineDisclaimer', defaultDeclineDisclaimer)(),
  ReportComplianceFeedback: (ruleID: string, originalText: string) => resolveHandler('ReportComplianceFeedback', defaultReportComplianceFeedback)(ruleID, originalText),
  CheckUpdate: () => resolveHandler('CheckUpdate', defaultCheckUpdate)(),
  DownloadUpdate: (req: DownloadUpdateRequest) => resolveHandler('DownloadUpdate', defaultDownloadUpdate)(req),
  ApplyUpdate: (path: string) => resolveHandler('ApplyUpdate', defaultApplyUpdate)(path),
  GetUpdateSettings: () => resolveHandler('GetUpdateSettings', defaultGetUpdateSettings)(),
  SetUpdateSettings: (req: UpdateSettingsResponse) => resolveHandler('SetUpdateSettings', defaultSetUpdateSettings)(req),
  SkipUpdateVersion: (v: string) => resolveHandler('SkipUpdateVersion', defaultSkipUpdateVersion)(v),
  OpenDownloadURL: (url: string) => resolveHandler('OpenDownloadURL', defaultOpenDownloadURL)(url),
  SaveAPIKey: (provider: string, apiKey: string) => resolveHandler('SaveAPIKey', defaultSaveAPIKey)(provider, apiKey),
  HasAPIKey: (provider: string) => resolveHandler('HasAPIKey', defaultHasAPIKey)(provider),
  TestAPIKey: (provider: string, apiKey: string, apiHost: string) => resolveHandler('TestAPIKey', defaultTestAPIKey)(provider, apiKey, apiHost),
  GetVersion: () => resolveHandler('GetVersion', defaultGetVersion)(),
  DetectAuthMethods: () => resolveHandler('DetectAuthMethods', defaultDetectAuthMethods)(),
  DetectCLIToken: (providerType: string) => resolveHandler('DetectCLIToken', defaultDetectCLIToken)(providerType),
  BuildCLIProvider: (providerType: string, modelID: string) => resolveHandler('BuildCLIProvider', defaultBuildCLIProvider)(providerType, modelID),
  StartOAuthDeviceFlow: (providerType: string) => resolveHandler('StartOAuthDeviceFlow', defaultStartOAuthDeviceFlow)(providerType),
  CancelOAuthDeviceFlow: (deviceCode: string) => resolveHandler('CancelOAuthDeviceFlow', defaultCancelOAuthDeviceFlow)(deviceCode),
  GetOAuthDeviceFlowStatus: (deviceCode: string) => resolveHandler('GetOAuthDeviceFlowStatus', defaultGetOAuthDeviceFlowStatus)(deviceCode),
  DetectOllama: () => resolveHandler('DetectOllama', defaultDetectOllama)(),
  StartOllamaServer: () => resolveHandler('StartOllamaServer', defaultStartOllamaServer)(),
  PullOllamaModel: (modelName: string) => resolveHandler('PullOllamaModel', defaultPullOllamaModel)(modelName),
  EnsureOllamaAndSmolLM2: () => resolveHandler('EnsureOllamaAndSmolLM2', defaultEnsureOllamaAndSmolLM2)(),
  CreateOllamaProvider: () => resolveHandler('CreateOllamaProvider', defaultCreateOllamaProvider)(),
}

// --- 辅助工具函数 ---

/**
 * 设置自定义 mock 处理器，覆盖默认行为。
 */
export function setMockHandlers(handlers: Record<string, (...args: any[]) => any>): void {
  mockHandlers = { ...handlers }
}

/**
 * 重置所有 mock 状态到初始值。
 */
export function resetWailsMock(): void {
  mockHandlers = {}
  mockConversations = []
  mockMessages = {}
  mockNextConvId = 1
  listeners.clear()
}

/**
 * 获取当前 mock 的会话列表（用于断言）。
 */
export function getMockConversations(): ConversationSummary[] {
  return [...mockConversations]
}

/**
 * 获取指定会话的消息（用于断言）。
 */
export function getMockMessages(convId: string): Array<{ role: string; content: string }> {
  return [...(mockMessages[convId] ?? [])]
}

/**
 * 模拟流式响应，按指定 chunks 和时间间隔推送 token。
 */
export async function mockStreamResponse(
  chunks: string[],
  options: { delayMs?: number; emitEnd?: boolean } = {}
): Promise<void> {
  const { delayMs = 10, emitEnd = true } = options
  EventsEmit('chat:stream_chunk', { type: 'start', payload: '' })
  for (const chunk of chunks) {
    await new Promise((r) => setTimeout(r, delayMs))
    EventsEmit('chat:stream_chunk', { type: 'content', payload: chunk })
  }
  if (emitEnd) {
    EventsEmit('chat:stream_chunk', { type: 'done', payload: '' })
  }
}

/**
 * 模拟 L1 阻断级合规拦截响应。
 */
export function mockL1BlockResponse(): SendMessageResponse {
  return {
    reply: '我无法提供医疗诊断或治疗建议。建议您咨询专业医生。',
    confidence: 1.0,
    warnings: ['L1_BLOCK'],
  }
}

/**
 * 模拟 L2 警告级合规响应。
 */
export function mockL2WarningResponse(): SendMessageResponse {
  return {
    reply: '这可能是某种健康问题的表现，建议咨询医生确认。',
    confidence: 0.75,
    warnings: ['L2_WARNING', 'NOTICE:此内容涉及健康风险，仅供参考'],
  }
}

/**
 * 模拟包含脱敏占位符的响应。
 */
export function mockDesensitizedResponse(): SendMessageResponse {
  return {
    reply: '用户 <NAME_1> 的联系方式是 <PHONE_1>，身份证 <ID_1>。',
    confidence: 0.95,
    warnings: [],
  }
}

/**
 * 注册全局 window.go.main.WailsApp 对象。
 * 在测试 setup 中调用一次即可。
 */
export function registerGlobalWailsMock(): void {
  // @ts-ignore
  if (typeof window !== 'undefined') {
    // @ts-ignore
    window.go = window.go ?? {}
    // @ts-ignore
    window.go.main = window.go.main ?? {}
    // @ts-ignore
    window.go.main.WailsApp = MockWailsApp
  }
}
