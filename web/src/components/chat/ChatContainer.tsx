import { useEffect, useRef } from 'react'
import { Virtuoso, type VirtuosoHandle } from 'react-virtuoso'
import { MessageBubble } from './MessageBubble'
import { TypingIndicator } from './TypingIndicator'
import { ContextUsageBar } from './ContextUsageBar'
import { CompressSessionButton } from './CompressSessionButton'
import type { ChatMessage } from '@/stores/chatStore'
import { useChatStore } from '@/stores/chatStore'
import { useSettingsStore } from '@/stores/settingsStore'
import { registerContextUsageListeners, recomputeUsage } from '@/services/contextUsageService'
import { Bot, Plus } from 'lucide-react'

interface ChatContainerProps {
  messages: ChatMessage[]
  isStreaming: boolean
  onStartNewConversation?: () => void
  onRetry?: (messageId: string) => void
  onReportCompliance?: (messageId: string, ruleID: string) => void
  onFollowupClick?: (text: string) => void
  conversationId?: string
  providerId?: string
  modelId?: string
}

const isTest = import.meta.env.VITEST === 'true'

/**
 * 消息列表滚动容器，使用 react-virtuoso 虚拟列表优化长会话性能。
 * 测试环境回退到普通 map 渲染（jsdom 不支持 ResizeObserver 布局计算）。
 * 自动滚动到底部（仅在用户已位于底部时）。
 */
export function ChatContainer({ messages, isStreaming, onStartNewConversation, onRetry, onReportCompliance, onFollowupClick, conversationId: conversationIdProp, providerId: providerIdProp, modelId: modelIdProp }: ChatContainerProps) {
  const virtuosoRef = useRef<VirtuosoHandle>(null)
  const bottomRef = useRef<HTMLDivElement>(null)

  const storeConversationId = useChatStore((s) => s.currentConversationId)
  const storeProviderId = useSettingsStore((s) => s.activeProviderId)
  const storeModelId = useSettingsStore((s) => s.activeModelId)
  const conversationProviderId = useChatStore(
    (s) => s.conversations.find((c) => c.id === (conversationIdProp ?? s.currentConversationId))?.providerId
  )
  const conversationModelId = useChatStore(
    (s) => s.conversations.find((c) => c.id === (conversationIdProp ?? s.currentConversationId))?.modelId
  )

  const conversationId = conversationIdProp ?? storeConversationId ?? undefined
  const providerId = conversationProviderId ?? providerIdProp ?? storeProviderId ?? undefined
  const modelId = conversationModelId ?? modelIdProp ?? storeModelId ?? undefined

  // 注册上下文用量相关事件监听
  useEffect(() => {
    const cleanup = registerContextUsageListeners()
    return cleanup
  }, [])

  // 进入会话时立即估算一次，保证首屏有值
  useEffect(() => {
    if (conversationId) {
      void recomputeUsage(conversationId)
    }
  }, [conversationId])

  // 新消息到达或流式输出时自动滚底
  useEffect(() => {
    if (isTest) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
    } else if (messages.length > 0) {
      virtuosoRef.current?.scrollToIndex({ index: messages.length - 1, behavior: 'smooth' })
    }
  }, [messages, isStreaming])

  if (messages.length === 0) {
    return (
      <div className="flex-1 overflow-y-auto px-4 py-2">
        <div className="max-w-4xl mx-auto flex flex-col items-center justify-center h-full min-h-[300px] text-muted-foreground">
          <div className="w-16 h-16 rounded-2xl bg-white/60 dark:bg-white/10 backdrop-blur border border-border/40 flex items-center justify-center mb-4 shadow-sm">
            <Bot size={32} className="text-primary/70" />
          </div>
          <h2 className="text-lg font-medium mb-2 text-foreground/80">健康信息助手</h2>
          <p className="text-sm text-center max-w-sm mb-4 text-muted-foreground">
            我是你的私人健康信息参考工具。我可以帮你了解症状相关信息、推荐就诊科室、管理家族健康档案。
            <br />
            <span className="text-xs opacity-70 mt-1 block">
              请注意：我不提供医疗诊断或治疗建议。如有健康疑虑，请咨询专业医生。
            </span>
          </p>
          {onStartNewConversation && (
            <button
              onClick={onStartNewConversation}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:opacity-90 transition-opacity shadow-sm"
            >
              <Plus size={16} />
              新建对话
            </button>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className={`flex flex-col flex-1 px-4 py-2 ${isTest ? 'overflow-y-auto' : 'overflow-hidden'}`}>
      <div className={`max-w-4xl mx-auto w-full ${isTest ? '' : 'flex-1 min-h-0'}`}>
        {isTest ? (
          <>
            {messages.map((msg) => (
              <MessageBubble
                key={msg.id}
                message={msg}
                onRetry={onRetry}
                onReportCompliance={onReportCompliance}
                onFollowupClick={onFollowupClick}
              />
            ))}
            {isStreaming && messages.length > 0 && messages[messages.length - 1]?.role !== 'assistant' && (
              <div className="flex gap-3 my-4">
                <div className="shrink-0 w-8 h-8 rounded-full bg-accent text-accent-foreground flex items-center justify-center">
                  <Bot size={16} />
                </div>
                <div className="bg-ai-bg dark:bg-ai-bg-dark border border-border rounded-2xl rounded-tl-sm px-4 py-3 shadow-sm">
                  <TypingIndicator />
                </div>
              </div>
            )}
            <div ref={bottomRef} />
          </>
        ) : (
          <Virtuoso
            ref={virtuosoRef}
            style={{ height: '100%' }}
            data={messages}
            followOutput="auto"
            itemContent={(_index, msg) => (
              <MessageBubble
                key={msg.id}
                message={msg}
                onRetry={onRetry}
                onReportCompliance={onReportCompliance}
                onFollowupClick={onFollowupClick}
              />
            )}
            components={{
              Footer: () =>
                isStreaming && messages.length > 0 && messages[messages.length - 1]?.role !== 'assistant' ? (
                  <div className="flex gap-3 my-4">
                    <div className="shrink-0 w-8 h-8 rounded-full bg-accent text-accent-foreground flex items-center justify-center">
                      <Bot size={16} />
                    </div>
                    <div className="bg-ai-bg dark:bg-ai-bg-dark border border-border rounded-2xl rounded-tl-sm px-4 py-3 shadow-sm">
                      <TypingIndicator />
                    </div>
                  </div>
                ) : null,
            }}
          />
        )}
      </div>

      {conversationId && providerId && modelId && (
        <div className="max-w-4xl mx-auto w-full flex items-center gap-3 px-1 pt-2">
          <ContextUsageBar conversationId={conversationId} />
          <CompressSessionButton conversationId={conversationId} providerId={providerId} modelId={modelId} />
        </div>
      )}
    </div>
  )
}
