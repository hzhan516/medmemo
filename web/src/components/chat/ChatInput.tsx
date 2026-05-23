import { useState, useRef, useCallback, useEffect } from 'react'
import { Send, Square } from 'lucide-react'

interface ChatInputProps {
  onSend: (message: string) => void
  onStop?: () => void
  onNewConversation?: () => void
  isLoading?: boolean
  placeholder?: string
  blockedByEmergency?: boolean
  lastUserMessage?: string
}

/**
 * 底部聊天输入区域。
 * 最小高度 56px，最大 120px，支持自动扩展。
 * 快捷键：Enter 发送、Shift+Enter 换行、Escape 清空。
 * 支持 /new 命令新建会话。
 * 空输入框时按 Up Arrow 可编辑上一条消息 [Issue#032]。
 */
export function ChatInput({
  onSend,
  onStop,
  onNewConversation,
  isLoading = false,
  placeholder,
  blockedByEmergency = false,
  lastUserMessage,
}: ChatInputProps) {
  const [content, setContent] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  // 自动调整高度
  useEffect(() => {
    const el = textareaRef.current
    if (!el) return
    el.style.height = 'auto'
    const newHeight = Math.min(Math.max(el.scrollHeight, 48), 220)
    el.style.height = `${newHeight}px`
  }, [content])

  const handleSend = useCallback(() => {
    const trimmed = content.trim()
    if (!trimmed || isLoading || blockedByEmergency) return

    // /new 命令：新建会话
    if (trimmed === '/new') {
      onNewConversation?.()
      setContent('')
      return
    }

    onSend(trimmed)
    setContent('')
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto'
    }
  }, [content, isLoading, onSend, onNewConversation, blockedByEmergency])

  const handleStop = useCallback(() => {
    if (!isLoading || !onStop) return
    onStop()
  }, [isLoading, onStop])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        if (isLoading && onStop) {
          handleStop()
        } else {
          handleSend()
        }
      }
      if (e.key === 'Escape') {
        setContent('')
      }
      if (e.key === 'ArrowUp' && content === '' && lastUserMessage) {
        e.preventDefault()
        setContent(lastUserMessage)
      }
    },
    [handleSend, handleStop, isLoading, onStop, content, lastUserMessage]
  )

  const displayPlaceholder = placeholder ?? (isLoading ? 'AI 正在生成回复...' : '输入你的健康问题，或输入 /new 新建会话')

  return (
    <div className="shrink-0 border-t border-border bg-background px-4 py-3">
      <div className="flex items-end gap-2 max-w-3xl mx-auto">
        <div className="flex-1 relative">
          <textarea
            ref={textareaRef}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={displayPlaceholder}
            rows={1}
            className="
              w-full min-h-[56px] max-h-[120px] resize-none
              rounded-xl border border-input bg-background px-4 py-3 pr-10
              text-sm placeholder:text-muted-foreground
              focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent
              transition-all
            "
            disabled={isLoading || blockedByEmergency}
          />
          <div className="absolute right-3 bottom-3 text-[10px] text-muted-foreground select-none">
            {content.length > 0 && !isLoading && (
              <span>
                {content.length > 2000 ? (
                  <span className="text-destructive">{content.length}</span>
                ) : (
                  content.length
                )}
                /2000
              </span>
            )}
          </div>
        </div>

        {blockedByEmergency ? (
          <button
            disabled
            className="
              shrink-0 w-10 h-10 rounded-xl flex items-center justify-center
              bg-muted text-muted-foreground cursor-not-allowed
            "
            aria-label="请先确认紧急警告"
          >
            <Send size={18} />
          </button>
        ) : isLoading ? (
          <button
            onClick={handleStop}
            className="
              shrink-0 w-10 h-10 rounded-xl flex items-center justify-center
              bg-destructive text-destructive-foreground
              hover:opacity-90 transition-all shadow-sm
            "
            aria-label="停止生成"
            title="停止生成"
          >
            <Square size={14} fill="currentColor" />
          </button>
        ) : (
          <button
            onClick={handleSend}
            disabled={!content.trim()}
            className={`
              shrink-0 w-10 h-10 rounded-xl flex items-center justify-center
              transition-all
              ${content.trim()
                ? 'bg-primary text-primary-foreground hover:opacity-90 shadow-sm'
                : 'bg-muted text-muted-foreground cursor-not-allowed'
              }
            `}
            aria-label="发送"
          >
            <Send size={18} />
          </button>
        )}
      </div>

      <div className="text-center mt-1.5">
        <span className="text-[10px] text-muted-foreground">
          {blockedByEmergency
            ? '请先确认紧急症状警告后方可继续'
            : isLoading
              ? 'Enter 停止生成 · 正在接收回复'
              : 'Enter 发送 · Shift+Enter 换行 · Escape 取消 · ↑ 编辑上一条 · /new 新建会话'}
        </span>
      </div>
    </div>
  )
}
