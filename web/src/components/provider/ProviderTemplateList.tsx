import { useState, useMemo, useCallback } from 'react'
import { Search } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { ProviderTemplateCard } from './ProviderTemplateCard'
import type { ProviderTemplate } from '@/types/provider'
import templatesData from '@/data/provider-templates.json'

interface ProviderTemplateListProps {
  onSelectTemplate: (template: ProviderTemplate) => void
  isAddedCheck?: (templateId: string) => boolean
}

/**
 * Provider 模板列表容器。
 * 负责加载模板 JSON、搜索过滤、响应式网格布局。
 */
export function ProviderTemplateList({
  onSelectTemplate,
  isAddedCheck,
}: ProviderTemplateListProps) {
  const templates = templatesData as ProviderTemplate[]
  const [searchQuery, setSearchQuery] = useState('')

  const filtered = useMemo(() => {
    const q = searchQuery.trim().toLowerCase()
    if (!q) return templates
    return templates.filter(
      (t) =>
        t.name.toLowerCase().includes(q) ||
        t.description.toLowerCase().includes(q) ||
        t.id.toLowerCase().includes(q)
    )
  }, [templates, searchQuery])

  const handleCardClick = useCallback(
    (template: ProviderTemplate) => {
      onSelectTemplate(template)
    },
    [onSelectTemplate]
  )

  return (
    <div className="space-y-4">
      {/* 搜索栏 */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <Input
          placeholder="搜索 Provider…"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-9"
          data-testid="provider-search-input"
        />
      </div>

      {/* 结果计数 */}
      {searchQuery.trim() && (
        <p className="text-xs text-muted-foreground">
          找到 {filtered.length} 个匹配结果
        </p>
      )}

      {/* 卡片网格 */}
      {filtered.length > 0 ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {filtered.map((template) => (
            <ProviderTemplateCard
              key={template.id}
              template={template}
              onClick={() => handleCardClick(template)}
              isAdded={isAddedCheck?.(template.id)}
            />
          ))}
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center py-12 gap-2">
          <p className="text-sm text-muted-foreground">未找到匹配的 Provider</p>
          <p className="text-xs text-muted-foreground">尝试更换关键词搜索</p>
        </div>
      )}
    </div>
  )
}
