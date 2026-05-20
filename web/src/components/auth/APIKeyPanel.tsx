import { useState, useCallback } from 'react'
import { KeyRound, Eye, EyeOff, CheckCircle2, XCircle, Loader2 } from 'lucide-react'
import { SaveAPIKey } from '@wails/go/main/WailsApp'
import type { AuthMethodDetectStatus, ProviderConfig } from '@/types/provider'

interface APIKeyPanelProps {
  status: AuthMethodDetectStatus | undefined
  onProviderCreated: (provider: ProviderConfig) => void
}

const providers = [
  { id: 'openai', name: 'OpenAI', host: 'https://api.openai.com', model: 'gpt-4o' },
  { id: 'kimi', name: 'Kimi (Moonshot)', host: 'https://api.moonshot.cn', model: 'moonshot-v1-8k' },
  { id: 'deepseek', name: 'DeepSeek', host: 'https://api.deepseek.com', model: 'deepseek-chat' },
  { id: 'claude', name: 'Claude', host: 'https://api.anthropic.com', model: 'claude-3-5-sonnet' },
  { id: 'qwen', name: '通义千问', host: 'https://dashscope.aliyuncs.com', model: 'qwen-turbo' },
]

const keyPrefixPatterns: Record<string, RegExp> = {
  openai: /^sk-/,
  kimi: /^sk-/,
  deepseek: /^sk-/,
  claude: /^sk-ant-/,
  qwen: /^sk-/,
}

/**
 * API Key 配置面板。
 * 输入 API Key、厂商选择、格式校验、保存到密钥环保管。
 */
export function APIKeyPanel({ status, onProviderCreated }: APIKeyPanelProps) {
  const [selectedProvider, setSelectedProvider] = useState('kimi')
  const [apiKey, setApiKey] = useState('')
  const [showKey, setShowKey] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const provider = providers.find((p) => p.id === selectedProvider)!
  const prefixValid = apiKey ? (keyPrefixPatterns[selectedProvider]?.test(apiKey) ?? true) : true
  const lengthValid = apiKey.length >= 8

  const handleSave = useCallback(async () => {
    if (!apiKey.trim()) {
      setError('请输入 API Key')
      return
    }
    if (!prefixValid) {
      setError('API Key 前缀格式不正确')
      return
    }

    setLoading(true)
    setError(null)
    try {
      await SaveAPIKey(selectedProvider, apiKey.trim())

      const newProvider: ProviderConfig = {
        id: `${selectedProvider}_apikey_${Date.now()}`,
        templateId: selectedProvider,
        name: provider.name,
        apiHost: provider.host,
        apiKey: apiKey.trim(),
        modelId: provider.model,
        temperature: 0.7,
        timeoutMs: 30000,
        maxRetries: 3,
        group: '云端',
        enabled: true,
        sortOrder: 0,
        createdAt: Date.now(),
        updatedAt: Date.now(),
        authMethod: 'api_key',
        authParams: {},
      }
      onProviderCreated(newProvider)
      setApiKey('')
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setLoading(false)
    }
  }, [apiKey, selectedProvider, prefixValid, provider, onProviderCreated])

  if (!status) {
    return (
      <div className="p-4 rounded-lg bg-muted/50 text-sm text-muted-foreground">
        尚未检测 API Key 状态。
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* 状态卡片 */}
      <div className="flex items-start gap-3 p-3 rounded-lg border">
        <KeyRound className="w-5 h-5 text-muted-foreground shrink-0 mt-0.5" />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">API Key</span>
            {status.connected ? (
              <CheckCircle2 className="w-4 h-4 text-green-500" />
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
        <select
          value={selectedProvider}
          onChange={(e) => {
            setSelectedProvider(e.target.value)
            setError(null)
          }}
          className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/30"
        >
          {providers.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
      </div>

      {/* API Key 输入 */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium">API Key</label>
        <div className="relative">
          <input
            type={showKey ? 'text' : 'password'}
            value={apiKey}
            onChange={(e) => {
              setApiKey(e.target.value)
              setError(null)
            }}
            placeholder={`${provider.name} 的 API Key`}
            className="w-full px-3 py-2 pr-10 rounded-lg border border-border bg-background text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/30"
          />
          <button
            type="button"
            onClick={() => setShowKey(!showKey)}
            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
          >
            {showKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
          </button>
        </div>
        {apiKey && !prefixValid && (
          <p className="text-xs text-amber-600">API Key 前缀格式不符合预期</p>
        )}
        {apiKey && prefixValid && !lengthValid && (
          <p className="text-xs text-amber-600">API Key 长度不足</p>
        )}
        <p className="text-xs text-muted-foreground">
          API Key 将通过系统密钥环保管，不会以明文存储在本地文件中。
        </p>
      </div>

      {/* 保存按钮 */}
      <button
        onClick={handleSave}
        disabled={loading || !apiKey.trim()}
        className="w-full py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
      >
        {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <KeyRound className="w-4 h-4" />}
        {loading ? '保存中…' : '保存 API Key'}
      </button>

      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  )
}
