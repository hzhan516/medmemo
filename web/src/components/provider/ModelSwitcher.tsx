import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { ChevronDown, Cloud, Server, Plus } from 'lucide-react'
import { useSettingsStore, type ProviderHealthStatus } from '@/stores/settingsStore'
import { useProviderStore } from '@/stores/providerStore'
import { useProviderHealth } from '@/hooks/useProviderHealth'
interface ModelItem {
  modelId: string
  modelName: string
  providerId: string
  providerName: string
  providerGroup: string
  isLocal: boolean
}

/**
 * 运行时模型切换器（CherryStudio 风格）。
 * 展示所有已启用的模型，按服务商分组。
 * 选择模型后自动关联到对应 provider。
 */
export function ModelSwitcher() {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [toast, setToast] = useState<{ message: string; visible: boolean } | null>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)

  const providers = useProviderStore((s) => s.providers)
  const activeProviderId = useSettingsStore((s) => s.activeProviderId)
  const activeModelId = useSettingsStore((s) => s.activeModelId)
  const setActiveProviderId = useSettingsStore((s) => s.setActiveProviderId)
  const setActiveModelId = useSettingsStore((s) => s.setActiveModelId)
  const setLastSelectedProviderId = useSettingsStore((s) => s.setLastSelectedProviderId)
  const healthStatus = useSettingsStore((s) => s.providerHealthStatus)
  const { refreshHealth } = useProviderHealth()

  // 首次加载时刷新健康状态
  useEffect(() => {
    refreshHealth()
  }, [])

  // 构建所有已启用的模型列表
  const enabledModels = useMemo((): ModelItem[] => {
    const items: ModelItem[] = []
    for (const p of providers) {
      if (!p.enabled) continue
      const status = healthStatus[p.id] ?? 'unknown'
      if (status === 'red') continue
      const isLocal =
        p.group === '本地' || p.apiHost.includes('localhost') || p.apiHost.includes('127.0.0.1')
      const models = p.models && p.models.length > 0 ? p.models : p.modelId ? [{ id: p.modelId, name: p.modelId, enabled: true }] : []
      for (const m of models) {
        if (!m.enabled) continue
        items.push({
          modelId: m.id,
          modelName: m.name || m.id,
          providerId: p.id,
          providerName: p.name,
          providerGroup: p.group,
          isLocal,
        })
      }
    }
    return items
  }, [providers, healthStatus])

  // 按分组聚合模型
  const dropdownGroups = useMemo(() => {
    const map = new Map<string, ModelItem[]>()
    for (const item of enabledModels) {
      const list = map.get(item.providerGroup) || []
      list.push(item)
      map.set(item.providerGroup, list)
    }
    return Array.from(map.entries()).sort(([a], [b]) => a.localeCompare(b))
  }, [enabledModels])

  // 当前选中的模型信息
  const activeItem = useMemo(() => {
    return enabledModels.find((m) => m.modelId === activeModelId && m.providerId === activeProviderId)
  }, [enabledModels, activeModelId, activeProviderId])

  const activeStatus: ProviderHealthStatus = activeProviderId
    ? (healthStatus[activeProviderId] ?? 'unknown')
    : 'unknown'

  const showToast = useCallback((message: string) => {
    setToast({ message, visible: true })
    setTimeout(() => {
      setToast((prev) => (prev ? { ...prev, visible: false } : null))
      setTimeout(() => setToast(null), 200)
    }, 2000)
  }, [])

  const handleSwitch = useCallback(
    (modelId: string, providerId: string) => {
      const item = enabledModels.find((m) => m.modelId === modelId && m.providerId === providerId)
      if (!item) return
      setActiveModelId(modelId)
      setActiveProviderId(providerId)
      setLastSelectedProviderId(providerId)
      setOpen(false)
      showToast(`已切换至 ${item.modelName}`)
    },
    [enabledModels, setActiveModelId, setActiveProviderId, setLastSelectedProviderId, showToast]
  )

  // 点击外部关闭下拉
  useEffect(() => {
    if (!open) return
    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as Node
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(target) &&
        triggerRef.current &&
        !triggerRef.current.contains(target)
      ) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [open])

  const statusColor = useMemo(() => {
    switch (activeStatus) {
      case 'green':
        return 'bg-green-500'
      case 'yellow':
        return 'bg-yellow-500'
      case 'red':
        return 'bg-red-500'
      default:
        return 'bg-muted-foreground/50'
    }
  }, [activeStatus])

  // 空状态：无 Provider 配置
  if (providers.length === 0) {
    return (
      <button
        onClick={() => navigate('/settings')}
        className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-xs font-medium
                   bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
        data-testid="ms-empty-btn"
      >
        <Plus className="w-3.5 h-3.5" />
        添加模型
      </button>
    )
  }

  return (
    <div className="relative" data-testid="ms-wrapper">
      {/* 触发按钮 */}
      <button
        ref={triggerRef}
        onClick={() => setOpen((prev) => !prev)}
        className="flex items-center gap-2 px-2.5 py-1.5 rounded-md text-xs font-medium
                   bg-accent/50 hover:bg-accent transition-colors min-w-0"
        data-testid="ms-trigger"
      >
        <span
          className={`shrink-0 w-2 h-2 rounded-full ${statusColor}`}
          data-testid="ms-status-dot"
        />
        <span className="truncate max-w-[140px]" data-testid="ms-current-name">
          {activeItem ? activeItem.modelName : '未选择模型'}
        </span>
        <ChevronDown
          className={`w-3 h-3 text-muted-foreground transition-transform shrink-0 ${
            open ? 'rotate-180' : ''
          }`}
        />
      </button>

      {/* 下拉面板 */}
      {open && (
        <div
          ref={dropdownRef}
          className="absolute top-full right-0 mt-1.5 w-72 bg-popover border border-border
                     rounded-lg shadow-lg z-50 py-1 max-h-[60vh] overflow-y-auto"
          data-testid="ms-dropdown"
        >
          {dropdownGroups.length === 0 ? (
            <div className="px-3 py-4 text-center space-y-2">
              <p className="text-xs text-muted-foreground">暂无可用模型</p>
              <button
                onClick={() => {
                  setOpen(false)
                  navigate('/settings')
                }}
                className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
                data-testid="ms-dropdown-add-btn"
              >
                <Plus className="w-3 h-3" />
                添加模型
              </button>
            </div>
          ) : (
            <div className="space-y-1 px-1">
              {dropdownGroups.map(([groupName, items]) => (
                <div key={groupName} data-testid={`ms-group-${groupName}`}>
                  <div className="px-2.5 py-1 text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">
                    {groupName}
                  </div>
                  <div className="space-y-0.5">
                    {items.map((item) => {
                      const status = healthStatus[item.providerId] ?? 'unknown'
                      const isActive = activeModelId === item.modelId && activeProviderId === item.providerId
                      const isGreen = status === 'green'
                      const isYellow = status === 'yellow'

                      return (
                        <button
                          key={`${item.providerId}-${item.modelId}`}
                          onClick={() => isGreen && handleSwitch(item.modelId, item.providerId)}
                          disabled={!isGreen}
                          className={`w-full flex items-center gap-2.5 px-2.5 py-2 rounded-md text-left transition-colors
                            ${isActive ? 'bg-primary/10' : 'hover:bg-accent/50'}
                            ${isYellow ? 'opacity-50 cursor-not-allowed' : isGreen ? 'cursor-pointer' : 'cursor-not-allowed'}
                          `}
                          title={isYellow ? '连接缓慢' : isGreen ? '点击切换' : ''}
                          data-testid={`ms-model-${item.providerId}-${item.modelId}`}
                        >
                          <div
                            className={`shrink-0 w-6 h-6 rounded flex items-center justify-center
                              ${item.isLocal ? 'bg-amber-500/10 text-amber-600' : 'bg-blue-500/10 text-blue-600'}
                            `}
                          >
                            {item.isLocal ? (
                              <Server className="w-3 h-3" />
                            ) : (
                              <Cloud className="w-3 h-3" />
                            )}
                          </div>
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-1.5">
                              <span className="text-sm font-medium truncate">{item.modelName}</span>
                              {isActive && (
                                <span className="shrink-0 text-[10px] font-medium px-1 py-0.5 rounded-full bg-green-500/10 text-green-600">
                                  活跃
                                </span>
                              )}
                            </div>
                            <div className="text-[11px] text-muted-foreground truncate">
                              {item.providerName}
                            </div>
                          </div>
                          <span
                            className={`shrink-0 w-2 h-2 rounded-full ${
                              status === 'green'
                                ? 'bg-green-500'
                                : status === 'yellow'
                                  ? 'bg-yellow-500'
                                  : 'bg-muted-foreground/50'
                            }`}
                            data-testid={`ms-model-dot-${item.providerId}-${item.modelId}`}
                          />
                        </button>
                      )
                    })}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Toast 提示 */}
      {toast && (
        <div
          className={`absolute top-full right-0 mt-1.5 px-3 py-1.5 rounded-md bg-primary text-primary-foreground
                      text-xs font-medium shadow-lg z-50 transition-all duration-200
                      ${toast.visible ? 'opacity-100 translate-y-0' : 'opacity-0 -translate-y-1 pointer-events-none'}
                    `}
          data-testid="ms-toast"
        >
          {toast.message}
        </div>
      )}
    </div>
  )
}
