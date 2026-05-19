import { useState } from 'react'
import {
  MessageSquare,
  MoreHorizontal,
  Trash2,
  Edit3,
  Pin,
  PinOff,
  RotateCcw,
  AlertTriangle,
} from 'lucide-react'
import { useChatStore } from '@/stores/chatStore'
import { highlightText, validateConversationTitle, formatRelativeTime } from '@/lib/conversationUtils'

interface SidebarItemProps {
  id: string
  title: string
  preview?: string
  timestamp?: number
  unread?: number
  isActive?: boolean
  isTrashView?: boolean
  highlightQuery?: string
  onClick?: () => void
}

/**
 * 单条会话列表项。
 * 支持重命名（带校验）、删除/恢复、置顶/取消置顶、搜索高亮。
 */
export function SidebarItem({
  id,
  title,
  preview,
  timestamp,
  unread,
  isActive = false,
  isTrashView = false,
  highlightQuery = '',
  onClick,
}: SidebarItemProps) {
  const [isEditing, setIsEditing] = useState(false)
  const [editValue, setEditValue] = useState(title)
  const [editError, setEditError] = useState<string | null>(null)
  const [showMenu, setShowMenu] = useState(false)

  const softDeleteConversation = useChatStore((s) => s.softDeleteConversation)
  const permanentlyDeleteConversation = useChatStore((s) => s.permanentlyDeleteConversation)
  const restoreConversation = useChatStore((s) => s.restoreConversation)
  const pinConversation = useChatStore((s) => s.pinConversation)
  const unpinConversation = useChatStore((s) => s.unpinConversation)
  const isPinned = useChatStore((s) => s.conversations.find((c) => c.id === id)?.isPinned)

  const handleRename = () => {
    const trimmed = editValue.trim()
    const result = validateConversationTitle(trimmed)
    if (!result.valid) {
      setEditError(result.error || '标题不合法')
      return
    }
    if (trimmed && trimmed !== title) {
      useChatStore.getState().updateConversation(id, { title: trimmed })
    }
    setIsEditing(false)
    setEditError(null)
    setShowMenu(false)
  }

  const handleEditStart = () => {
    setEditValue(title)
    setEditError(null)
    setIsEditing(true)
    setShowMenu(false)
  }

  const handleSoftDelete = (e: React.MouseEvent) => {
    e.stopPropagation()
    softDeleteConversation(id)
    setShowMenu(false)
  }

  const handlePermanentDelete = (e: React.MouseEvent) => {
    e.stopPropagation()
    permanentlyDeleteConversation(id)
    setShowMenu(false)
  }

  const handleRestore = (e: React.MouseEvent) => {
    e.stopPropagation()
    restoreConversation(id)
    setShowMenu(false)
  }

  const handlePinToggle = (e: React.MouseEvent) => {
    e.stopPropagation()
    if (isPinned) {
      unpinConversation(id)
    } else {
      pinConversation(id)
    }
    setShowMenu(false)
  }

  const titleParts = highlightText(title, highlightQuery)
  const previewParts = preview ? highlightText(preview, highlightQuery) : []

  return (
    <div
      className={`
        group flex items-center gap-2 px-3 py-2.5 rounded-lg cursor-pointer
        transition-colors text-sm
        ${isActive
          ? 'bg-accent text-accent-foreground'
          : 'hover:bg-accent/50 text-muted-foreground hover:text-foreground'
        }
      `}
      onClick={onClick}
    >
      <MessageSquare size={16} className="shrink-0 mt-0.5" />

      {isEditing ? (
        <div className="flex-1 min-w-0">
          <input
            autoFocus
            className={`w-full bg-transparent border-b outline-none text-sm ${
              editError ? 'border-destructive' : 'border-primary'
            }`}
            value={editValue}
            onChange={(e) => {
              setEditValue(e.target.value)
              if (editError) setEditError(null)
            }}
            onBlur={handleRename}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleRename()
              if (e.key === 'Escape') {
                setEditValue(title)
                setEditError(null)
                setIsEditing(false)
              }
            }}
            onClick={(e) => e.stopPropagation()}
          />
          {editError && (
            <p className="text-[10px] text-destructive mt-0.5">{editError}</p>
          )}
        </div>
      ) : (
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2">
            <span className="truncate text-sm font-medium">
              {titleParts.map((part, i) =>
                part.highlight ? (
                  <mark key={i} className="bg-yellow-200 dark:bg-yellow-700 text-inherit rounded px-0.5">
                    {part.text}
                  </mark>
                ) : (
                  <span key={i}>{part.text}</span>
                )
              )}
            </span>
            {timestamp !== undefined && (
              <span className="text-[10px] text-muted-foreground shrink-0">
                {formatRelativeTime(timestamp)}
              </span>
            )}
          </div>
          {preview && previewParts.length > 0 && (
            <div className="flex items-center justify-between gap-2 mt-0.5">
              <span className="truncate text-xs text-muted-foreground">
                {previewParts.map((part, i) =>
                  part.highlight ? (
                    <mark key={i} className="bg-yellow-200 dark:bg-yellow-700 text-inherit rounded px-0.5">
                      {part.text}
                    </mark>
                  ) : (
                    <span key={i}>{part.text}</span>
                  )
                )}
              </span>
              {unread !== undefined && unread > 0 && (
                <span className="shrink-0 min-w-[18px] h-[18px] px-1.5 rounded-full bg-primary text-primary-foreground text-[10px] font-medium flex items-center justify-center">
                  {unread > 99 ? '99+' : unread}
                </span>
              )}
            </div>
          )}
        </div>
      )}

      {!isEditing && (
        <div className="relative">
          <button
            className={`
              p-1 rounded opacity-0 group-hover:opacity-100 transition-opacity
              ${isActive ? 'hover:bg-accent-foreground/10' : 'hover:bg-accent'}
            `}
            onClick={(e) => {
              e.stopPropagation()
              setShowMenu(!showMenu)
            }}
          >
            <MoreHorizontal size={14} />
          </button>

          {showMenu && (
            <div className="absolute right-0 top-7 z-20 w-32 bg-popover border border-border rounded-md shadow-lg py-1">
              {isTrashView ? (
                <>
                  <button
                    className="flex items-center gap-2 w-full px-3 py-1.5 text-xs hover:bg-accent transition-colors"
                    onClick={handleRestore}
                  >
                    <RotateCcw size={12} />
                    恢复
                  </button>
                  <button
                    className="flex items-center gap-2 w-full px-3 py-1.5 text-xs text-destructive hover:bg-destructive/10 transition-colors"
                    onClick={handlePermanentDelete}
                  >
                    <AlertTriangle size={12} />
                    永久删除
                  </button>
                </>
              ) : (
                <>
                  <button
                    className="flex items-center gap-2 w-full px-3 py-1.5 text-xs hover:bg-accent transition-colors"
                    onClick={(e) => {
                      e.stopPropagation()
                      handleEditStart()
                    }}
                  >
                    <Edit3 size={12} />
                    重命名
                  </button>
                  <button
                    className="flex items-center gap-2 w-full px-3 py-1.5 text-xs hover:bg-accent transition-colors"
                    onClick={handlePinToggle}
                  >
                    {isPinned ? <PinOff size={12} /> : <Pin size={12} />}
                    {isPinned ? '取消置顶' : '置顶'}
                  </button>
                  <button
                    className="flex items-center gap-2 w-full px-3 py-1.5 text-xs text-destructive hover:bg-destructive/10 transition-colors"
                    onClick={handleSoftDelete}
                  >
                    <Trash2 size={12} />
                    删除
                  </button>
                </>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
