import { useCallback } from 'react'
import { Terminal, Smartphone, KeyRound, Monitor, Cloud, ChevronDown, ChevronUp, Loader2, Sparkles, XCircle, CheckCircle2, AlertCircle } from 'lucide-react'
import type { AuthDetectResult, AuthMethodDetectStatus, ProviderConfig } from '@/types/provider'
import type { AuthPanel } from '@/hooks/useAuth'
import { CLITokenPanel } from './CLITokenPanel'
import { OAuthDevicePanel } from './OAuthDevicePanel'
import { APIKeyPanel } from './APIKeyPanel'
import { LocalModelPanel } from './LocalModelPanel'
import { ServiceAccountPanel } from './ServiceAccountPanel'

interface AuthSelectorProps {
  result: AuthDetectResult | null
  detecting: boolean
  error: string | null
  expandedPanel: AuthPanel | null
  ollamaPulling: boolean
  ollamaPullProgress: string
  ollamaServerStarting: boolean
  onDetect: () => void
  onSelectMethod: (method: AuthPanel) => void
  onProviderCreated: (provider: ProviderConfig) => void
  onSkip?: () => void
}

const methodMeta: Record<AuthPanel, { label: string; icon: typeof Terminal; tier: number }> = {
  cli_token: { label: 'CLI Token', icon: Terminal, tier: 1 },
  oauth_device: { label: 'OAuth Device Flow', icon: Smartphone, tier: 2 },
  api_key: { label: 'API Key', icon: KeyRound, tier: 3 },
  service_account: { label: 'Vertex AI', icon: Cloud, tier: 3 },
  local: { label: '本地模型 (Ollama)', icon: Monitor, tier: 4 },
}

function getStatusIcon(status: AuthMethodDetectStatus | undefined) {
  if (!status) return <XCircle className="w-4 h-4 text-muted-foreground" />
  if (status.connected) return <CheckCircle2 className="w-4 h-4 text-green-500" />
  if (status.available) return <AlertCircle className="w-4 h-4 text-amber-500" />
  return <XCircle className="w-4 h-4 text-muted-foreground" />
}

function getStatusText(status: AuthMethodDetectStatus | undefined) {
  if (!status) return '未检测'
  if (status.connected) return '已连接'
  if (status.available) return '可用'
  return '不可用'
}

/**
 * 认证方式智能选择主组件。
 * 展示四种认证方式的状态卡片，支持展开配置面板，智能推荐最佳方式。
 */
