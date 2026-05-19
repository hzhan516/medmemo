import { useState, useCallback, useEffect, useMemo } from 'react'
import { Plus, PanelLeftClose, PanelLeft, Search, Trash2, ArrowLeft } from 'lucide-react'
import { SidebarItem } from './SidebarItem'
import { ResizableHandle } from './ResizableHandle'
import { useChatStore, type Conversation } from '@/stores/chatStore'
import { groupConversationsByTime, filterConversations } from '@/lib/conversationUtils'

interface SidebarProps {
  activeId?: string | null
  onSelect?: (id: string) => void
  onNewConversation?: () => void
}

const MIN_WIDTH = 200
const MAX_WIDTH = 400
const DEFAULT_WIDTH = 280
const COLLAPSE_BREAKPOINT = 768

const GROUP_LABELS: Record<string, string> = {
  pinned: '置顶',
  today: '今天',
  yesterday: '昨天',
  last7Days: '近 7 天',
  earlier: '更早',
}

/**
 * 左侧会话列表侧边栏。
 * 支持拖拽调整宽度（200-400px），窗口 < 768px 时自动收起为图标导航栏。
 * 新增：搜索过滤、时间分组、回收站视图。
 */
export function Sidebar({
  activeId,
  onSelect,
  onNewConversation,
}: SidebarProps) {
  const [width, setWidth] = useState(DEFAULT_WIDTH)
  const [isCollapsed, setIsCollapsed] = useState(false)
  const [isMobile, setIsMobile] = useState(false)

  const conversations = useChatStore((s) => s.conversations)
  const deletedConversations = useChatStore((s) => s.deletedConversations)
  const searchQuery = useChatStore((s) => s.searchQuery)
  const showTrash = useChatStore((s) => s.showTrash)
  const setSearchQuery = useChatStore((s) => s.setSearchQuery)
  const setShowTrash = useChatStore((s) => s.setShowTrash)
  const cleanupOldDeleted = useChatStore((s) => s.cleanupOldDeleted)

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

  // 进入回收站时清理过期数据
  useEffect(() => {
    if (showTrash) {
      cleanupOldDeleted()
    }
  }, [showTrash, cleanupOldDeleted])

  const handleResize = useCallback((newWidth: number) => {
    const clamped = Math.max(MIN_WIDTH, Math.min(MAX_WIDTH, newWidth))
    setWidth(clamped)
  }, [])

  const filtered = useMemo(() => {
    const source = showTrash ? deletedConversations : conversations
    return filterConversations(source, searchQuery)
  }, [conversations, deletedConversations, searchQuery, showTrash])

  const grouped = useMemo(() => {
    return groupConversationsByTime(filtered)
  }, [filtered])

  const renderGroup = (groupKey: string, items: Conversation[]) => {
    if (items.length === 0) return null
    return (
      <div key={groupKey} className="mb-2">
        <div className="px-3 py-1 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">
          {GROUP_LABELS[groupKey] || groupKey}
        </div>
        {items.map((conv) => (
          <SidebarItem
            key={conv.id}
            id={conv.id}
            title={conv.title}
            preview={conv.preview}
            timestamp={conv.updatedAt}
            unread={conv.unread}
            isActive={conv.id === activeId}
            isTrashView={showTrash}
            highlightQuery={searchQuery}
            onClick={() => onSelect?.(conv.id)}
          />
        ))}
      </div>
    )
  }

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
        <div className="flex items-center justify-between px-3 py-3 border-b border-border gap-2">
          <button
            onClick={onNewConversation}
            className="flex items-center gap-2 flex-1 justify-center px-3 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:opacity-90 transition-opacity"
            title="新建会话 (Ctrl+N)"
          >
            <Plus size={16} />
            新建对话
          </button>
          {!isMobile && (
            <button
              onClick={() => setIsCollapsed(true)}
              className="p-2 rounded-md hover:bg-accent transition-colors"
              aria-label="收起侧边栏"
            >
              <PanelLeftClose size={18} />
            </button>
          )}
        </div>

        {/* 搜索框或回收站返回 */}
        <div className="px-3 py-2 border-b border-border">
          {showTrash ? (
            <button
              onClick={() => setShowTrash(false)}
              className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              <ArrowLeft size={14} />
              返回会话列表
            </button>
          ) : (
            <div className="relative">
              <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="搜索会话..."
                className="w-full pl-8 pr-3 py-1.5 text-sm rounded-md bg-muted border-0 outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
              />
              {searchQuery && (
                <button
                  onClick={() => setSearchQuery('')}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground hover:text-foreground"
                >
                  清除
                </button>
              )}
            </div>
          )}
        </div>

        {/* 会话列表 */}
        <div className="flex-1 overflow-y-auto px-2 py-2">
          {filtered.length === 0 && (
            <div className="text-center text-xs text-muted-foreground py-8">
              {showTrash
                ? '回收站为空'
                : searchQuery
                  ? '未找到匹配的会话'
                  : '暂无会话，点击上方按钮开始'}
            </div>
          )}

          {showTrash
            ? filtered.map((conv) => (
                <SidebarItem
                  key={conv.id}
                  id={conv.id}
                  title={conv.title}
                  preview={conv.preview}
                  timestamp={conv.updatedAt}
                  unread={0}
                  isActive={conv.id === activeId}
                  isTrashView={true}
                  highlightQuery={searchQuery}
                  onClick={() => onSelect?.(conv.id)}
                />
              ))
            : (
              <>
                {renderGroup('pinned', grouped.pinned)}
                {renderGroup('today', grouped.today)}
                {renderGroup('yesterday', grouped.yesterday)}
                {renderGroup('last7Days', grouped.last7Days)}
                {renderGroup('earlier', grouped.earlier)}
              </>
            )}
        </div>

        {/* 底部回收站入口 */}
        {!showTrash && (
          <div className="px-3 py-2 border-t border-border">
            <button
              onClick={() => setShowTrash(true)}
              className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors w-full"
            >
              <Trash2 size={12} />
              回收站
              {deletedConversations.length > 0 && (
                <span className="ml-auto min-w-[18px] h-[18px] px-1 rounded-full bg-muted text-muted-foreground text-[10px] font-medium flex items-center justify-center">
                  {deletedConversations.length}
                </span>
              )}
            </button>
          </div>
        )}
      </aside>

      {!isMobile && <ResizableHandle onResize={handleResize} />}
    </>
  )
}
