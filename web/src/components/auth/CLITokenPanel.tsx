import { useState, useCallback } from 'react'
import { Terminal, CheckCircle2, AlertCircle, XCircle, Loader2 } from 'lucide-react'
import { BuildCLIProvider } from '@wails/go/main/WailsApp'
import type { AuthMethodDetectStatus, ProviderConfig } from '@/types/provider'

interface CLITokenPanelProps {
  status: AuthMethodDetectStatus | undefined
  onProviderCreated: (provider: ProviderConfig) => void
}

/**
 * CLI Token 配置面板。
 * 展示 CLI 检测状态，支持一键构建 Provider 配置。
 */
export function CLITokenPanel({ status, onProviderCreated }: CLITokenPanelProps) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleUseCLI = useCallback(async () => {
    if (!status?.provider_type) return
    setLoading(true)
    setError(null)
    try {
      const modelID = status.provider_type === 'kimi' ? 'moonshot-v1-8k' : 'gemini-pro'
      const provider = await BuildCLIProvider(status.provider_type, modelID)
      onProviderCreated(provider)
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建 Provider 失败')
    } finally {
      setLoading(false)
    }
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
            <span className="text-sm font-medium">{status.provider_type ? status.provider_type.toUpperCase() : 'CLI'} Token</span>
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

      {error && (
        <p className="text-xs text-destructive">{error}</p>
      )}
    </div>
  )
}
