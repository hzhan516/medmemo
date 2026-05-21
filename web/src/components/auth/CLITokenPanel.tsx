import { useState, useCallback } from 'react'
import { Terminal, CheckCircle2, AlertCircle, XCircle, Loader2, AlertTriangle, RefreshCw } from 'lucide-react'
import { BuildCLIProvider } from '@wails/go/main/WailsApp'
import type { AuthMethodDetectStatus, ProviderConfig } from '@/types/provider'

interface CLITokenPanelProps {
  status: AuthMethodDetectStatus | undefined
  onProviderCreated: (provider: ProviderConfig) => void
}

type CLIErrorType = 'network' | 'expired' | 'refresh' | 'unknown'

interface CLIError {
  message: string
  type: CLIErrorType
}

function classifyCLIError(message: string): CLIErrorType {
  if (message.includes('无法连接') || message.includes('超时')) return 'network'
  if (message.includes('已过期') || message.includes('无效')) return 'expired'
  if (message.includes('刷新')) return 'refresh'
  return 'unknown'
}

function getErrorTitle(type: CLIErrorType): string {
  switch (type) {
    case 'network':
      return '网络连接失败'
    case 'expired':
      return 'CLI Token 已过期'
    case 'refresh':
      return '凭证刷新失败'
    default:
      return '创建 Provider 失败'
  }
}

function getErrorActionHint(type: CLIErrorType): string | null {
  switch (type) {
    case 'network':
      return '如果您确认网络正常且 Token 有效，可以选择跳过验证直接保存。'
    case 'expired':
      return '请按上方提示重新登录 CLI 工具后再试。'
    case 'refresh':
      return '请按上方提示重新登录 CLI 工具后再试。'
    default:
      return null
  }
}

const cliProviderTemplates: Record<
  string,
  { name: string; apiHost: string; defaultCredentialPath: string }
> = {
  kimi: {
    name: 'Kimi (CLI)',
    apiHost: 'https://api.moonshot.cn',
    defaultCredentialPath: '~/.kimi/credentials/kimi-code.json',
  },
  gemini: {
    name: 'Gemini (CLI)',
    apiHost: 'https://generativelanguage.googleapis.com/v1beta/openai/',
    defaultCredentialPath: '~/.config/gcloud/application_default_credentials.json',
  },
}

/**
 * CLI Token 配置面板。
 * 展示 CLI 检测状态，支持一键构建 Provider 配置。
 */
export function CLITokenPanel({ status, onProviderCreated }: CLITokenPanelProps) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<CLIError | null>(null)

  const handleUseCLI = useCallback(async () => {
    if (!status?.provider_type) return
    setLoading(true)
    setError(null)
    try {
      const modelID = status.provider_type === 'kimi' ? 'moonshot-v1-8k' : 'gemini-pro'
      const provider = await BuildCLIProvider(status.provider_type, modelID)
      onProviderCreated(provider as unknown as Parameters<typeof onProviderCreated>[0])
    } catch (err) {
      const message = typeof err === 'string' ? err : (err instanceof Error ? err.message : '创建 Provider 失败')
      setError({ message, type: classifyCLIError(message) })
    } finally {
      setLoading(false)
    }
  }, [status, onProviderCreated])

  const handleSkipValidation = useCallback(() => {
    if (!status?.provider_type) return
    const tmpl = cliProviderTemplates[status.provider_type]
    if (!tmpl) return

    const modelID = status.provider_type === 'kimi' ? 'moonshot-v1-8k' : 'gemini-pro'
    const provider: ProviderConfig = {
      id: `cli-${status.provider_type}-${Date.now()}`,
      templateId: status.provider_type,
      name: tmpl.name,
      apiHost: tmpl.apiHost,
      apiKey: '',
      modelId: modelID,
      temperature: 0.7,
      timeoutMs: 30000,
      maxRetries: 3,
      group: 'CLI',
      enabled: true,
      sortOrder: 0,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      authMethod: 'cli_token',
      authParams: {
        cliCredentialPath: tmpl.defaultCredentialPath,
      },
      models: [{ id: modelID, name: modelID, enabled: true }],
    }
    onProviderCreated(provider)
    setError(null)
  }, [status, onProviderCreated])

  if (!status) {
    return (
      <div className="p-4 rounded-lg bg-muted/50 text-sm text-muted-foreground">
        尚未检测 CLI 状态。
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* 状态卡片 */}
      <div className="flex items-start gap-3 p-3 rounded-lg border">
        <Terminal className="w-5 h-5 text-muted-foreground shrink-0 mt-0.5" />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">
              {status.provider_type ? status.provider_type.toUpperCase() : 'CLI'} Token
            </span>
            {status.connected ? (
              <CheckCircle2 className="w-4 h-4 text-green-500" />
            ) : status.available ? (
              <AlertCircle className="w-4 h-4 text-amber-500" />
            ) : (
              <XCircle className="w-4 h-4 text-muted-foreground" />
            )}
          </div>
          <p className="text-xs text-muted-foreground mt-1">{status.detail}</p>
          {status.error && (
            <p className="text-xs text-destructive mt-1">{status.error}</p>
          )}
        </div>
      </div>

      {/* 操作按钮 */}
      {status.connected && (
        <button
          onClick={handleUseCLI}
          disabled={loading}
          className="w-full py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
        >
          {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Terminal className="w-4 h-4" />}
          {loading ? '正在创建…' : '一键使用 CLI Token'}
        </button>
      )}

      {status.available && !status.connected && (
        <div className="p-3 rounded-lg bg-amber-500/5 border border-amber-500/20 text-xs text-amber-700">
          {status.error || '请先完成 CLI 登录后再使用此方式'}
        </div>
      )}

      {!status.available && (
        <div className="p-3 rounded-lg bg-muted text-xs text-muted-foreground space-y-2">
          <p>未检测到支持的 CLI 工具。可安装以下工具：</p>
          <div className="space-y-1 font-mono text-[11px]">
            <p className="bg-background rounded px-2 py-1"># Kimi CLI</p>
            <p>npm install -g @kimi-ai/cli</p>
            <p>kimi login</p>
          </div>
          <div className="space-y-1 font-mono text-[11px]">
            <p className="bg-background rounded px-2 py-1"># Gemini CLI (gcloud)</p>
            <p>gcloud auth login</p>
          </div>
        </div>
      )}

      {/* 错误提示卡片 */}
      {error && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800">
          <AlertTriangle className="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
          <div className="flex-1 min-w-0 space-y-1.5">
            <p className="text-sm text-amber-800 dark:text-amber-200 font-medium">
              {getErrorTitle(error.type)}
            </p>
            <p className="text-xs text-amber-700 dark:text-amber-300">{error.message}</p>
            {getErrorActionHint(error.type) && (
              <p className="text-xs text-amber-600 dark:text-amber-400">
                {getErrorActionHint(error.type)}
              </p>
            )}
            <div className="flex items-center gap-2 pt-1">
              {error.type === 'network' && (
                <button
                  type="button"
                  onClick={handleSkipValidation}
                  className="text-xs text-amber-800 dark:text-amber-200 underline hover:no-underline"
                >
                  跳过验证，直接保存
                </button>
              )}
              <button
                type="button"
                onClick={handleUseCLI}
                disabled={loading}
                className="text-xs text-amber-800 dark:text-amber-200 underline hover:no-underline disabled:opacity-50 flex items-center gap-1"
              >
                <RefreshCw className="w-3 h-3" />
                重试
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
