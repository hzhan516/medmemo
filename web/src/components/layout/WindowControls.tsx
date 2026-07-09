import { useState, useCallback, useEffect } from 'react'
import {
  WindowMinimise,
  WindowToggleMaximise,
  WindowIsMaximised,
  Quit,
} from '@wails/runtime/runtime'
import { Minus, Square, X, Copy } from 'lucide-react'

interface WindowControlsProps {
  className?: string
}

/**
 * 应用级窗口控制按钮（最小化、最大化/还原、关闭）。
 * 使用 Wails runtime 窗口 API，替代系统原生标题栏按钮。
 * 仅应在 frameless 模式下渲染（Linux-only MVP）。
 */
export function WindowControls({ className }: WindowControlsProps) {
  const [isMaximised, setIsMaximised] = useState(false)

  const sync = useCallback(async () => {
    try {
      setIsMaximised(await WindowIsMaximised())
    } catch {
      setIsMaximised(false)
    }
  }, [])

  useEffect(() => {
    sync()
  }, [sync])

  // 点击最大化/还原后短延迟重新同步，兼容 Linux window manager 状态更新
  const handleToggle = useCallback(async () => {
    try {
      WindowToggleMaximise()
      // 延迟 150ms 等待 window manager 完成状态切换
      setTimeout(() => { sync() }, 150)
    } catch {
      // 静默忽略
    }
  }, [sync])

  // 监听 window resize 同步最大化状态
  useEffect(() => {
    const onResize = () => { sync() }
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [sync])

  return (
    <div className={`flex items-center ${className ?? ''}`}>
      {/* 最小化 */}
      <button
        onClick={() => WindowMinimise()}
        className="w-11 h-8 flex items-center justify-center rounded-md hover:bg-white/50 dark:hover:bg-white/10 transition-colors focus:outline-none focus:ring-2 focus:ring-ring"
        aria-label="最小化窗口"
        title="最小化"
      >
        <Minus size={14} />
      </button>

      {/* 最大化 / 还原 */}
      <button
        onClick={handleToggle}
        className="w-11 h-8 flex items-center justify-center rounded-md hover:bg-white/50 dark:hover:bg-white/10 transition-colors focus:outline-none focus:ring-2 focus:ring-ring"
        aria-label={isMaximised ? '还原窗口' : '最大化窗口'}
        title={isMaximised ? '还原' : '最大化'}
      >
        {isMaximised ? <Copy size={13} /> : <Square size={13} />}
      </button>

      {/* 关闭 */}
      <button
        onClick={() => Quit()}
        className="w-11 h-8 flex items-center justify-center rounded-md hover:bg-destructive hover:text-destructive-foreground transition-colors focus:outline-none focus:ring-2 focus:ring-destructive"
        aria-label="关闭应用"
        title="关闭"
      >
        <X size={16} />
      </button>
    </div>
  )
}
