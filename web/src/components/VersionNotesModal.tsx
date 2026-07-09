import { useState, useCallback } from 'react'
import { Sparkles, Wrench, ChevronDown, ChevronUp, BookOpen, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { VersionNote } from '@/hooks/useVersionNotes'

interface VersionNotesModalProps {
  notes: VersionNote[]
  currentVersion: string
  onDismiss: () => void
}

/**
 * 版本提示弹窗组件。
 * 展示当前版本及历史版本的功能与修复，倒序排列。
 * 最新版本默认展开，旧版本默认收起。
 */
export function VersionNotesModal({ notes, currentVersion, onDismiss }: VersionNotesModalProps) {
  const [expanded, setExpanded] = useState<Record<number, boolean>>(() => {
    // 默认展开第一个（最新版本）
    const init: Record<number, boolean> = {}
    if (notes.length > 0) {
      init[0] = true
    }
    return init
  })

  const toggleExpand = useCallback((index: number) => {
    setExpanded((prev) => ({ ...prev, [index]: !prev[index] }))
  }, [])

  if (notes.length === 0) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
      <div className="w-full max-w-2xl max-h-[85vh] flex flex-col rounded-2xl bg-background border border-border shadow-2xl overflow-hidden">
        {/* 头部 */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border bg-muted/40 shrink-0">
          <div className="flex items-center gap-3">
            <BookOpen className="w-5 h-5 text-primary shrink-0" />
            <div>
              <h2 className="text-lg font-semibold text-foreground">新功能与改进</h2>
              <p className="text-xs text-muted-foreground">以下是目前版本已包含的全部功能与修复</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-[11px] px-2 py-0.5 rounded-full bg-primary/10 text-primary border border-primary/20 font-medium">
              {currentVersion}
            </span>
            <button
              onClick={onDismiss}
              className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
            >
              <X size={18} />
            </button>
          </div>
        </div>

        {/* 内容区 */}
        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-3">
          {notes.map((note, index) => {
            const isExpanded = expanded[index] ?? false
            const isLatest = index === 0

            return (
              <div
                key={note.version}
                className={`rounded-xl border transition-colors ${
                  isLatest
                    ? 'border-primary/30 bg-primary/5'
                    : 'border-border bg-card'
                }`}
              >
                {/* 版本标题栏（可点击展开/收起） */}
                <button
                  onClick={() => toggleExpand(index)}
                  className="w-full flex items-center justify-between px-4 py-3 text-left hover:bg-accent/50 rounded-xl transition-colors"
                >
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-semibold text-foreground">{note.version}</span>
                    <span className="text-xs text-muted-foreground">{note.title}</span>
                    {isLatest && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-primary/10 text-primary font-medium">
                        当前版本
                      </span>
                    )}
                  </div>
                  {isExpanded ? (
                    <ChevronUp size={16} className="text-muted-foreground shrink-0" />
                  ) : (
                    <ChevronDown size={16} className="text-muted-foreground shrink-0" />
                  )}
                </button>

                {/* 展开内容 */}
                {isExpanded && (
                  <div className="px-4 pb-4 space-y-4">
                    {/* 新增功能 */}
                    {(note.features?.length ?? 0) > 0 && (
                      <div>
                        <div className="flex items-center gap-1.5 mb-2">
                          <Sparkles size={14} className="text-amber-500" />
                          <span className="text-xs font-medium text-foreground">新增功能</span>
                        </div>
                        <ul className="space-y-1.5">
                          {(note.features ?? []).map((feature: string, i: number) => (
                            <li
                              key={i}
                              className="flex items-start gap-2 text-sm text-foreground/80"
                            >
                              <span className="mt-1.5 w-1 h-1 rounded-full bg-amber-500 shrink-0" />
                              {feature}
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}

                    {/* 问题修复 */}
                    {(note.fixes?.length ?? 0) > 0 && (
                      <div>
                        <div className="flex items-center gap-1.5 mb-2">
                          <Wrench size={14} className="text-blue-500" />
                          <span className="text-xs font-medium text-foreground">问题修复</span>
                        </div>
                        <ul className="space-y-1.5">
                          {(note.fixes ?? []).map((fix: string, i: number) => (
                            <li
                              key={i}
                              className="flex items-start gap-2 text-sm text-foreground/80"
                            >
                              <span className="mt-1.5 w-1 h-1 rounded-full bg-blue-500 shrink-0" />
                              {fix}
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>

        {/* 底部 */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-border bg-muted/30 shrink-0">
          <Button variant="default" onClick={onDismiss}>
            知道了
          </Button>
        </div>
      </div>
    </div>
  )
}
