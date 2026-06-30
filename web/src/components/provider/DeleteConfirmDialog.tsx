import { AlertTriangle, X } from 'lucide-react'
import type { ProviderConfig } from '@/types/provider'

interface DeleteConfirmDialogProps {
  provider: ProviderConfig | null
  isActiveProvider: boolean
  open: boolean
  onClose: () => void
  onConfirm: () => void
}

/**
 * Provider 删除确认弹窗。
 * 若该 Provider 是当前活跃模型，显示警告并禁用删除。
 */
export function DeleteConfirmDialog({
  provider,
  isActiveProvider,
  open,
  onClose,
  onConfirm,
}: DeleteConfirmDialogProps) {
  if (!open || !provider) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div
        className="w-full max-w-sm mx-4 rounded-xl border border-border/60 bg-background/95 backdrop-blur-xl shadow-xl"
        role="alertdialog"
        aria-modal="true"
        data-testid="delete-confirm-dialog"
      >
        {/* 头部 */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-border/60">
          <h2 className="text-base font-semibold text-foreground">确认删除</h2>
          <button
            onClick={onClose}
            className="p-1.5 rounded-md hover:bg-accent transition-colors"
            aria-label="关闭"
          >
            <X className="w-4 h-4 text-muted-foreground" />
          </button>
        </div>

        <div className="px-5 py-4 space-y-4">
          <p className="text-sm text-foreground">
            确定要删除 Provider <span className="font-medium">「{provider.name}」</span> 吗？
            此操作不可恢复。
          </p>

          {isActiveProvider && (
            <div className="flex items-start gap-2 p-3 rounded-lg bg-destructive/5 border border-destructive/20">
              <AlertTriangle className="w-4 h-4 text-destructive shrink-0 mt-0.5" />
              <div className="text-sm text-destructive">
                该 Provider 当前为活跃模型，请先切换到其他模型后再删除。
              </div>
            </div>
          )}
        </div>

        {/* 底部按钮 */}
        <div className="flex items-center justify-end gap-2 px-5 py-4 border-t border-border">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-lg border border-border text-sm font-medium text-foreground hover:bg-accent transition-colors"
          >
            取消
          </button>
          <button
            onClick={onConfirm}
            disabled={isActiveProvider}
            className="px-4 py-2 rounded-lg bg-destructive text-destructive-foreground text-sm font-medium hover:bg-destructive/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            data-testid="delete-confirm-btn"
          >
            删除
          </button>
        </div>
      </div>
    </div>
  )
}
