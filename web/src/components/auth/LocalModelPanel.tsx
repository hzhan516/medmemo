import { useState, useCallback } from 'react'
import { Monitor, CheckCircle2, XCircle, Loader2, Download, Play } from 'lucide-react'
import {
  DetectOllama,
  StartOllamaServer,
  EnsureOllamaAndSmolLM2,
  CreateOllamaProvider,
} from '@wails/go/main/WailsApp'
import type { AuthMethodDetectStatus, ProviderConfig, OllamaDetectResult } from '@/types/provider'

interface LocalModelPanelProps {
  status: AuthMethodDetectStatus | undefined
  ollamaPulling: boolean
  ollamaPullProgress: string
  ollamaServerStarting: boolean
  onProviderCreated: (provider: ProviderConfig) => void
}

export function LocalModelPanel({
  status,
  ollamaPulling,
  ollamaPullProgress,
  ollamaServerStarting,
  onProviderCreated,
}: LocalModelPanelProps) {
  const [localStatus, setLocalStatus] = useState<OllamaDetectResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleDetect = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const result = await DetectOllama()
      setLocalStatus(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : '检测失败')
    } finally {
      setLoading(false)
    }
  }, [])

  const handleStartServer = useCallback(async () => {
    setError(null)
    try {
      await StartOllamaServer()
    } catch (err) {
      setError(err instanceof Error ? err.message : '启动失败')
    }
  }, [])

  const handleEnsureAll = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const result = await EnsureOllamaAndSmolLM2()
      setLocalStatus(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : '操作失败')
    } finally {
      setLoading(false)
    }
  }, [])

  const handleCreateProvider = useCallback(async () => {
    setCreating(true)
    setError(null)
    try {
      const provider = await CreateOllamaProvider()
      onProviderCreated(provider)
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建 Provider 失败')
    } finally {
      setCreating(false)
    }
  }, [onProviderCreated])

  const displayStatus = localStatus || (status ? ({
    installed: status.available,
    running: status.connected,
    has_smollm2: status.connected,
    install_guide: status.error || undefined,
  } as unknown as OllamaDetectResult) : null)

  if (!displayStatus) {
    return (
      <div className="space-y-4">
        <div className="p-4 rounded-lg bg-muted/50 text-sm text-muted-foreground">
          尚未检测 Ollama 状态。
        </div>
        <button
          onClick={handleDetect}
          disabled={loading}
          className="w-full py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
        >
          {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Monitor className="w-4 h-4" />}
          检测 Ollama 环境
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-3 p-3 rounded-lg border">
        <Monitor className="w-5 h-5 text-muted-foreground shrink-0 mt-0.5" />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">Ollama 本地模型</span>
            {displayStatus.installed && displayStatus.running ? (
              <CheckCircle2 className="w-4 h-4 text-green-500" />
            ) : displayStatus.installed ? (
              <XCircle className="w-4 h-4 text-amber-500" />
            ) : (
              <XCircle className="w-4 h-4 text-muted-foreground" />
            )}
          </div>
          <div className="mt-1 space-y-0.5">
            <div className="flex items-center gap-1.5 text-xs">
              <span className={displayStatus.installed ? 'text-green-600' : 'text-muted-foreground'}>
                {displayStatus.installed ? '已安装' : '未安装'}
              </span>
            </div>
            <div className="flex items-center gap-1.5 text-xs">
              <span className={displayStatus.running ? 'text-green-600' : 'text-muted-foreground'}>
                {displayStatus.running ? '运行中' : '未运行'}
              </span>
            </div>
            <div className="flex items-center gap-1.5 text-xs">
              <span className={displayStatus.has_smollm2 ? 'text-green-600' : 'text-muted-foreground'}>
                {displayStatus.has_smollm2 ? 'SmolLM2 已就绪' : 'SmolLM2 未下载'}
              </span>
            </div>
          </div>
        </div>
      </div>

      {(ollamaPulling || ollamaServerStarting) && (
        <div className="space-y-2">
          <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
            <div className="h-full bg-primary animate-pulse rounded-full" style={{ width: ollamaPulling ? '60%' : '40%' }} />
          </div>
          <p className="text-xs text-muted-foreground">
            {ollamaServerStarting ? '正在启动 Ollama 服务...' : ollamaPullProgress || '正在下载模型...'}
          </p>
        </div>
      )}

      {!displayStatus.installed && (
        <div className="p-3 rounded-lg bg-muted text-xs text-muted-foreground space-y-2">
          <p>Ollama 未安装，请执行以下命令安装：</p>
          <pre className="bg-background rounded px-2 py-1.5 font-mono text-[11px] overflow-x-auto">
curl -fsSL https://ollama.com/install.sh | sh
          </pre>
          <p className="text-[11px]">安装完成后点击重新检测。</p>
          <button
            onClick={handleDetect}
            disabled={loading}
            className="w-full mt-1 py-2 rounded-lg border text-sm hover:bg-accent transition-colors disabled:opacity-50"
          >
            {loading ? '检测中...' : '重新检测'}
          </button>
        </div>
      )}

      {displayStatus.installed && !displayStatus.running && !ollamaServerStarting && (
        <button
          onClick={handleStartServer}
          className="w-full py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors flex items-center justify-center gap-2"
        >
          <Play className="w-4 h-4" />
          启动 Ollama 服务
        </button>
      )}

      {displayStatus.installed && displayStatus.running && !displayStatus.has_smollm2 && !ollamaPulling && (
        <button
          onClick={handleEnsureAll}
          disabled={loading}
          className="w-full py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
        >
          {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Download className="w-4 h-4" />}
          {loading ? '处理中...' : '下载 SmolLM2 模型'}
        </button>
      )}

      {displayStatus.installed && displayStatus.running && displayStatus.has_smollm2 && (
        <button
          onClick={handleCreateProvider}
          disabled={creating}
          className="w-full py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
        >
          {creating ? <Loader2 className="w-4 h-4 animate-spin" /> : <Monitor className="w-4 h-4" />}
          {creating ? '创建中...' : '创建本地 Provider'}
        </button>
      )}

      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  )
}
