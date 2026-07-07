import { cn } from '@/lib/utils'
import { useChatStore } from '@/stores/chatStore'
import { formatContextLabel, colorState } from '@/utils/contextUsage'

interface Props {
  conversationId: string
}

/**
 * 会话上下文用量进度条组件。
 *
 * - 展示当前会话已用 token 占最大上下文窗口的比例。
 * - 根据用量比例切换正常/警告/危险颜色。
 * - 用量为估算值时显示 "≈" 标记。
 */
export function ContextUsageBar({ conversationId }: Props) {
  const usage = useChatStore((s) => s.contextUsageMap[conversationId])
  if (!usage) {
    return (
      <div className="flex items-center gap-2 text-xs" aria-hidden="true">
        <div className="h-1.5 flex-1 rounded-full bg-muted overflow-hidden">
          <div className="h-full w-full animate-pulse bg-muted-foreground/20" />
        </div>
        <span className="w-16 h-3 rounded bg-muted-foreground/20 animate-pulse" />
      </div>
    )
  }

  const pct = Math.min(Math.max(usage.ratio, 0), 1) * 100
  const state = colorState(usage.ratio)
  const label = formatContextLabel(usage.usedTokens, usage.maxTokens)

  return (
    <div
      className="flex items-center gap-2 text-xs"
      role="progressbar"
      aria-label="上下文用量"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(pct)}
    >
      <div className="h-1.5 flex-1 rounded-full bg-muted overflow-hidden">
        <div
          className={cn(
            'h-full transition-all duration-300',
            state === 'normal' && 'bg-emerald-500',
            state === 'warning' && 'bg-amber-500',
            state === 'critical' && 'bg-red-500'
          )}
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="whitespace-nowrap text-muted-foreground">
        {label}
        {usage.approximate && <span className="ml-0.5">≈</span>}
      </span>
    </div>
  )
}
