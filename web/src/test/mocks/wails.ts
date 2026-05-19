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
  for (const chunk of chunks) {
    accumulated += chunk
    await new Promise((r) => setTimeout(r, 10))
    EventsEmit('chat:stream:token', chunk)
  }
  if (!mockMessages[req.conversation_id]) {
    mockMessages[req.conversation_id] = []
  }
  mockMessages[req.conversation_id].push(
    { role: 'user', content: req.messages[req.messages.length - 1]?.content ?? '' },
    { role: 'assistant', content: accumulated }
  )
  EventsEmit('chat:stream:end', null)
}

const defaultStopGeneration = async (): Promise<void> => {
  EventsEmit('chat:stream:interrupted', null)
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

const defaultGetVersion = async (): Promise<string> => {
  return '0.5.0-test'
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
  GetVersion: () => resolveHandler('GetVersion', defaultGetVersion)(),
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
  for (const chunk of chunks) {
    await new Promise((r) => setTimeout(r, delayMs))
    EventsEmit('chat:stream:token', chunk)
  }
  if (emitEnd) {
    EventsEmit('chat:stream:end', null)
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
