import { useState, useCallback, useEffect } from 'react'
import { KeyRound, Eye, EyeOff, CheckCircle2, XCircle, Loader2, AlertTriangle } from 'lucide-react'
import { SaveAPIKey, TestAPIKey } from '@wails/go/main/WailsApp'
import { BrowserOpenURL } from '@wails/runtime'
import type { AuthMethodDetectStatus, ProviderConfig } from '@/types/provider'
import templatesData from '@/data/provider-templates.json'
import type { ProviderTemplate } from '@/types/provider'
import { APIKeyGuide } from './APIKeyGuide'

interface APIKeyPanelProps {
  status: AuthMethodDetectStatus | undefined
  onProviderCreated: (provider: ProviderConfig) => void
}

// 从模板中提取支持 api_key 的厂商
const allProviders: ProviderTemplate[] = (templatesData as ProviderTemplate[]).filter((t) =>
  t.authMethods.includes('api_key')
)

// API Key 格式校验正则
const keyPrefixPatterns: Record<string, RegExp> = {
  openai: /^sk-(proj-)?[A-Za-z0-9]{20,}$/,
  kimi: /^sk-[a-f0-9]{48}$/,
  deepseek: /^sk-[a-f0-9]{32}$/,
  claude: /^sk-ant-[a-zA-Z0-9]{32,}$/,
  gemini: /^AIza[A-Za-z0-9_-]{35,}$/,
  qwen: /^sk-[a-f0-9]{32,}$/,
  zhipu: /^[a-f0-9]{32}\.[a-f0-9]{16}$/,
  grok: /^xai-[A-Za-z0-9]{32,}$/,
  doubao: /^[a-f0-9-]{36,}$/,
  minimax: /^[A-Za-z0-9]{32,}$/,
  xiaomi: /^[A-Za-z0-9]{32,}$/,
  hunyuan: /^[A-Za-z0-9]{32,}$/,
}

const clipboardKeyPatterns: Record<string, RegExp> = {
  openai: /^sk-(proj-)?[A-Za-z0-9]{20,}$/,
  kimi: /^sk-[a-f0-9]{48}$/,
  deepseek: /^sk-[a-f0-9]{32}$/,
  claude: /^sk-ant-[a-zA-Z0-9]{32,}$/,
  gemini: /^AIza[A-Za-z0-9_-]{35,}$/,
  qwen: /^sk-[a-f0-9]{32,}$/,
  zhipu: /^[a-f0-9]{32}\.[a-f0-9]{16}$/,
  grok: /^xai-[A-Za-z0-9]{32,}$/,
  doubao: /^[a-f0-9-]{36,}$/,
  minimax: /^[A-Za-z0-9]{32,}$/,
  xiaomi: /^[A-Za-z0-9]{32,}$/,
  hunyuan: /^[A-Za-z0-9]{32,}$/,
}

