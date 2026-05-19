import { useState, useRef, useCallback, useEffect } from 'react'
import { Send, Loader2 } from 'lucide-react'

interface ChatInputProps {
  onSend: (message: string) => void
  isLoading?: boolean
  placeholder?: string
}

/**
 * 底部聊天输入区域。
 * 最小高度 120px，最大 300px，支持自动扩展。
 * 快捷键：Enter 发送、Shift+Enter 换行、Escape 清空。
 * 空输入框时按 Up Arrow 可编辑上一条消息 [Issue#032]。
 */
export function ChatInput({
  onSend,
  isLoading = false,
  placeholder = '输入你的健康问题...',
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
    if (!trimmed || isLoading) return
    onSend(trimmed)
    setContent('')
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto'
    }
  }, [content, isLoading, onSend])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        handleSend()
      }
      if (e.key === 'Escape') {
        setContent('')
      }
    },
    [handleSend]
  )

  return (
    <div className="shrink-0 border-t border-border bg-background px-4 py-3">
      <div className="flex items-end gap-2 max-w-3xl mx-auto">
        <div className="flex-1 relative">
          <textarea
            ref={textareaRef}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            rows={1}
            className="
              w-full min-h-[48px] max-h-[220px] resize-none
              rounded-xl border border-input bg-background px-4 py-3 pr-10
              text-sm placeholder:text-muted-foreground
              focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent
              transition-all
            "
            disabled={isLoading}
          />
          <div className="absolute right-3 bottom-3 text-[10px] text-muted-foreground select-none">
            {content.length > 0 && (
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

        <button
          onClick={handleSend}
          disabled={!content.trim() || isLoading}
          className={`
            shrink-0 w-10 h-10 rounded-xl flex items-center justify-center
            transition-all
            ${content.trim() && !isLoading
              ? 'bg-primary text-primary-foreground hover:opacity-90 shadow-sm'
              : 'bg-muted text-muted-foreground cursor-not-allowed'
            }
          `}
          aria-label="发送"
        >
          {isLoading ? <Loader2 size={18} className="animate-spin" /> : <Send size={18} />}
        </button>
      </div>

      <div className="text-center mt-1.5">
        <span className="text-[10px] text-muted-foreground">
          Enter 发送 · Shift+Enter 换行 · Escape 取消
        </span>
      </div>
    </div>
  )
}
