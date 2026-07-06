import * as WailsApp from '@wails/go/main/WailsApp'
import { EventsOn } from '@wails/runtime/runtime'
import { main, models } from '@wails/go/models'
import { useChatStore } from '@/stores/chatStore'
import { useSettingsStore } from '@/stores/settingsStore'
import { useProviderStore } from '@/stores/providerStore'
import { logger } from '@/lib/logger'

const DEBOUNCE_MS = 250

interface Deferred {
  promise: Promise<void>
  resolve: () => void
  timer: ReturnType<typeof setTimeout>
}

const pending = new Map<string, Deferred>()

function getDeferred(convId: string): Deferred {
  const existing = pending.get(convId)
  if (existing) {
    clearTimeout(existing.timer)
    return existing
  }

  let resolveFn: () => void = () => {}
  const promise = new Promise<void>((resolve) => {
    resolveFn = resolve
  })

  const deferred: Deferred = {
    promise,
    resolve: resolveFn,
    timer: 0 as unknown as ReturnType<typeof setTimeout>,
  }
  pending.set(convId, deferred)
  return deferred
}

/**
 * 重新估算指定会话的上下文用量。
 * 调用会被防抖 250ms，避免消息追加过程中产生大量估算请求。
 */
export function recomputeUsage(convId: string): Promise<void> {
  const deferred = getDeferred(convId)

  deferred.timer = setTimeout(async () => {
    pending.delete(convId)
    try {
      await doRecomputeUsage(convId)
    } catch (err) {
      logger.error('[contextUsage] failed to recompute usage:', err)
    } finally {
      deferred.resolve()
    }
  }, DEBOUNCE_MS)

  return deferred.promise
}

async function doRecomputeUsage(convId: string): Promise<void> {
  const { messagesMap, setContextUsage } = useChatStore.getState()
  const { activeProviderId, activeModelId } = useSettingsStore.getState()
  const { providers, getEnabledProviders } = useProviderStore.getState()

  const messages = messagesMap[convId] ?? []

  let providerId = activeProviderId ?? ''
  let modelId = activeModelId ?? ''

  if (!providerId || !modelId) {
    const enabledProviders = getEnabledProviders()
    const fallbackProvider = enabledProviders[0] ?? providers.find((p) => p.enabled)
    if (fallbackProvider) {
      providerId = providerId || fallbackProvider.id
      modelId =
        modelId ||
        fallbackProvider.models?.find((m) => m.enabled)?.id ||
        fallbackProvider.modelId
    }
  }

  if (!providerId || !modelId) {
    logger.warn('[contextUsage] skip estimation: missing provider or model', {
      convId,
      providerId,
      modelId,
    })
    return
  }

  const req = new main.EstimateContextUsageRequest({
    conversationId: convId,
    messages: messages.map((m) => new models.Message({ role: m.role, content: m.content })),
    providerId,
    modelId,
  })

  const result = await WailsApp.EstimateContextUsage(req)

  setContextUsage(convId, {
    usedTokens: result.usedTokens,
    maxTokens: result.maxTokens,
    ratio: result.ratio,
    approximate: result.approximate,
    authoritative: false,
  })
}

/**
 * 触发指定会话的上下文压缩，并在完成后刷新用量显示。
 */
export async function compressSession(args: {
  conversationId: string
  providerId: string
  modelId: string
}): Promise<void> {
  const { setCompressing } = useChatStore.getState()
  setCompressing(args.conversationId, true)

  try {
    await WailsApp.CompressSession(args)
  } finally {
    setCompressing(args.conversationId, false)
  }

  await recomputeUsage(args.conversationId)
}

/**
 * 注册上下文用量相关事件监听。
 * 返回的清理函数会移除本服务注册的所有监听。
 */
export function registerContextUsageListeners(): () => void {
  const { setAuthoritativeUsed } = useChatStore.getState()

  const removeConfidence = EventsOn(
    'chat:stream:confidence',
    (payload: {
      conversation_id?: string
      prompt_tokens?: number
    }) => {
      const convId = payload.conversation_id
      const tokens = payload.prompt_tokens
      if (convId && tokens && tokens > 0) {
        setAuthoritativeUsed(convId, tokens)
      }
    }
  )

  const removeAutoCompressed = EventsOn(
    'context:auto_compressed',
    (payload: { conversation_id?: string }) => {
      const convId = payload.conversation_id
      if (convId) {
        void recomputeUsage(convId)
      }
    }
  )

  const removeUsageRefresh = EventsOn(
    'context:usage_refresh',
    (payload: { conversation_id?: string }) => {
      const convId = payload.conversation_id
      if (convId) {
        void recomputeUsage(convId)
      }
    }
  )

  return () => {
    removeConfidence()
    removeAutoCompressed()
    removeUsageRefresh()
  }
}
