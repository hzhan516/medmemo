import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type Theme = 'light' | 'dark' | 'system'
type ComplianceBarMode = 'always' | 'first' | 'off'
type UpdateChannel = 'stable' | 'beta'
type DesensitizationLevel = 'standard' | 'strict' | 'off'

interface SettingsState {
  theme: Theme
  selectedModel: string
  complianceBarMode: ComplianceBarMode
  autoCheckUpdate: boolean
  updateChannel: UpdateChannel
  desensitizationLevel: DesensitizationLevel
  dataRetentionDays: number

  setTheme: (theme: Theme) => void
  setSelectedModel: (model: string) => void
  setComplianceBarMode: (mode: ComplianceBarMode) => void
  setAutoCheckUpdate: (enabled: boolean) => void
  setUpdateChannel: (channel: UpdateChannel) => void
  setDesensitizationLevel: (level: DesensitizationLevel) => void
  setDataRetentionDays: (days: number) => void
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      theme: 'system',
      selectedModel: 'kimi-lite',
      complianceBarMode: 'always',
      autoCheckUpdate: true,
      updateChannel: 'beta',
      desensitizationLevel: 'standard',
      dataRetentionDays: 30,

      setTheme: (theme) => set({ theme }),
      setSelectedModel: (model) => set({ selectedModel: model }),
      setComplianceBarMode: (mode) => set({ complianceBarMode: mode }),
      setAutoCheckUpdate: (enabled) => set({ autoCheckUpdate: enabled }),
      setUpdateChannel: (channel) => set({ updateChannel: channel }),
      setDesensitizationLevel: (level) => set({ desensitizationLevel: level }),
      setDataRetentionDays: (days) => set({ dataRetentionDays: days }),
    }),
    {
      name: 'medmemo-settings',
    }
  )
)
