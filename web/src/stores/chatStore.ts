import { create } from 'zustand'

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
  isStreaming: boolean
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
  addMessage: (message: ChatMessage) => void
  updateLastMessage: (content: string, append?: boolean) => void
  appendToLastMessage: (content: string) => void
  setLastMessageError: (error: string) => void
  abortLastMessage: () => void
  setStreaming: (streaming: boolean) => void
  clearMessages: () => void
  setLastMessageWarnings: (warnings: string[]) => void
  setLastMessageReplacedTerms: (terms: string[]) => void
  setLastMessageTokenUsage: (promptTokens: number, completionTokens: number, totalTokens: number) => void

  setConversations: (conversations: Conversation[]) => void
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

export const useChatStore = create<ChatState>((set) => ({
  messages: [],
  isStreaming: false,
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

  addMessage: (message) =>
    set((state) => ({ messages: [...state.messages, message] })),

  updateLastMessage: (content, append = false) =>
    set((state) => {
      const msgs = [...state.messages]
      if (msgs.length === 0) return state
      const last = msgs[msgs.length - 1]
      if (last.role !== 'assistant') return state
      msgs[msgs.length - 1] = {
        ...last,
        content: append ? last.content + content : content,
      }
      return { messages: msgs }
    }),

  appendToLastMessage: (content) =>
    set((state) => {
      const msgs = [...state.messages]
      if (msgs.length === 0) return state
      const lastIdx = msgs.length - 1
      const last = msgs[lastIdx]
      if (last.role !== 'assistant') return state
      msgs[lastIdx] = {
        ...last,
        content: last.content + content,
      }
      return { messages: msgs }
    }),

  setLastMessageError: (error) =>
    set((state) => {
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
      return { messages: msgs }
    }),

  abortLastMessage: () =>
    set((state) => {
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
      return { messages: msgs }
    }),

  setStreaming: (streaming) => set({ isStreaming: streaming }),

  setLastMessageWarnings: (warnings) =>
    set((state) => {
      const msgs = [...state.messages]
      if (msgs.length === 0) return state
      const lastIdx = msgs.length - 1
      const last = msgs[lastIdx]
      if (last.role !== 'assistant') return state
      msgs[lastIdx] = { ...last, warnings }
      return { messages: msgs }
    }),

  setLastMessageReplacedTerms: (terms) =>
    set((state) => {
      const msgs = [...state.messages]
      if (msgs.length === 0) return state
      const lastIdx = msgs.length - 1
      const last = msgs[lastIdx]
      if (last.role !== 'assistant') return state
      msgs[lastIdx] = { ...last, replacedTerms: terms }
      return { messages: msgs }
    }),

  setLastMessageTokenUsage: (promptTokens, completionTokens, totalTokens) =>
    set((state) => {
      const msgs = [...state.messages]
      if (msgs.length === 0) return state
      const lastIdx = msgs.length - 1
      const last = msgs[lastIdx]
      if (last.role !== 'assistant') return state
      msgs[lastIdx] = { ...last, promptTokens, completionTokens, totalTokens }
      return { messages: msgs }
    }),

  clearMessages: () => set({ messages: [], currentConversationId: null }),

  setConversations: (conversations) => set({ conversations }),

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
      return {
        conversations: state.conversations.filter((c) => c.id !== id),
        deletedConversations: [
          { ...conv, deletedAt: Date.now() },
          ...state.deletedConversations,
        ],
        lastDeleted: { ...conv, deletedAt: Date.now() },
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
    set({ currentConversationId: id, messages: [] }),

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
