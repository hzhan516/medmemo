import { create } from 'zustand'
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
  initialized: boolean

  // 同步状态操作（由外层异步调用成功后触发）
  setProviders: (providers: ProviderConfig[]) => void
  addProvider: (config: ProviderConfig) => ProviderConfig
  updateProvider: (config: ProviderConfig) => void
  removeProvider: (id: string) => void

  getProviderById: (id: string) => ProviderConfig | undefined
  getEnabledProviders: () => ProviderConfig[]
  hasProvider: (templateId: string) => boolean

  // 导入/导出保持纯前端逻辑（数据转换后由外层逐条调用后端 API）
  importProviders: (configs: ProviderConfig[], mode: ImportMode) => ImportResult
  replaceAllProviders: (configs: ProviderConfig[]) => ImportResult
}

// ID 生成：templateId + 时间戳 + 4 位随机数
function generateId(templateId: string): string {
  return `${templateId}_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`
}

/**
 * 填充 ProviderConfig 默认值，旧数据 models 兜底。
 * 旧数据无 models 字段时，从 modelId 自动创建单模型列表。
 */
function normalizeProviderConfig(
  config: Omit<ProviderConfig, 'id' | 'createdAt' | 'updatedAt'>
): Omit<ProviderConfig, 'id' | 'createdAt' | 'updatedAt'> {
  const normalized: Omit<ProviderConfig, 'id' | 'createdAt' | 'updatedAt'> = {
    ...config,
    authMethod: config.authMethod || 'api_key',
    authParams: config.authParams || {},
    models: config.models || [],
  }

  // 旧数据 models 兜底
  if ((!normalized.models || normalized.models.length === 0) && normalized.modelId) {
    normalized.models = [
      { id: normalized.modelId, name: normalized.modelId, enabled: true },
    ]
  }

  return normalized
}

export const useProviderStore = create<ProviderState>()((set, get) => ({
  providers: [],
  initialized: false,

  setProviders: (providers) => {
    set({ providers, initialized: true })
  },

  addProvider: (config) => {
    const now = Date.now()
    const normalized = normalizeProviderConfig(config)
    const newProvider: ProviderConfig = {
      ...normalized,
      id: config.id || generateId(config.templateId || 'custom'),
      createdAt: config.createdAt ?? now,
      updatedAt: config.updatedAt ?? now,
      needsApiKey: !config.apiKey || config.apiKey === '',
    }
    set((state) => ({
      providers: [...state.providers, newProvider],
    }))
    return newProvider
  },

  updateProvider: (config) => {
    set((state) => ({
      providers: state.providers.map((p) =>
        p.id === config.id ? config : p
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
      const cfg = normalizeProviderConfig(configs[i])
      const hasValidModel = cfg.models && cfg.models.length > 0 && cfg.models.some((m) => m.enabled)
      if (!cfg.name || !cfg.apiHost || (!hasValidModel && !cfg.modelId)) {
        result.errors.push(`第 ${i + 1} 条记录缺少必填字段（name/apiHost/modelId 或 models）`)
        continue
      }
      if (mode === 'merge') {
        const dupIndex = existing.findIndex((p) => p.name === cfg.name)
        if (dupIndex >= 0) {
          result.skipped++
          continue
        }
      }
      const newProvider: ProviderConfig = {
        ...cfg,
        id: configs[i].id || generateId(cfg.templateId || 'custom'),
        createdAt: configs[i].createdAt ?? Date.now(),
        updatedAt: configs[i].updatedAt ?? Date.now(),
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
      const cfg = normalizeProviderConfig(configs[i])
      const hasValidModel = cfg.models && cfg.models.length > 0 && cfg.models.some((m) => m.enabled)
      if (!cfg.name || !cfg.apiHost || (!hasValidModel && !cfg.modelId)) {
        result.errors.push(`第 ${i + 1} 条记录缺少必填字段（name/apiHost/modelId 或 models）`)
        continue
      }
      toAdd.push({
        ...cfg,
        id: configs[i].id || generateId(cfg.templateId || 'custom'),
        createdAt: configs[i].createdAt ?? Date.now(),
        updatedAt: configs[i].updatedAt ?? Date.now(),
        needsApiKey: !cfg.apiKey || cfg.apiKey === '',
      })
      result.added++
    }

    set({ providers: toAdd })
    return result
  },
}))
