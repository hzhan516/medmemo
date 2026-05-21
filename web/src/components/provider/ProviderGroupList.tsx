import { useState, useMemo } from 'react'
import { ChevronDown, ChevronRight, Pencil, Trash2, Cloud, Server } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import type { ProviderConfig } from '@/types/provider'

interface ProviderGroupListProps {
  providers: ProviderConfig[]
  activeProviderId: string | null
  onEdit: (provider: ProviderConfig) => void
  onDelete: (provider: ProviderConfig) => void
}

/**
 * 按分组折叠展示已添加的 Provider 列表。
 * 每个分组可展开/收起，Provider 按 group 字段聚合。
 */
export function ProviderGroupList({
  providers,
  activeProviderId,
  onEdit,
  onDelete,
}: ProviderGroupListProps) {
  // 按分组聚合
  const groups = useMemo(() => {
    const map = new Map<string, ProviderConfig[]>()
    for (const p of providers) {
      const list = map.get(p.group) || []
      list.push(p)
      map.set(p.group, list)
    }
    // 按分组名排序
    return Array.from(map.entries()).sort(([a], [b]) => a.localeCompare(b))
  }, [providers])

  // 默认展开所有分组
  const [expanded, setExpanded] = useState<Set<string>>(() => {
    return new Set(groups.map(([name]) => name))
  })

  const toggleGroup = (name: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(name)) {
        next.delete(name)
      } else {
        next.add(name)
      }
      return next
    })
  }

  if (providers.length === 0) {
    return (
      <div className="text-center py-6 text-sm text-muted-foreground border border-dashed border-border rounded-lg">
        暂无已添加的 Provider
      </div>
    )
  }

  return (
    <div className="space-y-2" data-testid="provider-group-list">
      {groups.map(([groupName, items]) => {
        const isExpanded = expanded.has(groupName)
        return (
          <Card key={groupName} className="border-border overflow-hidden">
            {/* 分组头 */}
            <button
              onClick={() => toggleGroup(groupName)}
              className="w-full flex items-center justify-between px-3 py-2.5 hover:bg-accent/50 transition-colors"
              data-testid={`group-header-${groupName}`}
            >
              <div className="flex items-center gap-2">
                {isExpanded ? (
                  <ChevronDown className="w-4 h-4 text-muted-foreground" />
                ) : (
                  <ChevronRight className="w-4 h-4 text-muted-foreground" />
                )}
                <span className="text-sm font-medium">{groupName}</span>
                <span className="text-[11px] text-muted-foreground bg-muted px-1.5 py-0.5 rounded-full">
                  {items.length}
                </span>
              </div>
            </button>

            {/* 分组内容 */}
            {isExpanded && (
              <CardContent className="p-0 px-3 pb-2 space-y-1.5">
                {items.map((p) => {
                  const isActive = activeProviderId === p.id
                  const isLocal = p.group === '本地' || p.apiHost.includes('localhost') || p.apiHost.includes('127.0.0.1')
                  return (
                    <div
                      key={p.id}
                      className={`flex items-center justify-between px-3 py-2 rounded-lg border transition-colors ${
                        isActive
                          ? 'border-primary/40 bg-primary/5'
                          : 'border-border/60 hover:border-border hover:bg-accent/30'
                      }`}
                      data-testid={`provider-item-${p.id}`}
                    >
                      <div className="flex items-center gap-2.5 min-w-0">
                        <div
                          className={`shrink-0 w-7 h-7 rounded-md flex items-center justify-center ${
                            isLocal
                              ? 'bg-amber-500/10 text-amber-600'
                              : 'bg-blue-500/10 text-blue-600'
                          }`}
                        >
                          {isLocal ? (
                            <Server className="w-3.5 h-3.5" />
                          ) : (
                            <Cloud className="w-3.5 h-3.5" />
                          )}
                        </div>
                        <div className="min-w-0">
                          <div className="flex items-center gap-1.5">
                            <span className="text-sm font-medium text-foreground truncate">
                              {p.name}
                            </span>
                            {isActive && (
                              <span className="text-[10px] font-medium px-1 py-0.5 rounded-full bg-green-500/10 text-green-600 shrink-0">
                                活跃
                              </span>
                            )}
                            {!p.enabled && (
                              <span className="text-[10px] font-medium px-1 py-0.5 rounded-full bg-gray-500/10 text-gray-500 shrink-0">
                                禁用
                              </span>
                            )}
                          </div>
                          <div className="text-[11px] text-muted-foreground truncate">
                            {p.models && p.models.length > 0
                              ? `${p.models.filter((m) => m.enabled).length}/${p.models.length} 个模型 · ${p.apiHost}`
                              : `${p.modelId} · ${p.apiHost}`}
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-0.5 shrink-0">
                        <button
                          onClick={() => onEdit(p)}
                          className="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
                          title="编辑"
                          aria-label={`编辑 ${p.name}`}
                          data-testid={`provider-edit-btn-${p.id}`}
                        >
                          <Pencil className="w-3.5 h-3.5" />
                        </button>
                        <button
                          onClick={() => onDelete(p)}
                          className="p-1.5 rounded-md text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
                          title="删除"
                          aria-label={`删除 ${p.name}`}
                          data-testid={`provider-delete-btn-${p.id}`}
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>
                  )
                })}
              </CardContent>
            )}
          </Card>
        )
      })}
    </div>
  )
}
