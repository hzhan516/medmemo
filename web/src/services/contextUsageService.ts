import * as WailsApp from '@wails/go/main/WailsApp'
import { EventsOn } from '@wails/runtime/runtime'
import { main, models } from '@wails/go/models'
import { useChatStore, type ChatMessage } from '@/stores/chatStore'
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
  const { messagesMap, setContextUsage, conversations } = useChatStore.getState()
  const { activeProviderId, activeModelId } = useSettingsStore.getState()
  const { providers, getEnabledProviders } = useProviderStore.getState()

  const messages = messagesMap[convId] ?? []
  const conv = conversations.find((c) => c.id === convId)

  let providerId = conv?.providerId ?? activeProviderId ?? ''
  let modelId = conv?.modelId ?? activeModelId ?? ''

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
  strategy?: string
  anchorCount?: number
  recentCount?: number
}): Promise<void> {
  const { setCompressing } = useChatStore.getState()
  setCompressing(args.conversationId, true)

  try {
    await WailsApp.CompressSession({
      conversationId: args.conversationId,
      providerId: args.providerId,
      modelId: args.modelId,
      strategy: args.strategy ?? '',
      anchorCount: args.anchorCount ?? 0,
      recentCount: args.recentCount ?? 0,
    })
  } finally {
    setCompressing(args.conversationId, false)
  }

  await refreshMessagesAfterCompression(args.conversationId)
  await recomputeUsage(args.conversationId)
}

async function refreshMessagesAfterCompression(convId: string): Promise<void> {
  try {
    const raw = await WailsApp.GetConversationMessages(convId)
    const messages: ChatMessage[] = raw.map((m) => ({
      id: m.id,
      role: m.role as ChatMessage['role'],
      content: m.content,
      timestamp: Number(m.timestamp),
    }))
    useChatStore.getState().setMessagesForConversation(convId, messages)
  } catch (err) {
    logger.warn('[contextUsage] failed to refresh messages after compression:', err)
  }
}

/**
 * 注册上下文用量相关事件监听。
 * 返回的清理函数会移除本服务注册的所有监听。
 */
export function registerContextUsageListeners(): () => void {
  const { setAuthoritativeUsed } = useChatStore.getState()
  let activeConvId: string | null = useChatStore.getState().currentConversationId
  let prevMessages: ChatMessage[] | null = null
  let prevStreaming = activeConvId
    ? useChatStore.getState().streamingIds.has(activeConvId)
    : false

  const unsubCurrent = useChatStore.subscribe((state) => {
    const id = state.currentConversationId
    if (id === activeConvId) return
    activeConvId = id
    if (id) {
      prevMessages = state.messagesMap[id] ?? null
      prevStreaming = state.streamingIds.has(id)
      void recomputeUsage(id)
    }
  })

  const unsubMessages = useChatStore.subscribe((state) => {
    if (!activeConvId) return
    const streaming = state.streamingIds.has(activeConvId)
    const msgs = state.messagesMap[activeConvId] ?? null
    const msgsChanged = msgs !== prevMessages
    const streamingEnded = prevStreaming && !streaming
    prevMessages = msgs
    prevStreaming = streaming

    // 流式进行中：prompt 侧基本不变、authoritative 会在结束后到达，
    // 跳过重估以避免逐 chunk 触发的组装风暴（C5-3 B-a）。
    if (streaming) return
    if (msgsChanged || streamingEnded) {
      void recomputeUsage(activeConvId)
    }
  })

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
    unsubCurrent()
    unsubMessages()
    removeConfidence()
    removeAutoCompressed()
    removeUsageRefresh()
  }
}
