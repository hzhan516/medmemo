import { create } from 'zustand'
import type { ConfidenceResult } from '@/components/confidence/types'

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: number
  isStreaming?: boolean
  interrupted?: boolean
  error?: string
  warnings?: string[] // 合规检测标记：L2_WARNING / L3_NOTICE
  replacedTerms?: string[] // inline 替换中被替换的用词规则 ID 列表
  complianceFeedback?: 'none' | 'submitted' // 申诉状态
  promptTokens?: number // 该轮次输入 token 数
  completionTokens?: number // 该轮次输出 token 数
  totalTokens?: number // 该轮次总 token 数
  truncated?: boolean // 输出是否被截断（finish_reason == "length"）
  confidence?: ConfidenceResult // 回答置信度结果
  conversationId?: string // 所属会话 ID，用于在 currentConversationId 为 null 时仍能正确归档到 messagesMap
}

export interface EmergencyAlert {
  level: 'A' | 'B'
  message: string
  action: string
}

export interface Conversation {
  id: string
  title: string
  preview?: string
  updatedAt: number
  unread: number
  isPinned?: boolean
  deletedAt?: number
}

interface ChatState {
  messages: ChatMessage[]
  // 按会话缓存消息数组，实现真正的会话隔离
  messagesMap: Record<string, ChatMessage[]>
  // 当前会话是否在流式中（派生自 streamingIds）
  isStreaming: boolean
  // 哪些会话正在流式中
  streamingIds: Set<string>
  currentConversationId: string | null

  conversations: Conversation[]
  deletedConversations: Conversation[]
  searchQuery: string
  lastDeleted: Conversation | null
  showTrash: boolean
  dismissedBarSessions: string[] // 已手动关闭合规提示条的会话 ID 列表

  // 紧急症状检测状态
  emergencyAlert: EmergencyAlert | null
  emergencyWarningAcknowledged: boolean // B 级警告是否已点击「我已了解」

  setConversationId: (id: string | null) => void
  addMessage: (message: ChatMessage, convId?: string) => void
  addMessageForConversation: (convId: string, message: ChatMessage) => void
  updateLastMessage: (content: string, append?: boolean) => void
  appendToLastMessage: (content: string) => void
  appendToLastMessageForConversation: (convId: string, content: string) => void
  setLastMessageError: (error: string) => void
  setLastMessageErrorForConversation: (convId: string, error: string) => void
  abortLastMessage: () => void
  setStreaming: (streaming: boolean) => void
  setStreamingForConversation: (convId: string, streaming: boolean) => void
  clearMessages: () => void
  setLastMessageWarnings: (warnings: string[]) => void
  setLastMessageWarningsForConversation: (convId: string, warnings: string[]) => void
  setLastMessageReplacedTerms: (terms: string[]) => void
  setLastMessageReplacedTermsForConversation: (convId: string, terms: string[]) => void
  setLastMessageTokenUsage: (promptTokens: number, completionTokens: number, totalTokens: number) => void
  setLastMessageTokenUsageForConversation: (convId: string, promptTokens: number, completionTokens: number, totalTokens: number) => void
  setLastMessageConfidence: (confidence: ConfidenceResult) => void
  setLastMessageConfidenceForConversation: (convId: string, confidence: ConfidenceResult) => void
  setLastMessageTruncatedForConversation: (convId: string, truncated: boolean) => void
  replaceLastMessageForConversation: (convId: string, content: string) => void
  setMessages: (messages: ChatMessage[]) => void

  setConversations: (conversations: Conversation[]) => void
  setDeletedConversations: (deletedConversations: Conversation[]) => void
  addConversation: (conversation: Conversation) => void
  updateConversation: (id: string, updates: Partial<Conversation>) => void
  removeConversation: (id: string) => void
  pinConversation: (id: string) => void
  unpinConversation: (id: string) => void
  softDeleteConversation: (id: string) => void
  undoDelete: () => void
  permanentlyDeleteConversation: (id: string) => void
  restoreConversation: (id: string) => void
  setSearchQuery: (query: string) => void
  setShowTrash: (show: boolean) => void
  selectConversation: (id: string | null) => void
  cleanupOldDeleted: () => void

