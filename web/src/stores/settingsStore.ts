import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type Theme = 'light' | 'dark' | 'system'
type ComplianceBarMode = 'always' | 'first' | 'off'

interface SettingsState {
  theme: Theme
  selectedModel: string
  complianceBarMode: ComplianceBarMode

  setTheme: (theme: Theme) => void
  setSelectedModel: (model: string) => void
  setComplianceBarMode: (mode: ComplianceBarMode) => void
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      theme: 'system',
      selectedModel: 'kimi-lite',
      complianceBarMode: 'always',

      setTheme: (theme) => set({ theme }),
      setSelectedModel: (model) => set({ selectedModel: model }),
      setComplianceBarMode: (mode) => set({ complianceBarMode: mode }),
    }),
    {
      name: 'medmemo-settings',
    }
  )
)
