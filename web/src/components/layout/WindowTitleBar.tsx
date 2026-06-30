import { useCallback } from 'react'
import { WindowToggleMaximise } from '@wails/runtime/runtime'
import { WindowControls } from './WindowControls'

interface WindowTitleBarProps {
  showControls: boolean
}

/**
 * 全局应用 titlebar（frameless 模式下替代系统标题栏）。
 * 提供拖拽区域、应用标识和窗口控制按钮。
 * 高度 36px，与 macOS clinical workspace 风格一致。
 */
export function WindowTitleBar({ showControls }: WindowTitleBarProps) {
  const handleToggleMaximise = useCallback(() => {
    try {
      WindowToggleMaximise()
    } catch {
      // 静默忽略
    }
  }, [])

  return (
    <div className="app-drag-region h-9 shrink-0 mac-toolbar border-b border-white/20 dark:border-white/5 flex items-center select-none">
      {/* 应用标识 */}
      <div className="app-no-drag px-3 text-xs text-muted-foreground select-none">
        MedMemo
      </div>

      {/* 拖拽空白区域：双击最大化/还原 */}
      <div
        className="app-drag-region flex-1 self-stretch"
        onDoubleClick={handleToggleMaximise}
      />

      {/* 窗口控制按钮 */}
      {showControls && (
        <WindowControls className="app-no-drag" />
      )}
    </div>
  )
}
