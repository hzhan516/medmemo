import { useState, useCallback, useEffect, useRef } from 'react'
import { Smartphone, CheckCircle2, AlertCircle, XCircle, Loader2, Copy, ExternalLink } from 'lucide-react'
import { StartOAuthDeviceFlow, CancelOAuthDeviceFlow, GetOAuthDeviceFlowStatus, GetOAuthDeviceFlowProviders } from '@wails/go/main/WailsApp'
import { BrowserOpenURL } from '@wails/runtime'
import type { AuthMethodDetectStatus, ProviderConfig, ProviderTemplate } from '@/types/provider'
import templatesData from '@/data/provider-templates.json'

interface OAuthDevicePanelProps {
  status: AuthMethodDetectStatus | undefined
  onProviderCreated: (provider: ProviderConfig) => void
}

// 从模板中提取支持 oauth_device 的厂商
const oauthProviders: ProviderTemplate[] = (templatesData as ProviderTemplate[]).filter((t) =>
  t.authMethods.includes('oauth_device')
)

/**
 * OAuth Device Flow 配置面板。
 * 选择厂商 → 选择模型 → 启动 Device Flow → 展示用户码 → 轮询状态 → 成功后回调。
 */
export function OAuthDevicePanel({ status, onProviderCreated }: OAuthDevicePanelProps) {
  const [providers, setProviders] = useState<Array<{ provider_type: string; name: string; available: boolean; configured: boolean; detail: string }>>([])
  const [selectedProvider, setSelectedProvider] = useState('')
  const [selectedModel, setSelectedModel] = useState('')
  const [loadingProviders, setLoadingProviders] = useState(true)

  const [deviceCode, setDeviceCode] = useState<string | null>(null)
  const [userCode, setUserCode] = useState('')
  const [verificationURI, setVerificationURI] = useState('')
  const [polling, setPolling] = useState(false)
  const [flowStatus, setFlowStatus] = useState<string>('')
  const [error, setError] = useState<string | null>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const selectedProviderInfo = providers.find((p) => p.provider_type === selectedProvider)
  const selectedTemplate = oauthProviders.find((p) => p.id === selectedProvider)

  // 加载支持的 OAuth Device Flow 厂商列表
  useEffect(() => {
    let cancelled = false
    setLoadingProviders(true)
    GetOAuthDeviceFlowProviders()
      .then((list) => {
        if (cancelled) return
        setProviders(list)
        // 默认选中第一个可用的厂商
        const firstAvailable = list.find((p) => p.available)
        if (firstAvailable) {
          setSelectedProvider(firstAvailable.provider_type)
          const tmpl = oauthProviders.find((t) => t.id === firstAvailable.provider_type)
          if (tmpl) setSelectedModel(tmpl.defaultModel)
        } else if (list.length > 0) {
          setSelectedProvider(list[0].provider_type)
          const tmpl = oauthProviders.find((t) => t.id === list[0].provider_type)
          if (tmpl) setSelectedModel(tmpl.defaultModel)
        }
      })
      .catch(() => {
        if (cancelled) return
        setError('加载 OAuth Device Flow 厂商列表失败')
      })
      .finally(() => {
        if (!cancelled) setLoadingProviders(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  // 厂商切换时同步更新默认模型
  useEffect(() => {
    const tmpl = oauthProviders.find((t) => t.id === selectedProvider)
    if (tmpl) {
      setSelectedModel(tmpl.defaultModel)
    }
  }, [selectedProvider])

  const clearPolling = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current)
      intervalRef.current = null
    }
  }, [])

  useEffect(() => {
    return () => clearPolling()
  }, [clearPolling])

  const handleStart = useCallback(async () => {
    if (!selectedProvider) {
      setError('请先选择一个厂商')
      return
    }
    if (selectedProviderInfo && !selectedProviderInfo.available) {
      setError(`${selectedProviderInfo.name} 的 OAuth 配置不可用：${selectedProviderInfo.detail}`)
      return
    }

    setError(null)
    setFlowStatus('')
    clearPolling()
    try {
      const result = await StartOAuthDeviceFlow(selectedProvider)
      setDeviceCode(result.device_code)
      setUserCode(result.user_code)
      setVerificationURI(result.verification_uri)
      setPolling(true)

      // 开始轮询
      const intervalMs = (result.interval || 5) * 1000
      intervalRef.current = setInterval(async () => {
        try {
          const s = await GetOAuthDeviceFlowStatus(result.device_code)
          if (!s) {
            setFlowStatus('error')
            setPolling(false)
            clearPolling()
            return
          }
          setFlowStatus(s.status)
          if (s.status === 'success') {
            setPolling(false)
            clearPolling()
            // 构造 ProviderConfig 通知上层
            if (s.provider_id) {
              const providerConfig = buildProviderConfig(s.provider_type, s.provider_id, s.provider_name || '', selectedModel)
              onProviderCreated(providerConfig)
            }
          } else if (s.status === 'error' || s.status === 'cancelled') {
            setPolling(false)
            clearPolling()
            if (s.error) setError(s.error)
          }
        } catch {
          // 轮询异常继续
        }
      }, intervalMs)
    } catch (err) {
      setError(typeof err === 'string' ? err : (err instanceof Error ? err.message : '启动 Device Flow 失败'))
    }
  }, [selectedProvider, selectedProviderInfo, selectedModel, clearPolling, onProviderCreated])

  const handleCancel = useCallback(async () => {
    if (deviceCode) {
      await CancelOAuthDeviceFlow(deviceCode)
    }
    clearPolling()
    setDeviceCode(null)
    setUserCode('')
    setVerificationURI('')
    setPolling(false)
    setFlowStatus('')
  }, [deviceCode, clearPolling])

  const handleOpenURL = useCallback(() => {
    if (verificationURI) {
      BrowserOpenURL(verificationURI)
    }
  }, [verificationURI])

  const handleCopyCode = useCallback(async () => {
    if (userCode) {
      await navigator.clipboard.writeText(userCode)
    }
  }, [userCode])

  if (!status) {
    return (
      <div className="p-4 rounded-lg bg-muted/50 text-sm text-muted-foreground">
        尚未检测 OAuth 状态。
      </div>
    )
  }

  // Device Flow 进行中
  if (deviceCode) {
    return (
      <div className="space-y-4">
        <div className="p-4 rounded-lg border space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">设备授权码</span>
            <button
              onClick={handleCopyCode}
              className="flex items-center gap-1 text-xs text-primary hover:underline"
            >
              <Copy className="w-3 h-3" />
              复制
            </button>
          </div>
          <div className="text-center py-3 bg-muted rounded-lg">
            <code className="text-2xl font-bold tracking-widest">{userCode}</code>
          </div>
          <p className="text-xs text-muted-foreground">
            请在浏览器中访问下方链接并输入上方授权码：
          </p>
          <button
            onClick={handleOpenURL}
            className="w-full flex items-center justify-center gap-2 py-2 rounded-lg border text-sm hover:bg-accent transition-colors"
          >
            <ExternalLink className="w-4 h-4" />
            {verificationURI}
          </button>
        </div>

        <div className="flex items-center gap-2 text-xs">
          {polling ? (
            <>
              <Loader2 className="w-3 h-3 animate-spin text-primary" />
              <span className="text-muted-foreground">等待授权…</span>
            </>
          ) : flowStatus === 'success' ? (
            <>
              <CheckCircle2 className="w-3 h-3 text-green-500" />
              <span className="text-green-600">授权成功</span>
            </>
          ) : (
            <>
              <XCircle className="w-3 h-3 text-destructive" />
              <span className="text-destructive">{error || '授权失败'}</span>
            </>
          )}
        </div>

        <button
          onClick={handleCancel}
          className="w-full py-2 rounded-lg border text-sm hover:bg-accent transition-colors"
        >
          取消授权
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-3 p-3 rounded-lg border">
        <Smartphone className="w-5 h-5 text-muted-foreground shrink-0 mt-0.5" />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">OAuth Device Flow</span>
            {status.connected ? (
              <CheckCircle2 className="w-4 h-4 text-green-500" />
            ) : status.available ? (
              <AlertCircle className="w-4 h-4 text-amber-500" />
            ) : (
              <XCircle className="w-4 h-4 text-muted-foreground" />
            )}
          </div>
          <p className="text-xs text-muted-foreground mt-1">{status.detail}</p>
        </div>
      </div>

      {/* 厂商选择 */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium">厂商</label>
        {loadingProviders ? (
          <div className="flex items-center gap-2 text-xs text-muted-foreground py-2">
            <Loader2 className="w-3 h-3 animate-spin" />
            加载厂商列表…
          </div>
        ) : (
          <>
            <select
              value={selectedProvider}
              onChange={(e) => {
                setSelectedProvider(e.target.value)
                setError(null)
              }}
              className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/30"
            >
              {providers.map((p) => (
                <option key={p.provider_type} value={p.provider_type}>
                  {p.name}
                </option>
              ))}
            </select>
            {selectedProviderInfo && !selectedProviderInfo.available && (
              <p className="text-xs text-amber-600">
                {selectedProviderInfo.detail}
              </p>
            )}
          </>
        )}
      </div>

      {/* 模型选择 */}
      {selectedTemplate && selectedTemplate.models.length > 0 && (
        <div className="space-y-1.5">
          <label className="text-sm font-medium">模型</label>
          <select
            value={selectedModel}
            onChange={(e) => setSelectedModel(e.target.value)}
            className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/30"
          >
            {selectedTemplate.models.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
        </div>
      )}

      {!status.connected && (
        <button
          onClick={handleStart}
          disabled={loadingProviders || !selectedProvider || (selectedProviderInfo?.available === false)}
          className="w-full py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
        >
          <Smartphone className="w-4 h-4" />
          启动 Device Flow 授权
        </button>
      )}

      {status.connected && (
        <div className="p-3 rounded-lg bg-green-500/5 border border-green-500/20 text-xs text-green-700">
          已通过 OAuth Device Flow 授权，无需重复操作。
        </div>
      )}

      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  )
}

/**
 * 根据 OAuth Device Flow 成功结果构造 ProviderConfig。
 * 与后端 inferProviderInfo 保持一致。
 */
function buildProviderConfig(providerType: string, providerID: string, providerName: string, modelId: string): ProviderConfig {
  const now = Date.now()
  const configs: Record<string, { apiHost: string; modelId: string; name: string }> = {
    kimi: {
      apiHost: 'https://api.moonshot.cn',
      modelId: modelId || 'moonshot-v1-8k',
      name: 'Kimi (OAuth)',
    },
    gemini: {
      apiHost: 'https://generativelanguage.googleapis.com/v1beta/openai',
      modelId: modelId || 'gemini-1.5-flash',
      name: 'Gemini (OAuth)',
    },
    microsoft: {
      apiHost: 'https://models.inference.ai.azure.com',
      modelId: modelId || 'gpt-4o',
      name: 'Microsoft (OAuth)',
    },
    github: {
      apiHost: 'https://models.inference.ai.azure.com',
      modelId: modelId || 'gpt-4o',
      name: 'GitHub (OAuth)',
    },
  }

  const cfg = configs[providerType] || { apiHost: '', modelId: modelId || '', name: providerName || `OAuth ${providerType}` }

  return {
    id: providerID,
    templateId: providerType,
    name: providerName || cfg.name,
    apiHost: cfg.apiHost,
    apiKey: '',
    modelId: cfg.modelId,
    temperature: 0.7,
    timeoutMs: 30000,
    maxRetries: 3,
    group: 'OAuth',
    enabled: true,
    sortOrder: 0,
    createdAt: now,
    updatedAt: now,
    authMethod: 'oauth_device',
    authParams: {},
    models: [{ id: cfg.modelId, name: cfg.modelId, enabled: true }],
  }
}
