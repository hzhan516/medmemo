import { create } from 'zustand'

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: number
  isStreaming?: boolean
  interrupted?: boolean
  error?: string
}

interface ChatState {
  messages: ChatMessage[]
  isStreaming: boolean
  currentConversationId: string | null

  setConversationId: (id: string | null) => void
  addMessage: (message: ChatMessage) => void
  updateLastMessage: (content: string, append?: boolean) => void
  appendToLastMessage: (content: string) => void
  setLastMessageError: (error: string) => void
  abortLastMessage: () => void
  setStreaming: (streaming: boolean) => void
  clearMessages: () => void
}

export const useChatStore = create<ChatState>((set) => ({
  messages: [],
  isStreaming: false,
  currentConversationId: null,

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

  clearMessages: () => set({ messages: [], currentConversationId: null }),
}))
