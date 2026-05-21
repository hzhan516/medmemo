import { useState, useEffect } from 'react'
import { Info, X } from 'lucide-react'
import { useSettingsStore } from '@/stores/settingsStore'
import { useChatStore } from '@/stores/chatStore'

interface ComplianceBarProps {
  conversationId: string | null
}

/**
 * 会话顶部合规提示条组件。
 *
 * - 固定高度 32px，展示合规文案。
 * - 支持会话级关闭（当前会话有效，新建会话重新展示）。
 * - 暗色模式下背景色自适应为深蓝 #1E3A5F。
 * - 首次进入会话时从顶部滑入，时长 200ms。
 */
export function ComplianceBar({ conversationId }: ComplianceBarProps) {
  const mode = useSettingsStore((s) => s.complianceBarMode)
  const dismissedSessions = useChatStore((s) => s.dismissedBarSessions)
  const dismissForSession = useChatStore((s) => s.dismissComplianceBarForSession)
  const [visible, setVisible] = useState(false)
  const [mounted, setMounted] = useState(false)

  // 控制是否展示提示条
  const shouldShow =
    mode !== 'off' &&
    conversationId != null &&
    !dismissedSessions.includes(conversationId)

  // 挂载时触发动画：先 mount，再设置 visible 触发 CSS transition
  useEffect(() => {
    if (shouldShow) {
      setMounted(true)
      const timer = requestAnimationFrame(() => {
        setVisible(true)
      })
      return () => cancelAnimationFrame(timer)
    } else {
      setVisible(false)
      const timer = setTimeout(() => setMounted(false), 200)
      return () => clearTimeout(timer)
    }
  }, [shouldShow])

  if (!mounted) return null

  const handleDismiss = () => {
    if (conversationId) {
      dismissForSession(conversationId)
    }
  }

  return (
    <div
      role="note"
      aria-label="合规提示"
      className={`
        shrink-0 h-8 flex items-center justify-between px-3 text-xs
        bg-[#EBF5FF] text-[#2563EB] border-b border-[#BFDBFE]
        dark:bg-[#1E3A5F] dark:text-[#93C5FD] dark:border-[#1E3A5F]/60
        transition-transform duration-200 ease-out
        ${visible ? 'translate-y-0' : '-translate-y-full'}
      `}
    >
      <div className="flex items-center gap-2 min-w-0">
        <Info size={14} className="shrink-0" />
        <span className="truncate">
          AI 提供的信息仅供参考，不构成医疗诊断或治疗建议
        </span>
      </div>
      <button
        onClick={handleDismiss}
        aria-label="关闭提示"
        className="shrink-0 ml-2 p-0.5 rounded hover:bg-black/5 dark:hover:bg-white/10 transition-colors"
      >
        <X size={14} />
      </button>
    </div>
  )
}
