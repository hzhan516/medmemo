import { Cloud, Server, Cpu, ExternalLink } from 'lucide-react'
import type { ProviderTemplate } from '@/types/provider'

interface ProviderTemplateCardProps {
  template: ProviderTemplate
  onClick: () => void
  isAdded?: boolean
}

/**
 * 选择图标组件，根据 Provider 类型映射。
 */
function ProviderIcon({ type }: { type: ProviderTemplate['type'] }) {
  const iconClass = 'w-5 h-5'
  if (type === 'cloud') {
    return <Cloud className={iconClass} />
  }
  if (type === 'local') {
    return <Server className={iconClass} />
  }
  return <Cpu className={iconClass} />
}

/**
 * 单个 Provider 模板卡片。
 * 展示图标、名称、描述、类型标签和文档链接。
 */
export function ProviderTemplateCard({ template, onClick, isAdded }: ProviderTemplateCardProps) {
  return (
    <div
      onClick={onClick}
      className="group relative flex flex-col rounded-xl border border-border bg-card p-4 shadow-sm cursor-pointer
                 hover:shadow-md hover:-translate-y-0.5 transition-all duration-200"
      data-testid={`provider-card-${template.id}`}
    >
      {/* 顶部：图标 + 名称 + 标签 */}
      <div className="flex items-start justify-between gap-2 mb-2">
        <div className="flex items-center gap-2.5 flex-1 min-w-0">
          <div
            className={`shrink-0 w-9 h-9 rounded-lg flex items-center justify-center ${
              template.type === 'local'
                ? 'bg-amber-500/10 text-amber-600'
                : 'bg-blue-500/10 text-blue-600'
            }`}
          >
            <ProviderIcon type={template.type} />
          </div>
          <div className="min-w-0 flex-1">
            <h3 className="text-sm font-semibold text-foreground truncate" title={template.name}>{template.name}</h3>
          </div>
        </div>
        <span
          className={`shrink-0 text-[10px] font-medium px-1.5 py-0.5 rounded-full ${
            template.type === 'local'
              ? 'bg-amber-500/10 text-amber-600'
              : 'bg-blue-500/10 text-blue-600'
          }`}
        >
          {template.type === 'local' ? '本地' : '云端'}
        </span>
      </div>

      {/* 描述 */}
      <p className="text-xs text-muted-foreground line-clamp-2 mb-3 flex-1">{template.description}</p>

      {/* 底部：文档链接 + 已添加标识 */}
      <div className="flex items-center justify-between mt-auto">
        <a
          href={template.docsUrl}
          target="_blank"
          rel="noopener noreferrer"
          onClick={(e) => e.stopPropagation()}
          className="inline-flex items-center gap-1 text-[11px] text-muted-foreground hover:text-primary transition-colors"
          title="查看官方文档"
        >
          <ExternalLink className="w-3 h-3" />
          文档
        </a>
        {isAdded && (
          <span className="text-[11px] text-green-600 font-medium">已添加</span>
        )}
      </div>
    </div>
  )
}
