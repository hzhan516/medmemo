import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type Theme = 'light' | 'dark' | 'system'
type ComplianceBarMode = 'always' | 'first' | 'off'
type UpdateChannel = 'stable' | 'beta'

interface SettingsState {
  theme: Theme
  selectedModel: string
  complianceBarMode: ComplianceBarMode
  autoCheckUpdate: boolean
  updateChannel: UpdateChannel

  setTheme: (theme: Theme) => void
  setSelectedModel: (model: string) => void
  setComplianceBarMode: (mode: ComplianceBarMode) => void
  setAutoCheckUpdate: (enabled: boolean) => void
  setUpdateChannel: (channel: UpdateChannel) => void
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      theme: 'system',
      selectedModel: 'kimi-lite',
      complianceBarMode: 'always',
      autoCheckUpdate: true,
      updateChannel: 'beta',

      setTheme: (theme) => set({ theme }),
      setSelectedModel: (model) => set({ selectedModel: model }),
      setComplianceBarMode: (mode) => set({ complianceBarMode: mode }),
      setAutoCheckUpdate: (enabled) => set({ autoCheckUpdate: enabled }),
      setUpdateChannel: (channel) => set({ updateChannel: channel }),
    }),
    {
      name: 'medmemo-settings',
    }
  )
)
