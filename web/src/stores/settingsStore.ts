import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type Theme = 'light' | 'dark' | 'system'

interface SettingsState {
  theme: Theme
  selectedModel: string
  complianceNoticeDismissed: boolean

  setTheme: (theme: Theme) => void
  setSelectedModel: (model: string) => void
  dismissComplianceNotice: () => void
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      theme: 'system',
      selectedModel: 'kimi-lite',
      complianceNoticeDismissed: false,

      setTheme: (theme) => set({ theme }),
      setSelectedModel: (model) => set({ selectedModel: model }),
      dismissComplianceNotice: () => set({ complianceNoticeDismissed: true }),
    }),
    {
      name: 'medmemo-settings',
    }
  )
)
