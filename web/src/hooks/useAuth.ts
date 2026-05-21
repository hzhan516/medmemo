import { useState, useCallback, useEffect, useRef } from 'react'
import { EventsOn } from '@wails/runtime'
import { DetectAuthMethods } from '@wails/go/main/WailsApp'
import type { AuthDetectResult } from '@/types/provider'

export type AuthPanel = 'cli_token' | 'oauth_device' | 'api_key' | 'service_account' | 'local'

interface AuthState {
  detecting: boolean
  detected: boolean
  error: string | null
  result: AuthDetectResult | null
  selectedMethod: AuthPanel | null
  expandedPanel: AuthPanel | null
  oauthDeviceCode: string | null
  oauthProviderType: string | null
  ollamaPulling: boolean
  ollamaPullProgress: string
  ollamaServerStarting: boolean
}

/**
 * 认证方式智能选择与状态管理 Hook。
 * 负责调用后端统一检测、管理面板展开状态、监听 OAuth/Ollama 事件。
 */
export function useAuth() {
  const [state, setState] = useState<AuthState>({
    detecting: false,
    detected: false,
    error: null,
    result: null,
    selectedMethod: null,
    expandedPanel: null,
    oauthDeviceCode: null,
    oauthProviderType: null,
    ollamaPulling: false,
    ollamaPullProgress: '',
    ollamaServerStarting: false,
  })

  const stateRef = useRef(state)
  stateRef.current = state

  // 监听 OAuth 事件
  useEffect(() => {
    const unSuccess = EventsOn('oauth:success', (_data: any) => {
      setState((prev) => ({ ...prev, oauthDeviceCode: null }))
    })
    const unError = EventsOn('oauth:error', (_data: any) => {
      setState((prev) => ({ ...prev, oauthDeviceCode: null }))
    })
    return () => {
      unSuccess()
      unError()
    }
  }, [])

  // 监听 Ollama 事件
  useEffect(() => {
    const unProgress = EventsOn('ollama:pull_progress', (data: any) => {
      setState((prev) => ({
        ...prev,
        ollamaPulling: true,
        ollamaPullProgress: data.progress || '',
      }))
    })
    const unDone = EventsOn('ollama:pull_done', (_data: any) => {
      setState((prev) => ({
        ...prev,
        ollamaPulling: false,
        ollamaPullProgress: '下载完成',
      }))
    })
    const unError = EventsOn('ollama:pull_error', (_data: any) => {
      setState((prev) => ({
        ...prev,
        ollamaPulling: false,
        ollamaPullProgress: '下载失败',
      }))
    })
    const unServerStarting = EventsOn('ollama:server_starting', (_data: any) => {
      setState((prev) => ({ ...prev, ollamaServerStarting: true }))
    })
    const unServerReady = EventsOn('ollama:server_ready', (_data: any) => {
      setState((prev) => ({ ...prev, ollamaServerStarting: false }))
    })
    const unServerError = EventsOn('ollama:server_error', (_data: any) => {
      setState((prev) => ({ ...prev, ollamaServerStarting: false }))
    })

    return () => {
      unProgress()
      unDone()
      unError()
      unServerStarting()
      unServerReady()
      unServerError()
    }
  }, [])

  const detect = useCallback(async () => {
    setState((prev) => ({
      ...prev,
      detecting: true,
      detected: false,
      error: null,
      result: null,
    }))

    try {
      const result = await DetectAuthMethods()
      setState((prev) => ({
        ...prev,
        detecting: false,
        detected: true,
        result,
        // 默认展开推荐的面板
        expandedPanel: (result.recommended as AuthPanel) || null,
      }))
      return result
    } catch (err) {
      const msg = err instanceof Error ? err.message : '检测失败'
      setState((prev) => ({
        ...prev,
        detecting: false,
        error: msg,
      }))
      return null
    }
  }, [])

  const selectMethod = useCallback((method: AuthPanel) => {
    setState((prev) => ({
      ...prev,
      selectedMethod: method,
      expandedPanel: prev.expandedPanel === method ? null : method,
    }))
  }, [])

  const expandNextAvailableTier = useCallback(() => {
    setState((prev) => {
      if (!prev.result) return prev
      const tiers = [1, 2, 3, 4]
      for (const tier of tiers) {
        const method = prev.result.results.find((r) => r.tier === tier && r.available)
        if (method) {
          return { ...prev, expandedPanel: method.method as AuthPanel }
        }
      }
      return prev
    })
  }, [])

  const setOAuthDeviceCode = useCallback((deviceCode: string | null, providerType: string | null) => {
    setState((prev) => ({
      ...prev,
      oauthDeviceCode: deviceCode,
      oauthProviderType: providerType,
    }))
  }, [])

  const reset = useCallback(() => {
    setState({
      detecting: false,
      detected: false,
      error: null,
      result: null,
      selectedMethod: null,
      expandedPanel: null,
      oauthDeviceCode: null,
      oauthProviderType: null,
      ollamaPulling: false,
      ollamaPullProgress: '',
      ollamaServerStarting: false,
    })
  }, [])

  return {
    ...state,
    detect,
    selectMethod,
    expandNextAvailableTier,
    setOAuthDeviceCode,
    reset,
  }
}
