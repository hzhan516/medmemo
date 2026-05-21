import { useEffect, useRef } from 'react'
import { MessageBubble } from './MessageBubble'
import { TypingIndicator } from './TypingIndicator'
import type { ChatMessage } from '@/stores/chatStore'
import { Bot, Plus } from 'lucide-react'

interface ChatContainerProps {
  messages: ChatMessage[]
  isStreaming: boolean
  onStartNewConversation?: () => void
  onRetry?: (messageId: string) => void
  onReportCompliance?: (messageId: string, ruleID: string) => void
}

/**
 * 消息列表滚动容器，自动滚动到底部。
 */
export function ChatContainer({ messages, isStreaming, onStartNewConversation, onRetry, onReportCompliance }: ChatContainerProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, isStreaming])

  return (
    <div className="flex-1 overflow-y-auto px-4 py-2">
      <div className="max-w-3xl mx-auto">
        {messages.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full min-h-[300px] text-muted-foreground">
            <div className="w-16 h-16 rounded-2xl bg-accent flex items-center justify-center mb-4">
              <Bot size={32} className="text-accent-foreground" />
            </div>
            <h2 className="text-lg font-medium mb-2">MedMemo 健康助手</h2>
            <p className="text-sm text-center max-w-sm mb-4">
              我是你的私人健康记忆助手。我可以帮你了解症状、推荐科室、管理家族健康档案。
              <br />
              <span className="text-xs opacity-70 mt-1 block">
                请注意：我不提供医疗诊断或治疗建议。
              </span>
            </p>
            {onStartNewConversation && (
              <button
                onClick={onStartNewConversation}
                className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:opacity-90 transition-opacity"
              >
                <Plus size={16} />
                新建对话
              </button>
            )}
          </div>
        )}

        {messages.map((msg) => (
          <MessageBubble key={msg.id} message={msg} onRetry={onRetry} onReportCompliance={onReportCompliance} />
        ))}

        {isStreaming && messages[messages.length - 1]?.role !== 'assistant' && (
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
      </div>
    </div>
  )
}
