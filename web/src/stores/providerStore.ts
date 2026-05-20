import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { ProviderConfig } from '@/types/provider'

interface ProviderState {
  providers: ProviderConfig[]

  addProvider: (config: Omit<ProviderConfig, 'id' | 'createdAt' | 'updatedAt'>) => ProviderConfig
  updateProvider: (config: ProviderConfig) => void
  removeProvider: (id: string) => void
  getProviderById: (id: string) => ProviderConfig | undefined
  getEnabledProviders: () => ProviderConfig[]
  hasProvider: (templateId: string) => boolean
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
