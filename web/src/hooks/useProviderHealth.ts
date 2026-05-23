import { useCallback, useEffect, useRef } from 'react'
import { useProviderStore } from '@/stores/providerStore'
import { useSettingsStore } from '@/stores/settingsStore'

/**
 * Provider 运行时健康状态检测 Hook。
 * 通过 fetch('/v1/models') 快速探活，更新 settingsStore 中的 providerHealthStatus。
 * 不依赖后端健康检测引擎（TASK-036 完成前的前端兜底方案）。
 */
export function useProviderHealth() {
  const providers = useProviderStore((s) => s.providers)
  const setProviderHealthStatus = useSettingsStore((s) => s.setProviderHealthStatus)

  // 避免重复检测的 ref
  const checkedIdsRef = useRef<Set<string>>(new Set())

  const checkProvider = useCallback(
    async (provider: { id: string; apiHost: string; apiKey: string }): Promise<void> => {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 2000)
      const start = performance.now()

      try {
        const resp = await fetch(provider.apiHost + '/v1/models', {
          headers: provider.apiKey ? { Authorization: 'Bearer ' + provider.apiKey } : {},
          signal: controller.signal,
        })
        clearTimeout(timeoutId)
        const latency = performance.now() - start

        if (resp.ok) {
          setProviderHealthStatus(provider.id, latency < 1000 ? 'green' : 'yellow')
        } else if (resp.status === 404) {
          // Host 可达但端点不支持，视为 green（与 TASK-040 一致）
          setProviderHealthStatus(provider.id, 'green')
        } else {
          setProviderHealthStatus(provider.id, 'red')
        }
      } catch {
        clearTimeout(timeoutId)
        setProviderHealthStatus(provider.id, 'red')
      }
    },
    [setProviderHealthStatus]
  )

  const refreshHealth = useCallback(async (): Promise<void> => {
    const enabled = providers.filter((p) => p.enabled)
    checkedIdsRef.current = new Set(enabled.map((p) => p.id))
    await Promise.all(enabled.map((p) => checkProvider(p)))
  }, [providers, checkProvider])

  // Provider 列表变化时，自动检测新增/变更的 Provider
  useEffect(() => {
    const enabled = providers.filter((p) => p.enabled)
    const unchecked = enabled.filter((p) => !checkedIdsRef.current.has(p.id))
    if (unchecked.length > 0) {
      unchecked.forEach((p) => checkedIdsRef.current.add(p.id))
      // 并发检测，不阻塞 UI
      Promise.all(unchecked.map((p) => checkProvider(p))).catch(() => {
        // 静默处理，单个 Provider 检测失败不影响其他
      })
    }
  }, [providers, checkProvider])

  return { refreshHealth }
}
