import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  recomputeUsage,
  compressSession,
  registerContextUsageListeners,
} from '@/services/contextUsageService'
import { useChatStore } from '@/stores/chatStore'
import { useProviderStore } from '@/stores/providerStore'
import { useSettingsStore } from '@/stores/settingsStore'
import { resetWailsMock, setMockHandlers, EventsEmit } from '@/test/mocks/wails'

const CONV_ID = 'conv-service'

function makeProvider(id: string, modelId: string, enabled = true) {
  return {
    id,
    name: `Provider ${id}`,
    apiHost: 'https://example.com',
    modelId,
    models: [{ id: modelId, name: `Model ${modelId}`, enabled: true }],
    temperature: 0.7,
    timeoutMs: 30000,
    maxRetries: 3,
    maxTokens: 2048,
    group: 'cloud',
    enabled,
    sortOrder: 0,
    createdAt: Date.now(),
    updatedAt: Date.now(),
    auth_method: 'api_key',
    auth_params: {},
  }
}

describe('contextUsageService', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    resetWailsMock()
    setMockHandlers({})
    useChatStore.setState({
      currentConversationId: CONV_ID,
      conversations: [],
      messagesMap: {},
      contextUsageMap: {},
    })
    useProviderStore.setState({ providers: [] })
    useSettingsStore.setState({ activeProviderId: null, activeModelId: null })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('recomputeUsage', () => {
    it('debounces multiple calls within 250ms to a single EstimateContextUsage', async () => {
      const estimateSpy = vi.fn().mockResolvedValue({
        usedTokens: 100,
        maxTokens: 1000,
        ratio: 0.1,
        approximate: false,
      })
      setMockHandlers({ EstimateContextUsage: estimateSpy })
      useProviderStore.setState({ providers: [makeProvider('p1', 'm1')] })
      useSettingsStore.setState({ activeProviderId: 'p1', activeModelId: 'm1' })

      const p1 = recomputeUsage(CONV_ID)
      const p2 = recomputeUsage(CONV_ID)
      const p3 = recomputeUsage(CONV_ID)

      expect(estimateSpy).not.toHaveBeenCalled()

      await vi.advanceTimersByTimeAsync(250)
      await Promise.all([p1, p2, p3])

      expect(estimateSpy).toHaveBeenCalledTimes(1)
    })
  })

  describe('doRecomputeUsage provider/model resolution', () => {
    it('prefers session-level provider/model over global active', async () => {
      const estimateSpy = vi.fn().mockResolvedValue({
        usedTokens: 1,
        maxTokens: 10,
        ratio: 0.1,
        approximate: false,
      })
      setMockHandlers({ EstimateContextUsage: estimateSpy })
      useChatStore.setState({
        conversations: [
          { id: CONV_ID, title: 'Test', updatedAt: Date.now(), unread: 0, providerId: 'session-p', modelId: 'session-m' },
        ],
      })
      useProviderStore.setState({ providers: [makeProvider('fallback-p', 'fallback-m')] })
      useSettingsStore.setState({ activeProviderId: 'global-p', activeModelId: 'global-m' })

      await recomputeUsage(CONV_ID)
      await vi.advanceTimersByTimeAsync(250)

      expect(estimateSpy).toHaveBeenCalledWith(
        expect.objectContaining({ providerId: 'session-p', modelId: 'session-m' })
      )
    })

    it('falls back to global active provider/model when session lacks them', async () => {
      const estimateSpy = vi.fn().mockResolvedValue({
        usedTokens: 1,
        maxTokens: 10,
        ratio: 0.1,
        approximate: false,
      })
      setMockHandlers({ EstimateContextUsage: estimateSpy })
      useChatStore.setState({
        conversations: [{ id: CONV_ID, title: 'Test', updatedAt: Date.now(), unread: 0 }],
      })
      useProviderStore.setState({ providers: [makeProvider('fallback-p', 'fallback-m')] })
      useSettingsStore.setState({ activeProviderId: 'global-p', activeModelId: 'global-m' })

      await recomputeUsage(CONV_ID)
      await vi.advanceTimersByTimeAsync(250)

      expect(estimateSpy).toHaveBeenCalledWith(
        expect.objectContaining({ providerId: 'global-p', modelId: 'global-m' })
      )
    })

    it('falls back to first enabled provider when no session or global active', async () => {
      const estimateSpy = vi.fn().mockResolvedValue({
        usedTokens: 1,
        maxTokens: 10,
        ratio: 0.1,
        approximate: false,
      })
      setMockHandlers({ EstimateContextUsage: estimateSpy })
      useChatStore.setState({
        conversations: [{ id: CONV_ID, title: 'Test', updatedAt: Date.now(), unread: 0 }],
      })
      useProviderStore.setState({ providers: [makeProvider('enabled-p', 'enabled-m')] })

      await recomputeUsage(CONV_ID)
      await vi.advanceTimersByTimeAsync(250)

      expect(estimateSpy).toHaveBeenCalledWith(
        expect.objectContaining({ providerId: 'enabled-p', modelId: 'enabled-m' })
      )
    })

    it('skips and warns when no provider or model can be resolved', async () => {
      const estimateSpy = vi.fn().mockResolvedValue({
        usedTokens: 1,
        maxTokens: 10,
        ratio: 0.1,
        approximate: false,
      })
      setMockHandlers({ EstimateContextUsage: estimateSpy })
      useChatStore.setState({
        conversations: [{ id: CONV_ID, title: 'Test', updatedAt: Date.now(), unread: 0 }],
      })

      const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
      await recomputeUsage(CONV_ID)
      await vi.advanceTimersByTimeAsync(250)
      warnSpy.mockRestore()

      expect(estimateSpy).not.toHaveBeenCalled()
    })
  })

  describe('chat:stream:confidence listener', () => {
    it('calls setAuthoritativeUsed when prompt_tokens > 0', () => {
      useChatStore.setState({
        contextUsageMap: {
          [CONV_ID]: {
            usedTokens: 0,
            maxTokens: 1000,
            ratio: 0,
            approximate: false,
            authoritative: false,
            isCompressing: false,
            updatedAt: Date.now(),
          },
        },
      })
      const cleanup = registerContextUsageListeners()

      EventsEmit('chat:stream:confidence', {
        conversation_id: CONV_ID,
        prompt_tokens: 500,
      })

      const usage = useChatStore.getState().contextUsageMap[CONV_ID]
      expect(usage.usedTokens).toBe(500)
      expect(usage.authoritative).toBe(true)
      expect(usage.approximate).toBe(false)
      cleanup()
    })

    it('ignores events with prompt_tokens equal to 0', () => {
      useChatStore.setState({
        contextUsageMap: {
          [CONV_ID]: {
            usedTokens: 100,
            maxTokens: 1000,
            ratio: 0.1,
            approximate: false,
            authoritative: false,
            isCompressing: false,
            updatedAt: Date.now(),
          },
        },
      })
      const cleanup = registerContextUsageListeners()

      EventsEmit('chat:stream:confidence', {
        conversation_id: CONV_ID,
        prompt_tokens: 0,
      })

      expect(useChatStore.getState().contextUsageMap[CONV_ID].usedTokens).toBe(100)
      cleanup()
    })
  })

  describe('compressSession', () => {
    it('refreshes messages before recomputing usage on success', async () => {
      const calls: string[] = []
      setMockHandlers({
        CompressSession: async () => {
          calls.push('CompressSession')
        },
        GetConversationMessages: async () => {
          calls.push('GetConversationMessages')
          return [{ id: 'm1', role: 'user', content: 'hi', timestamp: String(Date.now()) }]
        },
        EstimateContextUsage: async () => {
          calls.push('EstimateContextUsage')
          return { usedTokens: 1, maxTokens: 10, ratio: 0.1, approximate: false }
        },
      })
      useProviderStore.setState({ providers: [makeProvider('p1', 'm1')] })
      useSettingsStore.setState({ activeProviderId: 'p1', activeModelId: 'm1' })

      await compressSession({
        conversationId: CONV_ID,
        providerId: 'p1',
        modelId: 'm1',
      })
      await vi.advanceTimersByTimeAsync(250)

      expect(calls).toEqual(['CompressSession', 'GetConversationMessages', 'EstimateContextUsage'])
      expect(useChatStore.getState().messagesMap[CONV_ID]).toHaveLength(1)
    })
  })

  describe('context:auto_compressed and context:usage_refresh listeners', () => {
    it('triggers recompute on context:auto_compressed', async () => {
      const estimateSpy = vi.fn().mockResolvedValue({
        usedTokens: 1,
        maxTokens: 10,
        ratio: 0.1,
        approximate: false,
      })
      setMockHandlers({ EstimateContextUsage: estimateSpy })
      useProviderStore.setState({ providers: [makeProvider('p1', 'm1')] })
      useSettingsStore.setState({ activeProviderId: 'p1', activeModelId: 'm1' })

      const cleanup = registerContextUsageListeners()
      EventsEmit('context:auto_compressed', { conversation_id: CONV_ID })
      await vi.advanceTimersByTimeAsync(250)

      expect(estimateSpy).toHaveBeenCalledTimes(1)
      cleanup()
    })

    it('triggers recompute on context:usage_refresh', async () => {
      const estimateSpy = vi.fn().mockResolvedValue({
        usedTokens: 1,
        maxTokens: 10,
        ratio: 0.1,
        approximate: false,
      })
      setMockHandlers({ EstimateContextUsage: estimateSpy })
      useProviderStore.setState({ providers: [makeProvider('p1', 'm1')] })
      useSettingsStore.setState({ activeProviderId: 'p1', activeModelId: 'm1' })

      const cleanup = registerContextUsageListeners()
      EventsEmit('context:usage_refresh', { conversation_id: CONV_ID })
      await vi.advanceTimersByTimeAsync(250)

      expect(estimateSpy).toHaveBeenCalledTimes(1)
      cleanup()
    })
  })

  describe('registerContextUsageListeners cleanup', () => {
    it('unsubscribes all subscriptions and listeners', async () => {
      const estimateSpy = vi.fn().mockResolvedValue({
        usedTokens: 1,
        maxTokens: 10,
        ratio: 0.1,
        approximate: false,
      })
      setMockHandlers({ EstimateContextUsage: estimateSpy })
      useProviderStore.setState({ providers: [makeProvider('p1', 'm1')] })
      useSettingsStore.setState({ activeProviderId: 'p1', activeModelId: 'm1' })

      // 订阅返回的清理函数应被调用
      const unsubSpies = [vi.fn(), vi.fn()]
      let subscribeCount = 0
      const originalSubscribe = useChatStore.subscribe.bind(useChatStore)
      vi.spyOn(useChatStore, 'subscribe').mockImplementation((listener) => {
        const spy = unsubSpies[subscribeCount++] ?? vi.fn()
        const unsub = originalSubscribe(listener)
        return () => {
          spy()
          unsub()
        }
      })

      const cleanup = registerContextUsageListeners()
      cleanup()

      expect(unsubSpies[0]).toHaveBeenCalledTimes(1)
      expect(unsubSpies[1]).toHaveBeenCalledTimes(1)

      // 清理后事件不应再触发估算
      EventsEmit('context:auto_compressed', { conversation_id: CONV_ID })
      EventsEmit('context:usage_refresh', { conversation_id: CONV_ID })
      EventsEmit('chat:stream:confidence', { conversation_id: CONV_ID, prompt_tokens: 100 })
      await vi.advanceTimersByTimeAsync(250)

      expect(estimateSpy).not.toHaveBeenCalled()
    })
  })
})
