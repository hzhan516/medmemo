import { Moon, Sun, AlertTriangle, X } from 'lucide-react'
import { useTheme } from '@/hooks/useTheme'
import { useSettingsStore } from '@/stores/settingsStore'
import { useState } from 'react'

/**
 * 顶部标题栏（56px）+ 合规提示条（≤40px）。
 * 遵循 AGENTS.md 6.2 节布局与颜色规范。
 */
export function Header() {
  const { theme, setTheme } = useTheme()
  const { complianceNoticeDismissed, dismissComplianceNotice } = useSettingsStore()
  const [showCompliance, setShowCompliance] = useState(!complianceNoticeDismissed)

  const handleDismiss = () => {
    setShowCompliance(false)
    dismissComplianceNotice()
  }

  return (
    <header className="flex flex-col shrink-0">
      {/* 顶部标题栏 */}
      <div className="h-14 flex items-center justify-between px-4 border-b border-border bg-background/95 backdrop-blur">
        <div className="flex items-center gap-2">
          <h1 className="text-lg font-semibold tracking-tight">MedMemo</h1>
          <span className="text-xs text-muted-foreground hidden sm:inline">
            私人健康记忆助手
          </span>
        </div>

        <div className="flex items-center gap-1">
          {/* 主题切换 */}
          <button
            onClick={() => {
              const cycle: Array<'light' | 'dark' | 'system'> = ['light', 'dark', 'system']
              const idx = cycle.indexOf(theme)
              setTheme(cycle[(idx + 1) % cycle.length])
            }}
            className="p-2 rounded-md hover:bg-accent transition-colors"
            aria-label="切换主题"
            title={`当前主题: ${theme === 'light' ? '亮色' : theme === 'dark' ? '暗色' : '跟随系统'}`}
          >
            {theme === 'dark' ? <Sun size={18} /> : theme === 'light' ? <Moon size={18} /> : <Sun size={18} className="opacity-70" />}
          </button>
        </div>
      </div>

      {/* 合规提示条 */}
      {showCompliance && (
        <div className="flex items-center gap-2 px-4 py-2 bg-primary/10 text-xs text-primary">
          <AlertTriangle size={14} className="shrink-0 text-primary" />
          <span className="flex-1">
            本工具仅提供健康信息参考，不诊断、不治疗，紧急情况请拨打120。
          </span>
          <button
            onClick={handleDismiss}
            className="p-0.5 rounded hover:bg-primary/20 transition-colors"
            aria-label="关闭提示"
          >
            <X size={14} />
          </button>
        </div>
      )}
    </header>
  )
}