export function AuthSelector({
  result,
  detecting,
  error,
  expandedPanel,
  ollamaPulling,
  ollamaPullProgress,
  ollamaServerStarting,
  onDetect,
  onSelectMethod,
  onProviderCreated,
  onSkip,
}: AuthSelectorProps) {
  const getStatus = useCallback(
    (method: AuthPanel): AuthMethodDetectStatus | undefined => {
      return result?.results.find((r) => r.method === method)
    },
    [result]
  )

  const handleCardClick = useCallback(
    (method: AuthPanel) => {
      onSelectMethod(method)
    },
    [onSelectMethod]
  )

  // 检测中
  if (detecting) {
    return (
      <div className="flex flex-col items-center gap-3 py-8">
        <Loader2 className="w-8 h-8 text-primary animate-spin" />
        <p className="text-sm text-muted-foreground">正在检测认证环境...</p>
      </div>
    )
  }

  // 检测错误
  if (error) {
    return (
      <div className="space-y-4">
        <div className="p-4 rounded-lg bg-destructive/5 border border-destructive/20 text-sm text-destructive">
          {error}
        </div>
        <button
          onClick={onDetect}
          className="w-full py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
        >
          重新检测
        </button>
      </div>
    )
  }

  // 初始未检测状态
  if (!result) {
    return (
      <div className="space-y-4">
        <div className="text-center space-y-2">
          <h3 className="text-lg font-semibold">选择认证方式</h3>
          <p className="text-sm text-muted-foreground">MedMemo 支持多种认证方式，让我们先检测您的环境</p>
        </div>
        <button
          onClick={onDetect}
          className="w-full py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors flex items-center justify-center gap-2"
        >
          <Loader2 className="w-4 h-4" />
          开始检测
        </button>
        {onSkip && (
          <button
            onClick={onSkip}
            className="w-full py-2 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            跳过，稍后配置
          </button>
        )}
      </div>
    )
  }

  if (!result) return null

  const methods: AuthPanel[] = ['cli_token', 'oauth_device', 'api_key', 'service_account', 'local']
  const recommended = result.recommended as AuthPanel

  return (
    <div className="space-y-4">
      {/* 智能推荐横幅 */}
      {recommended && !result.all_unavailable && (
        <div className="flex items-center gap-2 p-3 rounded-lg bg-primary/5 border border-primary/20">
          <Sparkles className="w-4 h-4 text-primary shrink-0" />
          <p className="text-sm">
            <span className="font-medium">为您推荐：</span>
            <span className="text-primary">{methodMeta[recommended]?.label || recommended}</span>
          </p>
        </div>
      )}

      {result.all_unavailable && (
        <div className="flex items-center gap-2 p-3 rounded-lg bg-amber-500/5 border border-amber-500/20">
          <AlertCircle className="w-4 h-4 text-amber-500 shrink-0" />
          <p className="text-sm text-amber-700">
            未检测到可用的认证方式，建议配置 API Key 或安装 Ollama 本地模型。
          </p>
        </div>
      )}

      {/* 四种认证方式卡片 */}
      <div className="space-y-2">
        {methods.map((method) => {
          const meta = methodMeta[method]
          const status = getStatus(method)
          const isExpanded = expandedPanel === method
          const isRecommended = recommended === method

          return (
            <div key={method} className="rounded-lg border overflow-hidden">
              {/* 卡片头部 */}
              <button
                data-testid={`auth-card-${method}`}
                onClick={() => handleCardClick(method)}
                className="w-full flex items-center gap-3 p-3 text-left hover:bg-accent/50 transition-colors"
              >
                <meta.icon className="w-5 h-5 text-muted-foreground shrink-0" />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">{meta.label}</span>
                    {isRecommended && (
                      <span className="text-[10px] font-medium px-1.5 py-0.5 rounded-full bg-primary/10 text-primary">
                        推荐
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground">{status?.detail || '未检测'}</p>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <span className="text-xs text-muted-foreground">{getStatusText(status)}</span>
                  {getStatusIcon(status)}
                  {isExpanded ? (
                    <ChevronUp className="w-4 h-4 text-muted-foreground" />
                  ) : (
                    <ChevronDown className="w-4 h-4 text-muted-foreground" />
                  )}
                </div>
              </button>

              {/* 展开的配置面板 */}
              {isExpanded && (
                <div className="px-3 pb-3 border-t bg-muted/20">
                  <div className="pt-3">
                    {method === 'cli_token' && (
                      <CLITokenPanel status={status} onProviderCreated={onProviderCreated} />
                    )}
                    {method === 'oauth_device' && (
                      <OAuthDevicePanel status={status} onProviderCreated={onProviderCreated} />
                    )}
                    {method === 'api_key' && (
                      <APIKeyPanel status={status} onProviderCreated={onProviderCreated} />
                    )}
                    {method === 'service_account' && (
                      <ServiceAccountPanel status={status} onProviderCreated={onProviderCreated} />
                    )}
                    {method === 'local' && (
                      <LocalModelPanel
                        status={status}
                        ollamaPulling={ollamaPulling}
                        ollamaPullProgress={ollamaPullProgress}
                        ollamaServerStarting={ollamaServerStarting}
                        onProviderCreated={onProviderCreated}
                      />
                    )}
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>

      {/* 底部操作 */}
      <div className="flex items-center justify-between pt-2">
        <button
          onClick={onDetect}
          className="text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          重新检测
        </button>
        {onSkip && (
          <button
            onClick={onSkip}
            className="text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            跳过，稍后配置
          </button>
        )}
      </div>
    </div>
  )
}
