import { Download, X, ArrowRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { UpdateInfo } from '@/hooks/useUpdate'

interface UpdateBannerProps {
  info: UpdateInfo | null
  onShowDetails: () => void
  onDismiss: () => void
}

/**
 * 顶部更新提示横幅。
 * 非侵入式展示，用户可点击展开详情或关闭。
 */
export function UpdateBanner({ info, onShowDetails, onDismiss }: UpdateBannerProps) {
  if (!info) return null

  return (
    <div className="flex items-center justify-between gap-3 bg-primary/10 px-4 py-2 text-sm">
      <div className="flex items-center gap-2 min-w-0">
        <Download size={14} className="shrink-0 text-primary" />
        <span className="truncate">
          <span className="font-medium">MedMemo {info.display_version || info.version}</span> 已发布
          {info.prerelease && (
            <span className="ml-1 text-amber-600 dark:text-amber-400">（测试版）</span>
          )}
          {info.mandatory && (
            <span className="ml-1 text-red-600 dark:text-red-400">（安全补丁，建议立即更新）</span>
          )}
        </span>
      </div>
      <div className="flex items-center gap-2 shrink-0">
        <Button variant="ghost" size="sm" className="h-6 px-2 text-xs" onClick={onShowDetails}>
          查看详情
          <ArrowRight size={12} className="ml-1" />
        </Button>
        <Button variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={onDismiss}>
          <X size={14} />
        </Button>
      </div>
    </div>
  )
}
