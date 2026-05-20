import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { ProviderConfig } from '@/types/provider'

export type ImportMode = 'merge' | 'overwrite'

export interface ImportResult {
  added: number
  skipped: number
  replaced: number
  errors: string[]
}

interface ProviderState {
  providers: ProviderConfig[]

  addProvider: (config: Omit<ProviderConfig, 'id' | 'createdAt' | 'updatedAt'>) => ProviderConfig
  updateProvider: (config: ProviderConfig) => void
  removeProvider: (id: string) => void
  getProviderById: (id: string) => ProviderConfig | undefined
  getEnabledProviders: () => ProviderConfig[]
  hasProvider: (templateId: string) => boolean
  importProviders: (configs: Omit<ProviderConfig, 'id' | 'createdAt' | 'updatedAt'>[], mode: ImportMode) => ImportResult
  replaceAllProviders: (configs: Omit<ProviderConfig, 'id' | 'createdAt' | 'updatedAt'>[]) => ImportResult
}

function generateId(templateId: string): string {
  return `${templateId}_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`
}

export const useProviderStore = create<ProviderState>()(
  persist(
    (set, get) => ({
      providers: [],

      addProvider: (config) => {
        const now = Date.now()
        const newProvider: ProviderConfig = {
          ...config,
          id: generateId(config.templateId),
          createdAt: now,
          updatedAt: now,
        }
        set((state) => ({
          providers: [...state.providers, newProvider],
        }))
        return newProvider
      },

      updateProvider: (config) => {
        set((state) => ({
          providers: state.providers.map((p) =>
            p.id === config.id ? { ...config, updatedAt: Date.now() } : p
          ),
        }))
      },

      removeProvider: (id) => {
        set((state) => ({
          providers: state.providers.filter((p) => p.id !== id),
        }))
      },

      getProviderById: (id) => {
        return get().providers.find((p) => p.id === id)
      },

      getEnabledProviders: () => {
        return get().providers.filter((p) => p.enabled)
      },

      hasProvider: (templateId) => {
        return get().providers.some((p) => p.templateId === templateId)
      },

      importProviders: (configs, mode) => {
        const result: ImportResult = { added: 0, skipped: 0, replaced: 0, errors: [] }
        if (mode === 'overwrite') {
          set({ providers: [] })
        }
        const existing = get().providers
        const toAdd: ProviderConfig[] = []

        for (let i = 0; i < configs.length; i++) {
          const cfg = configs[i]
          if (!cfg.name || !cfg.apiHost || !cfg.modelId) {
            result.errors.push(`第 ${i + 1} 条记录缺少必填字段（name/apiHost/modelId）`)
            continue
          }
          if (mode === 'merge') {
            const dupIndex = existing.findIndex((p) => p.name === cfg.name)
            if (dupIndex >= 0) {
              // 合并模式下同名冲突：直接跳过，记录为 skipped
              result.skipped++
              continue
            }
          }
          const now = Date.now()
          const newProvider: ProviderConfig = {
            ...cfg,
            id: generateId(cfg.templateId || 'custom'),
            createdAt: now,
            updatedAt: now,
            needsApiKey: !cfg.apiKey || cfg.apiKey === '',
          }
          toAdd.push(newProvider)
          result.added++
        }

        set((state) => ({
          providers: [...state.providers, ...toAdd],
        }))
        return result
      },

      replaceAllProviders: (configs) => {
        const result: ImportResult = { added: 0, skipped: 0, replaced: 0, errors: [] }
        const toAdd: ProviderConfig[] = []

        for (let i = 0; i < configs.length; i++) {
          const cfg = configs[i]
          if (!cfg.name || !cfg.apiHost || !cfg.modelId) {
            result.errors.push(`第 ${i + 1} 条记录缺少必填字段（name/apiHost/modelId）`)
            continue
          }
          const now = Date.now()
          toAdd.push({
            ...cfg,
            id: generateId(cfg.templateId || 'custom'),
            createdAt: now,
            updatedAt: now,
            needsApiKey: !cfg.apiKey || cfg.apiKey === '',
          })
          result.added++
        }

        set({ providers: toAdd })
        return result
      },
    }),
    {
      name: 'medmemo-providers',
      // apiKey 不持久化到 localStorage，由系统密钥环单独管理
      partialize: (state) => ({
        providers: state.providers.map((p) => ({ ...p, apiKey: '' })),
      }),
    }
  )
)
