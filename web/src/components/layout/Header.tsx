import { Moon, Sun } from 'lucide-react'
import { useTheme } from '@/hooks/useTheme'
import { ModelSwitcher } from '@/components/provider/ModelSwitcher'

/**
 * 顶部标题栏（56px）。
 * 遵循 AGENTS.md 6.2 节布局与颜色规范。
 */
export function Header() {
  const { theme, setTheme } = useTheme()

  return (
    <header className="h-14 flex items-center justify-between px-4 mac-toolbar border-b border-white/20 dark:border-white/5 shrink-0">
      <div className="flex items-center gap-2">
        <h1 className="text-lg font-semibold tracking-tight">MedMemo</h1>
        <span className="text-xs text-muted-foreground hidden sm:inline">
          私人健康记忆助手
        </span>
      </div>

      <div className="flex items-center gap-2">
        {/* 模型切换器 */}
        <ModelSwitcher />

        {/* 主题切换 */}
        <button
          onClick={() => {
            const cycle: Array<'light' | 'dark' | 'system'> = ['light', 'dark', 'system']
            const idx = cycle.indexOf(theme)
            setTheme(cycle[(idx + 1) % cycle.length])
          }}
          className="p-2 rounded-md hover:bg-white/50 dark:hover:bg-white/10 transition-colors"
          aria-label="切换主题"
          title={`当前主题: ${theme === 'light' ? '亮色' : theme === 'dark' ? '暗色' : '跟随系统'}`}
        >
          {theme === 'dark' ? <Sun size={18} /> : theme === 'light' ? <Moon size={18} /> : <Sun size={18} className="opacity-70" />}
        </button>
      </div>
    </header>
  )
}
