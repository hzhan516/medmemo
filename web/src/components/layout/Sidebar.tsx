import { useState, useCallback, useEffect } from 'react'
import { Plus, PanelLeftClose, PanelLeft } from 'lucide-react'
import { SidebarItem } from './SidebarItem'
import { ResizableHandle } from './ResizableHandle'

interface ConversationItem {
  id: string
  title: string
  preview?: string
  timestamp?: string
  unread?: number
}

interface SidebarProps {
  conversations: ConversationItem[]
  activeId?: string | null
  onSelect?: (id: string) => void
  onNewConversation?: () => void
  onRename?: (id: string, title: string) => void
  onDelete?: (id: string) => void
}

const MIN_WIDTH = 200
const MAX_WIDTH = 400
const DEFAULT_WIDTH = 280
const COLLAPSE_BREAKPOINT = 768

/**
 * 左侧会话列表侧边栏。
 * 支持拖拽调整宽度（200-400px），窗口 < 768px 时自动收起为图标导航栏。
 */
export function Sidebar({
  conversations,
  activeId,
  onSelect,
  onNewConversation,
  onRename,
  onDelete,
}: SidebarProps) {
  const [width, setWidth] = useState(DEFAULT_WIDTH)
  const [isCollapsed, setIsCollapsed] = useState(false)
  const [isMobile, setIsMobile] = useState(false)

  useEffect(() => {
    const checkMobile = () => {
      const mobile = window.innerWidth < COLLAPSE_BREAKPOINT
      setIsMobile(mobile)
      if (mobile) {
        setIsCollapsed(true)
      }
    }
    checkMobile()
    window.addEventListener('resize', checkMobile)
    return () => window.removeEventListener('resize', checkMobile)
  }, [])

  const handleResize = useCallback((newWidth: number) => {
    const clamped = Math.max(MIN_WIDTH, Math.min(MAX_WIDTH, newWidth))
    setWidth(clamped)
  }, [])

  if (isCollapsed) {
    return (
      <div className="shrink-0 w-14 border-r border-border bg-background flex flex-col items-center py-3 gap-3">
        <button
          onClick={() => setIsCollapsed(false)}
          className="p-2 rounded-md hover:bg-accent transition-colors"
          aria-label="展开侧边栏"
        >
          <PanelLeft size={20} />
        </button>
        <button
          onClick={onNewConversation}
          className="p-2 rounded-md bg-primary text-primary-foreground hover:opacity-90 transition-opacity"
          aria-label="新建会话"
        >
          <Plus size={18} />
        </button>
      </div>
    )
  }

  return (
    <>
      <aside
        className="shrink-0 flex flex-col border-r border-border bg-background"
        style={{ width: isMobile ? '100%' : width }}
      >
        {/* 侧边栏头部 */}
        <div className="flex items-center justify-between px-3 py-3 border-b border-border">
          <button
            onClick={onNewConversation}
            className="flex items-center gap-2 flex-1 justify-center px-3 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:opacity-90 transition-opacity"
          >
            <Plus size={16} />
            新建对话
          </button>
          {!isMobile && (
            <button
              onClick={() => setIsCollapsed(true)}
              className="ml-2 p-2 rounded-md hover:bg-accent transition-colors"
              aria-label="收起侧边栏"
            >
              <PanelLeftClose size={18} />
            </button>
          )}
        </div>

        {/* 会话列表 */}
        <div className="flex-1 overflow-y-auto px-2 py-2 space-y-1">
          {conversations.length === 0 && (
            <div className="text-center text-xs text-muted-foreground py-8">
              暂无会话，点击上方按钮开始
            </div>
          )}
          {conversations.map((conv) => (
            <SidebarItem
              key={conv.id}
              id={conv.id}
              title={conv.title}
              preview={conv.preview}
              timestamp={conv.timestamp}
              unread={conv.unread}
              isActive={conv.id === activeId}
              onClick={() => onSelect?.(conv.id)}
              onRename={onRename}
              onDelete={onDelete}
            />
          ))}
        </div>
      </aside>

      {!isMobile && <ResizableHandle onResize={handleResize} />}
    </>
  )
}
