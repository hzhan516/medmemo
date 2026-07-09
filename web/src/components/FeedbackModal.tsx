import { useState, useEffect, useCallback } from 'react'
import { Bug, Shield, ExternalLink, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useWails } from '@/hooks/useWails'
import type { feedback } from '@wails/go/models'

interface FeedbackModalProps {
  isOpen: boolean
  errorInfo?: string
  onClose: () => void
  onSubmit: (description: string) => Promise<void>
}

/**
 * 问题反馈弹窗。
 * 展示系统信息、收集问题描述，并生成 GitHub Issue 预填链接。
 */
export function FeedbackModal({ isOpen, errorInfo, onClose, onSubmit }: FeedbackModalProps) {
  const { collectSystemInfo } = useWails()
  const [systemInfo, setSystemInfo] = useState<feedback.SystemInfo | null>(null)
  const [description, setDescription] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    if (isOpen) {
      let cancelled = false
      collectSystemInfo().then((info) => {
        if (!cancelled) setSystemInfo(info)
      })
      return () => { cancelled = true }
    }
  }, [isOpen, collectSystemInfo])

  // 自动预填错误信息到描述框
  useEffect(() => {
    if (errorInfo) {
      setDescription(`## 自动捕获的错误\n\n\`\`\`\n${errorInfo}\n\`\`\`\n\n## 补充说明\n\n`)
    } else {
      setDescription('')
    }
  }, [errorInfo])

  const handleSubmit = useCallback(async () => {
    setIsSubmitting(true)
    try {
      await onSubmit(description)
    } finally {
      setIsSubmitting(false)
    }
  }, [description, onSubmit])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-lg rounded-xl border border-border bg-background shadow-xl flex flex-col max-h-[85vh]">
        {/* Header */}
        <div className="flex items-center gap-3 px-5 py-4 border-b border-border">
          <div className="p-2 rounded-lg bg-primary/10">
            <Bug size={20} className="text-primary" />
          </div>
          <div>
            <h2 className="text-base font-semibold">问题反馈</h2>
            <p className="text-xs text-muted-foreground">帮助我们改进 MedMemo</p>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-5 py-4 space-y-4">
          {/* 系统信息卡片 */}
          {systemInfo && (
            <div className="rounded-lg bg-muted/50 border border-border/50 p-3 space-y-1.5">
              <p className="text-xs font-medium text-muted-foreground mb-2">系统信息（自动收集）</p>
              <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
                <span className="text-muted-foreground">App 版本</span>
                <span className="text-foreground font-medium">{systemInfo.app_version}</span>
                <span className="text-muted-foreground">操作系统</span>
                <span className="text-foreground font-medium">{systemInfo.os} / {systemInfo.arch}</span>
                <span className="text-muted-foreground">Go 版本</span>
                <span className="text-foreground font-medium">{systemInfo.go_version}</span>
                {systemInfo.build_time && (
                  <>
                    <span className="text-muted-foreground">构建时间</span>
                    <span className="text-foreground font-medium">{systemInfo.build_time}</span>
                  </>
                )}
              </div>
            </div>
          )}

          {/* 隐私提示 */}
          <div className="flex items-start gap-2 rounded-lg bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-900 p-3">
            <Shield size={14} className="text-blue-600 dark:text-blue-400 mt-0.5 shrink-0" />
            <p className="text-xs text-blue-700 dark:text-blue-300 leading-relaxed">
              报告中<strong>不包含</strong>任何对话内容、个人健康信息或 API 凭据。
              您可以在浏览器中编辑内容后再提交。
            </p>
          </div>

          {/* 问题描述 */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium">问题描述</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="请描述遇到的问题，例如：点击发送后无响应、模型切换失败等…"
              className="w-full min-h-[120px] max-h-[200px] px-3 py-2 text-sm rounded-lg bg-muted border border-border resize-y outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
            />
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-5 py-4 border-t border-border">
          <Button variant="ghost" size="sm" onClick={onClose} disabled={isSubmitting}>
            取消
          </Button>
          <Button
            size="sm"
            onClick={handleSubmit}
            disabled={isSubmitting}
            className="gap-1.5"
          >
            {isSubmitting ? (
              <Loader2 size={14} className="animate-spin" />
            ) : (
              <ExternalLink size={14} />
            )}
            {isSubmitting ? '正在打开…' : '生成报告并打开 GitHub'}
          </Button>
        </div>
      </div>
    </div>
  )
}
