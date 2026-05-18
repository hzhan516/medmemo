import { create } from 'zustand'

interface AppState {
  sidebarOpen: boolean
  currentConversationId: string | null
  currentModel: string
  complianceNoticeDismissed: boolean

  toggleSidebar: () => void
  setCurrentConversation: (id: string | null) => void
  setCurrentModel: (model: string) => void
  dismissComplianceNotice: () => void
}

export const useAppStore = create<AppState>((set) => ({
  sidebarOpen: true,
  currentConversationId: null,
  currentModel: 'kimi-lite',
  complianceNoticeDismissed: false,

  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
  setCurrentConversation: (id) => set({ currentConversationId: id }),
  setCurrentModel: (model) => set({ currentModel: model }),
  dismissComplianceNotice: () => set({ complianceNoticeDismissed: true }),
}))
