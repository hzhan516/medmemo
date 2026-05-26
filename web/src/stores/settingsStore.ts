import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type Theme = 'light' | 'dark' | 'system'
type ComplianceBarMode = 'always' | 'first' | 'off'
type UpdateChannel = 'stable' | 'beta'
type DesensitizationLevel = 'standard' | 'strict' | 'off'
export type ProviderHealthStatus = 'green' | 'yellow' | 'red' | 'unknown'

interface SettingsState {
  theme: Theme
  selectedModel: string
  complianceBarMode: ComplianceBarMode
  showConfidenceBar: boolean
  autoCheckUpdate: boolean
  updateChannel: UpdateChannel
  desensitizationLevel: DesensitizationLevel
  dataRetentionDays: number
  activeProviderId: string | null
  activeModelId: string | null
  providerHealthStatus: Record<string, ProviderHealthStatus>
  lastSelectedProviderId: string | null
  lastSeenVersionNotes: string

  setTheme: (theme: Theme) => void
  setSelectedModel: (model: string) => void
  setComplianceBarMode: (mode: ComplianceBarMode) => void
  setShowConfidenceBar: (show: boolean) => void
  setAutoCheckUpdate: (enabled: boolean) => void
  setUpdateChannel: (channel: UpdateChannel) => void
  setDesensitizationLevel: (level: DesensitizationLevel) => void
  setDataRetentionDays: (days: number) => void
  setActiveProviderId: (id: string | null) => void
  setActiveModelId: (id: string | null) => void
  setProviderHealthStatus: (id: string, status: ProviderHealthStatus) => void
  setLastSelectedProviderId: (id: string | null) => void
  setLastSeenVersionNotes: (version: string) => void
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      theme: 'system',
      selectedModel: 'kimi-lite',
      complianceBarMode: 'always',
      showConfidenceBar: true,
      autoCheckUpdate: true,
      updateChannel: 'beta',
      desensitizationLevel: 'standard',
      dataRetentionDays: 30,
      activeProviderId: null,
      activeModelId: null,
      providerHealthStatus: {},
      lastSelectedProviderId: null,
      lastSeenVersionNotes: '',

      setTheme: (theme) => set({ theme }),
      setSelectedModel: (model) => set({ selectedModel: model }),
      setComplianceBarMode: (mode) => set({ complianceBarMode: mode }),
      setShowConfidenceBar: (show) => set({ showConfidenceBar: show }),
      setAutoCheckUpdate: (enabled) => set({ autoCheckUpdate: enabled }),
      setUpdateChannel: (channel) => set({ updateChannel: channel }),
      setDesensitizationLevel: (level) => set({ desensitizationLevel: level }),
      setDataRetentionDays: (days) => set({ dataRetentionDays: days }),
      setActiveProviderId: (id) => set({ activeProviderId: id }),
      setActiveModelId: (id) => set({ activeModelId: id }),
      setProviderHealthStatus: (id, status) =>
        set((state) => ({
          providerHealthStatus: { ...state.providerHealthStatus, [id]: status },
        })),
      setLastSelectedProviderId: (id) => set({ lastSelectedProviderId: id }),
      setLastSeenVersionNotes: (version) => set({ lastSeenVersionNotes: version }),
    }),
    {
      name: 'medmemo-settings',
      partialize: (state) => ({
        theme: state.theme,
        selectedModel: state.selectedModel,
        complianceBarMode: state.complianceBarMode,
        showConfidenceBar: state.showConfidenceBar,
        autoCheckUpdate: state.autoCheckUpdate,
        updateChannel: state.updateChannel,
        desensitizationLevel: state.desensitizationLevel,
        dataRetentionDays: state.dataRetentionDays,
        activeProviderId: state.activeProviderId,
        activeModelId: state.activeModelId,
        lastSelectedProviderId: state.lastSelectedProviderId,
        lastSeenVersionNotes: state.lastSeenVersionNotes,
        // providerHealthStatus 不持久化（运行时状态）
      }),
    }
  )
)
