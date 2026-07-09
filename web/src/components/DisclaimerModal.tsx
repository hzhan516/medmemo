import { useState, useCallback } from 'react'
import { AlertTriangle, Check, X } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface DisclaimerModalProps {
  text: string
  version: string
  onAccept: (version: string) => void
  onDecline: () => void
}

/**
 * 免责声明全屏弹窗组件。
 *
 * 首次启动或免责声明版本更新时展示，用户必须明确同意后方可进入主界面。
 * 弹窗不可通过点击背景或按 ESC 关闭，以确保用户完整阅读并做出主动选择。
 */
export function DisclaimerModal({ text, version, onAccept, onDecline }: DisclaimerModalProps) {
  const [scrolledToBottom, setScrolledToBottom] = useState(false)

  const handleScroll = useCallback((e: React.UIEvent<HTMLDivElement>) => {
    const el = e.currentTarget
    const threshold = 20 // 距离底部 20px 即视为已阅读到底
    const isBottom = el.scrollHeight - el.scrollTop - el.clientHeight < threshold
    if (isBottom && !scrolledToBottom) {
      setScrolledToBottom(true)
    }
  }, [scrolledToBottom])

  const handleAccept = useCallback(() => {
    onAccept(version)
  }, [onAccept, version])

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="w-full max-w-2xl max-h-[90vh] flex flex-col rounded-2xl bg-background border border-border shadow-2xl overflow-hidden">
        {/* 头部 */}
        <div className="flex items-center gap-3 px-6 py-4 border-b border-border bg-muted/40">
          <AlertTriangle className="w-6 h-6 text-amber-500 shrink-0" />
          <div className="flex-1">
            <h2 className="text-lg font-semibold text-foreground">免责声明与用户协议</h2>
            <p className="text-xs text-muted-foreground mt-0.5">
              在使用 MedMemo 之前，请您仔细阅读以下条款
            </p>
          </div>
          <span className="text-[11px] px-2 py-0.5 rounded-full bg-muted text-muted-foreground border border-border">
            {version}
          </span>
        </div>

        {/* 内容区（可滚动） */}
        <div
          className="flex-1 overflow-y-auto px-6 py-5 text-sm leading-relaxed text-foreground/90 whitespace-pre-line"
          onScroll={handleScroll}
        >
          {text}
        </div>

        {/* 底部操作栏 */}
        <div className="flex flex-col gap-3 px-6 py-4 border-t border-border bg-muted/30">
          {!scrolledToBottom && (
            <p className="text-xs text-center text-amber-600 dark:text-amber-400">
              请向下滚动阅读完整条款后，方可进行操作
            </p>
          )}

          <div className="flex items-center justify-end gap-3">
            <Button
              variant="outline"
              size="default"
              onClick={onDecline}
              className="gap-1.5"
            >
              <X size={16} />
              不同意并退出
            </Button>
            <Button
              variant="default"
              size="default"
              onClick={handleAccept}
              disabled={!scrolledToBottom}
              className="gap-1.5"
            >
              <Check size={16} />
              我已阅读并同意
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
