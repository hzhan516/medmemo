import { useState } from 'react'
import { X, Download, SkipForward, ExternalLink, Loader2, CheckCircle2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { UpdateInfo } from '@/hooks/useUpdate'

interface UpdateModalProps {
  info: UpdateInfo | null
  isDownloading: boolean
  downloadProgress: { downloaded: number; total: number } | null
  downloadPath: string
  error: string
  onDownload: () => void
  onApply: () => void
  onSkip: () => void
  onDismiss: () => void
  onOpenDownloadPage: (url: string) => void
}

/**
 * 更新弹窗组件。
 * 展示新版本信息、发布说明，提供下载/安装/跳过操作。
 * 支持 macOS 场景下的浏览器引导下载。
 */
export function UpdateModal({
  info,
  isDownloading,
  downloadProgress,
  downloadPath,
  error,
  onDownload,
  onApply,
  onSkip,
  onDismiss,
  onOpenDownloadPage,
}: UpdateModalProps) {
  const [showDetails, setShowDetails] = useState(false)

  if (!info) return null

  const isMacOS = navigator.platform.toLowerCase().includes('mac')
  const progressPercent =
    downloadProgress && downloadProgress.total > 0
      ? Math.round((downloadProgress.downloaded / downloadProgress.total) * 100)
      : 0

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-lg rounded-lg border border-border bg-background shadow-lg">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border p-4">
          <div className="flex items-center gap-2">
            {info.mandatory ? (
              <span className="inline-flex items-center rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-800 dark:bg-red-900/30 dark:text-red-300">
                强制更新
              </span>
            ) : (
              <span className="inline-flex items-center rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-800 dark:bg-blue-900/30 dark:text-blue-300">
                新版本可用
              </span>
            )}
            <h2 className="text-lg font-semibold">
              MedMemo {info.display_version || info.version}
              {info.prerelease && (
                <span className="ml-2 inline-flex items-center rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-800 dark:bg-amber-900/30 dark:text-amber-300">
                  测试版
                </span>
              )}
            </h2>
          </div>
          <button
            onClick={onDismiss}
            className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
          >
            <X size={18} />
          </button>
        </div>

        {/* Content */}
        <div className="space-y-4 p-4">
          <p className="text-sm text-muted-foreground">{info.name}</p>

          {/* 发布说明 */}
          <div className="rounded-md bg-muted p-3">
            <button
              onClick={() => setShowDetails(!showDetails)}
              className="text-xs font-medium text-muted-foreground hover:text-foreground transition-colors"
            >
              {showDetails ? '收起' : '查看'} 更新内容
            </button>
            {showDetails && (
              <div className="mt-2 text-sm text-foreground whitespace-pre-wrap max-h-48 overflow-y-auto">
                {info.body || '暂无更新说明'}
              </div>
            )}
          </div>

          {/* 下载进度 */}
          {isDownloading && downloadProgress && (
            <div className="space-y-2">
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>下载中...</span>
                <span>
                  {formatBytes(downloadProgress.downloaded)} / {formatBytes(downloadProgress.total)}
                </span>
              </div>
              <div className="h-2 w-full rounded-full bg-muted">
                <div
                  className="h-2 rounded-full bg-primary transition-all"
                  style={{ width: `${progressPercent}%` }}
                />
              </div>
            </div>
          )}

          {/* 下载完成 */}
          {downloadPath && !isDownloading && (
            <div className="flex items-center gap-2 rounded-md bg-green-50 p-3 text-sm text-green-800 dark:bg-green-900/20 dark:text-green-300">
              <CheckCircle2 size={16} />
              <span>下载完成</span>
            </div>
          )}

          {/* 错误提示 */}
          {error && (
            <div className="rounded-md bg-red-50 p-3 text-sm text-red-800 dark:bg-red-900/20 dark:text-red-300">
              {error}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 border-t border-border p-4">
          {!info.mandatory && (
            <Button variant="ghost" size="sm" onClick={onSkip} disabled={isDownloading}>
              <SkipForward size={14} className="mr-1" />
              跳过此版本
            </Button>
          )}

          {isMacOS ? (
            <Button
              size="sm"
              onClick={() => {
                const url = `https://github.com/hzhan516/medmemo/releases/download/${info.version}/MedMemo.dmg`
                onOpenDownloadPage(url)
                onDismiss()
              }}
            >
              <ExternalLink size={14} className="mr-1" />
              前往下载
            </Button>
          ) : downloadPath ? (
            <Button size="sm" onClick={onApply}>
              <CheckCircle2 size={14} className="mr-1" />
              安装更新
            </Button>
          ) : (
            <Button size="sm" onClick={onDownload} disabled={isDownloading}>
              {isDownloading ? (
                <Loader2 size={14} className="mr-1 animate-spin" />
              ) : (
                <Download size={14} className="mr-1" />
              )}
              {isDownloading ? '下载中...' : '下载更新'}
            </Button>
          )}
        </div>
      </div>
    </div>
  )
}
