import { create } from 'zustand'

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: number
  isStreaming?: boolean
}

interface ChatState {
  messages: ChatMessage[]
  isStreaming: boolean
  currentConversationId: string | null

  setConversationId: (id: string | null) => void
  addMessage: (message: ChatMessage) => void
  updateLastMessage: (content: string, append?: boolean) => void
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

  setStreaming: (streaming) => set({ isStreaming: streaming }),

  clearMessages: () => set({ messages: [], currentConversationId: null }),
}))
