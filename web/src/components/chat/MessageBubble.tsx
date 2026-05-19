import { User, Bot, AlertCircle } from 'lucide-react'
import type { ChatMessage } from '@/stores/chatStore'
import { MarkdownRenderer } from '@/components/markdown/MarkdownRenderer'

interface MessageBubbleProps {
  message: ChatMessage
}

/**
 * 消息气泡组件。
 * 用户消息：蓝色渐变，右侧对齐，圆角 16px。
 * AI 消息：白色/暗色背景，左侧对齐。
 * 系统提示：浅色背景，居中，13px 小字。
 */
export function MessageBubble({ message }: MessageBubbleProps) {
  const { role, content, isStreaming } = message

  if (role === 'system') {
    return (
      <div className="flex justify-center my-3">
        <div className="flex items-center gap-1.5 px-4 py-2 rounded-full bg-system-gray text-xs text-muted-foreground">
          <AlertCircle size={12} />
          <span>{content}</span>
        </div>
      </div>
    )
  }

  const isUser = role === 'user'

  return (
    <div
      className={`flex gap-3 my-4 ${isUser ? 'flex-row-reverse' : 'flex-row'}`}
    >
      {/* 头像 */}
      <div
        className={`
          shrink-0 w-8 h-8 rounded-full flex items-center justify-center
          ${isUser
            ? 'bg-gradient-to-br from-user-blue to-user-blue-dark text-white'
            : 'bg-accent text-accent-foreground'
          }
        `}
      >
        {isUser ? <User size={16} /> : <Bot size={16} />}
      </div>

      {/* 气泡 */}
      <div
        className={`
          max-w-[80%] px-4 py-3 text-sm leading-relaxed
          ${isUser
            ? 'bg-gradient-to-br from-user-blue to-user-blue-dark text-white rounded-2xl rounded-tr-sm'
            : 'bg-ai-bg dark:bg-ai-bg-dark text-ai-text dark:text-gray-200 border border-border rounded-2xl rounded-tl-sm shadow-sm'
          }
        `}
      >
        {isUser ? (
          <div className="whitespace-pre-wrap break-words">
            {content}
            {isStreaming && (
              <span className="inline-block w-1.5 h-4 ml-0.5 bg-current opacity-50 animate-pulse" />
            )}
          </div>
        ) : (
          <div className="break-words">
            <MarkdownRenderer content={content} />
            {isStreaming && content.length > 0 && (
              <span className="inline-block w-1.5 h-4 ml-0.5 bg-current opacity-50 animate-pulse" />
            )}
          </div>
        )}
      </div>
    </div>
  )
}