/** API Key 配置面板 */
export function APIKeyPanel({ status, onProviderCreated }: APIKeyPanelProps) {
  const [selectedProvider, setSelectedProvider] = useState(
    allProviders.find((p) => p.id === 'kimi')?.id || allProviders[0]?.id || 'openai'
  )
  const [apiKey, setApiKey] = useState('')
  const [showKey, setShowKey] = useState(false)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<{ valid: boolean; message: string } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pasteHint, setPasteHint] = useState<string | null>(null)

  const provider = allProviders.find((p) => p.id === selectedProvider)!
  const prefixValid = apiKey ? (keyPrefixPatterns[selectedProvider]?.test(apiKey) ?? true) : true
  const lengthValid = apiKey.length >= 8

  // 智能粘贴：监听窗口聚焦时读取剪贴板
  useEffect(() => {
    const handleFocus = async () => {
      try {
        const text = await navigator.clipboard.readText()
        if (!text || apiKey) return

        // 检测是否匹配当前选中厂商的 API Key 格式
        const pattern = clipboardKeyPatterns[selectedProvider]
        if (pattern && pattern.test(text.trim())) {
          setApiKey(text.trim())
          setPasteHint(`已从剪贴板自动填充 ${provider.name} 的 API Key`)
          setTimeout(() => setPasteHint(null), 4000)
          return
        }

        // 也检测其他厂商格式，自动切换厂商
        for (const [pid, pat] of Object.entries(clipboardKeyPatterns)) {
          if (pid !== selectedProvider && pat.test(text.trim())) {
            setSelectedProvider(pid)
            setApiKey(text.trim())
            const pName = allProviders.find((p) => p.id === pid)?.name ?? pid
            setPasteHint(`检测到 ${pName} 的 API Key，已自动切换并填充`)
            setTimeout(() => setPasteHint(null), 4000)
            break
          }
        }
      } catch {
        // 静默失败：权限拒绝或无剪贴板访问时不报错
      }
    }

    window.addEventListener('focus', handleFocus)
    return () => window.removeEventListener('focus', handleFocus)
  }, [selectedProvider, apiKey, provider?.name])

  const handleTestAndSave = useCallback(async (forceSave = false) => {
    if (!apiKey.trim()) {
      setError('请输入 API Key')
      return
    }
    if (!prefixValid) {
      setError('API Key 前缀格式不正确')
      return
    }

    setError(null)
    setTestResult(null)

    // 强制保存时跳过验证
    if (!forceSave) {
      setTesting(true)
      try {
        const result = await TestAPIKey(selectedProvider, apiKey.trim(), provider.apiHost)
        setTestResult({ valid: result.valid, message: result.message })
        if (!result.valid) {
          setTesting(false)
          return
        }
      } catch (err) {
        setTestResult({ valid: false, message: typeof err === 'string' ? err : (err instanceof Error ? err.message : '验证失败') })
        setTesting(false)
        return
      }
      setTesting(false)
    }

    // 保存到密钥环
    setSaving(true)
    try {
      await SaveAPIKey(selectedProvider, apiKey.trim())

      const newProvider: ProviderConfig = {
        id: `${selectedProvider}_apikey_${Date.now()}`,
        templateId: selectedProvider,
        name: provider.name,
        apiHost: provider.apiHost,
        apiKey: apiKey.trim(),
        modelId: provider.defaultModel,
        temperature: 0.7,
        timeoutMs: 30000,
        maxRetries: 3,
        maxTokens: 4096,
        group: '云端',
        enabled: true,
        sortOrder: 0,
        createdAt: Date.now(),
        updatedAt: Date.now(),
        authMethod: 'api_key',
        authParams: {},
        models: [{ id: provider.defaultModel, name: provider.defaultModel, enabled: true }],
      }
      onProviderCreated(newProvider)
      setApiKey('')
      setTestResult(null)
    } catch (err) {
      setError(typeof err === 'string' ? err : (err instanceof Error ? err.message : '保存失败'))
    } finally {
      setSaving(false)
    }
  }, [apiKey, selectedProvider, prefixValid, provider, onProviderCreated])

  const handleOpenURL = useCallback((url: string) => {
    BrowserOpenURL(url)
  }, [])

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
            setTestResult(null)
            setApiKey('')
          }}
          className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/30"
        >
          {allProviders.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
      </div>

      {/* 模型选择 */}
      {provider.models.length > 0 && (
        <div className="space-y-1.5">
          <label className="text-sm font-medium">模型</label>
          <select
            value={provider.defaultModel}
            onChange={(e) => {
              // 更新 provider 的 defaultModel（通过重新选择）
              const idx = allProviders.findIndex((p) => p.id === selectedProvider)
              if (idx >= 0) {
                allProviders[idx] = { ...allProviders[idx], defaultModel: e.target.value }
              }
            }}
            className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/30"
          >
            {provider.models.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
        </div>
      )}

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
              setTestResult(null)
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
        {pasteHint && (
          <p className="text-xs text-green-600 flex items-center gap-1">
            <CheckCircle2 className="w-3 h-3" />
            {pasteHint}
          </p>
        )}
        <p className="text-xs text-muted-foreground">
          API Key 将通过系统密钥环保管，不会以明文存储在本地文件中。
        </p>
      </div>

      {/* API Key 获取引导 */}
      <APIKeyGuide providerId={selectedProvider} onOpenURL={handleOpenURL} />

      {/* 验证结果提示 */}
      {testResult && !testResult.valid && (
        <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800">
          <AlertTriangle className="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
          <div className="flex-1 min-w-0">
            <p className="text-sm text-amber-800 dark:text-amber-200 font-medium">API Key 验证失败</p>
            <p className="text-xs text-amber-700 dark:text-amber-300 mt-0.5">{testResult.message}</p>
            <button
              type="button"
              onClick={() => handleTestAndSave(true)}
              disabled={saving}
              className="mt-2 text-xs text-amber-800 dark:text-amber-200 underline hover:no-underline disabled:opacity-50"
            >
              仍然保存（跳过验证）
            </button>
          </div>
        </div>
      )}
      {testResult && testResult.valid && (
        <div className="flex items-center gap-2 p-3 rounded-lg bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800">
          <CheckCircle2 className="w-4 h-4 text-green-600 shrink-0" />
          <p className="text-sm text-green-800 dark:text-green-200">{testResult.message}</p>
        </div>
      )}

      {/* 保存按钮 */}
      <button
        onClick={() => handleTestAndSave(false)}
        disabled={testing || saving || !apiKey.trim()}
        className="w-full py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
      >
        {testing || saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <KeyRound className="w-4 h-4" />}
        {testing ? '验证中…' : saving ? '保存中…' : '保存并验证'}
      </button>

      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  )
}
