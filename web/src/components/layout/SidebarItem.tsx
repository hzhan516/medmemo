import { MessageSquare, MoreHorizontal, Trash2, Edit3 } from 'lucide-react'
import { useState } from 'react'

interface SidebarItemProps {
  id: string
  title: string
  isActive?: boolean
  onClick?: () => void
  onRename?: (id: string, newTitle: string) => void
  onDelete?: (id: string) => void
}

/**
 * 单条会话列表项。
 */
export function SidebarItem({
  id,
  title,
  isActive = false,
  onClick,
  onRename,
  onDelete,
}: SidebarItemProps) {
  const [isEditing, setIsEditing] = useState(false)
  const [editValue, setEditValue] = useState(title)
  const [showMenu, setShowMenu] = useState(false)

  const handleRename = () => {
    if (editValue.trim() && onRename) {
      onRename(id, editValue.trim())
    }
    setIsEditing(false)
    setShowMenu(false)
  }

  const handleDelete = () => {
    onDelete?.(id)
    setShowMenu(false)
  }

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
      <MessageSquare size={16} className="shrink-0" />

      {isEditing ? (
        <input
          autoFocus
          className="flex-1 bg-transparent border-b border-primary outline-none text-sm"
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          onBlur={handleRename}
          onKeyDown={(e) => {
            if (e.key === 'Enter') handleRename()
            if (e.key === 'Escape') {
              setEditValue(title)
              setIsEditing(false)
            }
          }}
          onClick={(e) => e.stopPropagation()}
        />
      ) : (
        <span className="flex-1 truncate">{title || '新对话'}</span>
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
            <div className="absolute right-0 top-7 z-20 w-28 bg-popover border border-border rounded-md shadow-lg py-1">
              <button
                className="flex items-center gap-2 w-full px-3 py-1.5 text-xs hover:bg-accent transition-colors"
                onClick={(e) => {
                  e.stopPropagation()
                  setIsEditing(true)
                }}
              >
                <Edit3 size={12} />
                重命名
              </button>
              <button
                className="flex items-center gap-2 w-full px-3 py-1.5 text-xs text-destructive hover:bg-destructive/10 transition-colors"
                onClick={(e) => {
                  e.stopPropagation()
                  handleDelete()
                }}
              >
                <Trash2 size={12} />
                删除
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
