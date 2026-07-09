import { useRef, useCallback, useEffect } from 'react'

interface ResizableHandleProps {
  onResize: (width: number) => void
}

/**
 * 侧边栏拖拽手柄，支持水平拖拽调整宽度。
 */
export function ResizableHandle({ onResize }: ResizableHandleProps) {
  const isDragging = useRef(false)
  const startX = useRef(0)
  const startWidth = useRef(0)
  const handleRef = useRef<HTMLDivElement>(null)

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isDragging.current) return
      const delta = e.clientX - startX.current
      onResize(startWidth.current + delta)
    },
    [onResize]
  )

  const handleMouseUp = useCallback(() => {
    if (isDragging.current) {
      isDragging.current = false
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }
  }, [handleMouseMove])

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      isDragging.current = true
      startX.current = e.clientX
      const sidebar = handleRef.current?.previousElementSibling as HTMLElement
      startWidth.current = sidebar?.offsetWidth || 280
      document.body.style.cursor = 'col-resize'
      document.body.style.userSelect = 'none'
      window.addEventListener('mousemove', handleMouseMove)
      window.addEventListener('mouseup', handleMouseUp)
    },
    [handleMouseMove, handleMouseUp]
  )

  // 组件卸载时清理
  useEffect(() => {
    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }
  }, [handleMouseMove, handleMouseUp])

  return (
    <div
      ref={handleRef}
      className="w-1 shrink-0 cursor-col-resize hover:bg-primary/20 active:bg-primary/40 transition-colors"
      onMouseDown={handleMouseDown}
    />
  )
}