  dismissComplianceBarForSession: (sessionId: string) => void

  // 紧急症状状态管理
  setEmergencyAlert: (alert: EmergencyAlert | null) => void
  acknowledgeEmergencyWarning: () => void
}

const THIRTY_DAYS_MS = 30 * 24 * 60 * 60 * 1000
const MAX_CONVERSATIONS = 500

// 辅助函数：更新 messagesMap 中指定会话的最后一条 assistant 消息
function updateLastAssistantInMap(
  map: Record<string, ChatMessage[]>,
  convId: string,
  updater: (msg: ChatMessage) => ChatMessage
): Record<string, ChatMessage[]> {
  const msgs = [...(map[convId] || [])]
  if (msgs.length === 0) return map
  const lastIdx = msgs.length - 1
  const last = msgs[lastIdx]
  if (last.role !== 'assistant') return map
  msgs[lastIdx] = updater(last)
  return { ...map, [convId]: msgs }
}

export const useChatStore = create<ChatState>((set) => ({
  messages: [],
  messagesMap: {},
  isStreaming: false,
  streamingIds: new Set(),
  currentConversationId: null,

  conversations: [],
  deletedConversations: [],
  searchQuery: '',
  lastDeleted: null,
  showTrash: false,
  dismissedBarSessions: [],
  emergencyAlert: null,
  emergencyWarningAcknowledged: false,

  setConversationId: (id) => set({ currentConversationId: id }),

  addMessage: (message, explicitConvId) =>
    set((state) => {
      const convId = explicitConvId || message.conversationId || state.currentConversationId
      const msgs = [...state.messages, message]
      if (!convId) return { messages: msgs }
      return {
        messages: msgs,
        messagesMap: { ...state.messagesMap, [convId]: msgs },
      }
    }),

  addMessageForConversation: (convId, message) =>
    set((state) => {
      const convMsgs = [...(state.messagesMap[convId] || []), message]
      const newMap = { ...state.messagesMap, [convId]: convMsgs }
      if (state.currentConversationId === convId) {
        return { messages: convMsgs, messagesMap: newMap }
      }
      return { messagesMap: newMap }
    }),

  updateLastMessage: (content, append = false) =>
    set((state) => {
      const convId = state.currentConversationId
      const msgs = [...state.messages]
      if (msgs.length === 0) return state
      const last = msgs[msgs.length - 1]
      if (last.role !== 'assistant') return state
      msgs[msgs.length - 1] = {
        ...last,
        content: append ? last.content + content : content,
      }
      // 优先从最后一条消息的 conversationId 获取，确保 map 同步不受 currentConversationId 影响
      const mapKey = last.conversationId || convId
      if (!mapKey) return { messages: msgs }
      return {
        messages: msgs,
        messagesMap: { ...state.messagesMap, [mapKey]: msgs },
      }
    }),

  appendToLastMessage: (content) =>
    set((state) => {
      const convId = state.currentConversationId
      const msgs = [...state.messages]
      if (msgs.length === 0) return state
      const lastIdx = msgs.length - 1
      const last = msgs[lastIdx]
      if (last.role !== 'assistant') return state
      msgs[lastIdx] = {
        ...last,
        content: last.content + content,
      }
      const mapKey = last.conversationId || convId
      if (!mapKey) return { messages: msgs }
      return {
        messages: msgs,
        messagesMap: { ...state.messagesMap, [mapKey]: msgs },
      }
    }),

  appendToLastMessageForConversation: (convId, content) =>
    set((state) => {
      const newMap = updateLastAssistantInMap(state.messagesMap, convId, (last) => ({
        ...last,
        content: last.content + content,
      }))
      if (state.currentConversationId === convId) {
        return { messages: newMap[convId] || [], messagesMap: newMap }
      }
      return { messagesMap: newMap }
    }),

  setLastMessageError: (error) =>
    set((state) => {
      const convId = state.currentConversationId
      const msgs = [...state.messages]
      if (msgs.length === 0) return state
      const lastIdx = msgs.length - 1
      const last = msgs[lastIdx]
      if (last.role !== 'assistant') return state
      msgs[lastIdx] = {
        ...last,
        isStreaming: false,
        error,
      }
      const mapKey = last.conversationId || convId
      if (!mapKey) return { messages: msgs }
      return {
        messages: msgs,
        messagesMap: { ...state.messagesMap, [mapKey]: msgs },
      }
    }),

  setLastMessageErrorForConversation: (convId, error) =>
    set((state) => {
      const newMap = updateLastAssistantInMap(state.messagesMap, convId, (last) => ({
        ...last,
        isStreaming: false,
        error,
      }))
      if (state.currentConversationId === convId) {
        return { messages: newMap[convId] || [], messagesMap: newMap }
      }
      return { messagesMap: newMap }
    }),

  abortLastMessage: () =>
    set((state) => {
      const convId = state.currentConversationId
      const msgs = [...state.messages]
      if (msgs.length === 0) return state
      const lastIdx = msgs.length - 1
      const last = msgs[lastIdx]
      if (last.role !== 'assistant') return state
      msgs[lastIdx] = {
        ...last,
        isStreaming: false,
        interrupted: true,
      }
      const mapKey = last.conversationId || convId
      if (!mapKey) return { messages: msgs }
      return {
        messages: msgs,
        messagesMap: { ...state.messagesMap, [mapKey]: msgs },
      }
    }),

  setStreaming: (streaming) =>
    set((state) => {
      const convId = state.currentConversationId
      if (!convId) return { isStreaming: streaming }
      const newIds = new Set(state.streamingIds)
      if (streaming) {
        newIds.add(convId)
      } else {
        newIds.delete(convId)
      }
      return { isStreaming: streaming, streamingIds: newIds }
    }),

  setStreamingForConversation: (convId, streaming) =>
    set((state) => {
      const newIds = new Set(state.streamingIds)
      if (streaming) {
        newIds.add(convId)
      } else {
        newIds.delete(convId)
      }
      const isCurrentStreaming = state.currentConversationId
        ? newIds.has(state.currentConversationId)
        : false
      const newMap = updateLastAssistantInMap(state.messagesMap, convId, (last) => ({
        ...last,
        isStreaming: streaming,
      }))
      if (state.currentConversationId === convId) {
        return {
          isStreaming: isCurrentStreaming,
          streamingIds: newIds,
          messages: newMap[convId] || [],
          messagesMap: newMap,
        }
      }
      return {
        isStreaming: isCurrentStreaming,
        streamingIds: newIds,
        messagesMap: newMap,
      }
    }),

  clearMessages: () => set({ messages: [], currentConversationId: null }),

  setLastMessageWarnings: (warnings) =>
    set((state) => {
      const convId = state.currentConversationId
      const msgs = [...state.messages]
      if (msgs.length === 0) return state
      const lastIdx = msgs.length - 1
      const last = msgs[lastIdx]
      if (last.role !== 'assistant') return state
      msgs[lastIdx] = { ...last, warnings }
      const mapKey = last.conversationId || convId
      if (!mapKey) return { messages: msgs }
      return {
        messages: msgs,
        messagesMap: { ...state.messagesMap, [mapKey]: msgs },
      }
    }),

  setLastMessageWarningsForConversation: (convId, warnings) =>
    set((state) => {
      const newMap = updateLastAssistantInMap(state.messagesMap, convId, (last) => ({
        ...last,
        warnings,
      }))
      if (state.currentConversationId === convId) {
        return { messages: newMap[convId] || [], messagesMap: newMap }
      }
      return { messagesMap: newMap }
    }),

  setLastMessageReplacedTerms: (terms) =>
    set((state) => {
      const convId = state.currentConversationId
      const msgs = [...state.messages]
      if (msgs.length === 0) return state
      const lastIdx = msgs.length - 1
      const last = msgs[lastIdx]
      if (last.role !== 'assistant') return state
      msgs[lastIdx] = { ...last, replacedTerms: terms }
      const mapKey = last.conversationId || convId
      if (!mapKey) return { messages: msgs }
      return {
        messages: msgs,
        messagesMap: { ...state.messagesMap, [mapKey]: msgs },
      }
    }),

  setLastMessageReplacedTermsForConversation: (convId, terms) =>
    set((state) => {
      const newMap = updateLastAssistantInMap(state.messagesMap, convId, (last) => ({
        ...last,
        replacedTerms: terms,
      }))
      if (state.currentConversationId === convId) {
        return { messages: newMap[convId] || [], messagesMap: newMap }
      }
      return { messagesMap: newMap }
    }),

  setLastMessageTokenUsage: (promptTokens, completionTokens, totalTokens) =>
    set((state) => {
      const convId = state.currentConversationId
      const msgs = [...state.messages]
      if (msgs.length === 0) return state
      const lastIdx = msgs.length - 1
      const last = msgs[lastIdx]
      if (last.role !== 'assistant') return state
      msgs[lastIdx] = { ...last, promptTokens, completionTokens, totalTokens }
      const mapKey = last.conversationId || convId
      if (!mapKey) return { messages: msgs }
      return {
        messages: msgs,
        messagesMap: { ...state.messagesMap, [mapKey]: msgs },
      }
    }),

  setLastMessageTokenUsageForConversation: (convId, promptTokens, completionTokens, totalTokens) =>
    set((state) => {
      const newMap = updateLastAssistantInMap(state.messagesMap, convId, (last) => ({
        ...last,
        promptTokens,
        completionTokens,
        totalTokens,
      }))
      if (state.currentConversationId === convId) {
        return { messages: newMap[convId] || [], messagesMap: newMap }
      }
      return { messagesMap: newMap }
    }),

  setLastMessageConfidence: (confidence) =>
    set((state) => {
      const convId = state.currentConversationId
      const msgs = [...state.messages]
      if (msgs.length === 0) return state
      const lastIdx = msgs.length - 1
      const last = msgs[lastIdx]
      if (last.role !== 'assistant') return state
      msgs[lastIdx] = { ...last, confidence }
      const mapKey = last.conversationId || convId
      if (!mapKey) return { messages: msgs }
      return {
        messages: msgs,
        messagesMap: { ...state.messagesMap, [mapKey]: msgs },
      }
    }),

  setLastMessageConfidenceForConversation: (convId, confidence) =>
    set((state) => {
      const newMap = updateLastAssistantInMap(state.messagesMap, convId, (last) => ({
        ...last,
        confidence,
      }))
      if (state.currentConversationId === convId) {
        return { messages: newMap[convId] || [], messagesMap: newMap }
      }
      return { messagesMap: newMap }
    }),

  setLastMessageTruncatedForConversation: (convId, truncated) =>
    set((state) => {
      const newMap = updateLastAssistantInMap(state.messagesMap, convId, (last) => ({
        ...last,
        truncated,
      }))
      if (state.currentConversationId === convId) {
        return { messages: newMap[convId] || [], messagesMap: newMap }
      }
      return { messagesMap: newMap }
    }),

  replaceLastMessageForConversation: (convId, content) =>
    set((state) => {
      const newMap = updateLastAssistantInMap(state.messagesMap, convId, (last) => ({
        ...last,
        content,
      }))
      if (state.currentConversationId === convId) {
        return { messages: newMap[convId] || [], messagesMap: newMap }
      }
      return { messagesMap: newMap }
    }),

  setMessages: (messages) =>
    set((state) => {
      const convId = state.currentConversationId
      if (!convId) return { messages }
      return {
        messages,
        messagesMap: { ...state.messagesMap, [convId]: messages },
      }
    }),

  setConversations: (conversations) => set({ conversations }),
  setDeletedConversations: (deletedConversations) => set({ deletedConversations }),

  addConversation: (conversation) =>
    set((state) => {
      let next = [conversation, ...state.conversations]
      // 自动归档：超出 500 条时移除最早更新的非置顶会话
      if (next.length > MAX_CONVERSATIONS) {
        const unpinned = next.filter((c) => !c.isPinned)
        if (unpinned.length > 0) {
          unpinned.sort((a, b) => a.updatedAt - b.updatedAt)
          const toRemove = unpinned[0]
          next = next.filter((c) => c.id !== toRemove.id)
        }
      }
      return { conversations: next }
    }),

  updateConversation: (id, updates) =>
    set((state) => ({
      conversations: state.conversations.map((c) =>
        c.id === id ? { ...c, ...updates } : c
      ),
    })),

  removeConversation: (id) =>
    set((state) => ({
      conversations: state.conversations.filter((c) => c.id !== id),
    })),

  pinConversation: (id) =>
    set((state) => ({
      conversations: state.conversations.map((c) =>
        c.id === id ? { ...c, isPinned: true } : c
      ),
    })),

  unpinConversation: (id) =>
    set((state) => ({
      conversations: state.conversations.map((c) =>
        c.id === id ? { ...c, isPinned: false } : c
      ),
    })),

  softDeleteConversation: (id) =>
    set((state) => {
      const conv = state.conversations.find((c) => c.id === id)
      if (!conv) return state
      // 切换会话前先将当前消息保存到 map
      const currentId = state.currentConversationId
      const newMap = currentId
        ? { ...state.messagesMap, [currentId]: [...state.messages] }
        : state.messagesMap
      return {
        conversations: state.conversations.filter((c) => c.id !== id),
        deletedConversations: [
          { ...conv, deletedAt: Date.now() },
          ...state.deletedConversations,
        ],
        lastDeleted: { ...conv, deletedAt: Date.now() },
        messages: [],
        currentConversationId: null,
        messagesMap: newMap,
      }
    }),

  undoDelete: () =>
    set((state) => {
      if (!state.lastDeleted) return state
      const restored = { ...state.lastDeleted, deletedAt: undefined }
      return {
        conversations: [restored, ...state.conversations],
        deletedConversations: state.deletedConversations.filter(
          (c) => c.id !== state.lastDeleted!.id
        ),
        lastDeleted: null,
      }
    }),

  permanentlyDeleteConversation: (id) =>
    set((state) => ({
      deletedConversations: state.deletedConversations.filter(
        (c) => c.id !== id
      ),
      lastDeleted:
        state.lastDeleted?.id === id ? null : state.lastDeleted,
    })),

  restoreConversation: (id) =>
    set((state) => {
      const conv = state.deletedConversations.find((c) => c.id === id)
      if (!conv) return state
      const restored = { ...conv, deletedAt: undefined }
      return {
        conversations: [restored, ...state.conversations],
        deletedConversations: state.deletedConversations.filter(
          (c) => c.id !== id
        ),
        lastDeleted:
          state.lastDeleted?.id === id ? null : state.lastDeleted,
      }
    }),

  setSearchQuery: (query) => set({ searchQuery: query }),

  setShowTrash: (show) => set({ showTrash: show }),

  selectConversation: (id) =>
    set((state) => {
      const currentId = state.currentConversationId
      // 保存当前会话的消息到 map（始终创建新对象，避免直接修改原 state）
      let newMap = currentId
        ? { ...state.messagesMap, [currentId]: [...state.messages] }
        : { ...state.messagesMap }
      // 从 map 加载目标会话的消息
      let loadedMessages = id ? (newMap[id] || []) : []
      // 防御性同步：若 map 中该会话消息为空但 messages 数组属于该会话且非空，保留当前 messages
      if (id && loadedMessages.length === 0 && state.currentConversationId === id && state.messages.length > 0) {
        loadedMessages = [...state.messages]
        newMap = { ...newMap, [id]: loadedMessages }
      }
      return {
        currentConversationId: id,
        messages: loadedMessages,
        messagesMap: newMap,
        isStreaming: id ? state.streamingIds.has(id) : false,
      }
    }),

  cleanupOldDeleted: () =>
    set((state) => ({
      deletedConversations: state.deletedConversations.filter(
        (c) => !c.deletedAt || Date.now() - c.deletedAt < THIRTY_DAYS_MS
      ),
    })),

  dismissComplianceBarForSession: (sessionId) =>
    set((state) => ({
      dismissedBarSessions: state.dismissedBarSessions.includes(sessionId)
        ? state.dismissedBarSessions
        : [...state.dismissedBarSessions, sessionId],
    })),

  setEmergencyAlert: (alert) =>
    set((state) => ({
      emergencyAlert: alert,
      // A 级弹窗重置确认状态；B 级保持原状态
      emergencyWarningAcknowledged:
        alert?.level === 'A' ? false : state.emergencyWarningAcknowledged,
    })),

  acknowledgeEmergencyWarning: () =>
    set({ emergencyWarningAcknowledged: true }),
}))
