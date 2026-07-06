import { useState, useEffect, useCallback } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  Eye,
  EyeOff,
  X,
  AlertTriangle,
  Pencil,
  Plus,
  Loader2,
  CheckCircle,
  XCircle,
  Clock,
  Trash2,
  Settings2,
  ListChecks,
  RefreshCw,
} from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { ProviderConfig, AuthMethod, AuthParams, ProviderModel, ProviderTemplate } from '@/types/provider'

const serviceFormSchema = z.object({
  name: z.string().min(1, '名称不能为空'),
  apiHost: z.string().regex(/^https?:\/\//, '必须以 http:// 或 https:// 开头'),
  apiKey: z.string(),
  temperature: z.number().min(0).max(2),
  timeoutMs: z.number().min(1000).max(300000),
  maxRetries: z.number().min(0).max(10),
  maxTokens: z.number().min(256).max(32768),
  group: z.string().min(1, '分组不能为空'),
  enabled: z.boolean(),
})

type ServiceFormData = z.infer<typeof serviceFormSchema>

type TestStatus = 'idle' | 'testing' | 'green' | 'yellow' | 'red'

interface TestRecord {
  status: Exclude<TestStatus, 'idle' | 'testing'>
  latencyMs: number
  error?: string
  checkedAt: number
}

interface ModelServiceDialogProps {
  mode: 'add' | 'edit'
  /** 新建时传入模板，编辑时传入已有 Provider */
  template?: ProviderTemplate | null
  provider?: ProviderConfig | null
  existingGroups: string[]
  open: boolean
  onClose: () => void
  onSave: (data: ServiceFormData & { id?: string; createdAt?: number; authMethod: AuthMethod; authParams: AuthParams; models: ProviderModel[] }) => void
}

const defaultValues: ServiceFormData = {
  name: '',
  apiHost: '',
  apiKey: '',
  temperature: 0.7,
  timeoutMs: 30000,
  maxRetries: 3,
  maxTokens: 4096,
  group: '默认',
  enabled: true,
}

const authMethodOptions: { value: AuthMethod; label: string }[] = [
  { value: 'api_key', label: 'API Key' },
  { value: 'cli_token', label: 'CLI Token' },
  { value: 'oauth_device', label: 'OAuth Device Flow' },
  { value: 'service_account', label: 'Service Account' },
]

function formatRelativeTime(ts: number): string {
  const diff = Math.floor((Date.now() - ts) / 1000)
  if (diff < 5) return '刚刚'
  if (diff < 60) return `${diff} 秒前`
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
  return `${Math.floor(diff / 3600)} 小时前`
}

/**
 * CherryStudio 风格的模型服务配置弹窗。
 * 双标签页：服务设置 + 模型管理。
 * 新建时从模板预填，编辑时加载已有配置。
 */
export function ModelServiceDialog({
  mode,
  template,
  provider,
  existingGroups,
  open,
  onClose,
  onSave,
}: ModelServiceDialogProps) {
  const [activeTab, setActiveTab] = useState<'service' | 'models'>('service')
  const [showKey, setShowKey] = useState(false)
  const [showUnsavedWarning, setShowUnsavedWarning] = useState(false)
  const [groupInputMode, setGroupInputMode] = useState<'select' | 'input'>('select')

  const [authMethod, setAuthMethod] = useState<AuthMethod>('api_key')
  const [authParams, setAuthParams] = useState<AuthParams>({})

  const [testStatus, setTestStatus] = useState<TestStatus>('idle')
  const [testLatencyMs, setTestLatencyMs] = useState(0)
  const [testError, setTestError] = useState('')
  const [testHistory, setTestHistory] = useState<TestRecord[]>([])

  const [models, setModels] = useState<ProviderModel[]>([])
  const [newModelId, setNewModelId] = useState('')
  const [newModelName, setNewModelName] = useState('')

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    getValues,
    formState: { errors, isDirty, isValid },
  } = useForm<ServiceFormData>({
    resolver: zodResolver(serviceFormSchema),
    defaultValues,
    mode: 'onChange',
  })

  useEffect(() => {
    if (!open) return
    setActiveTab('service')
    setShowKey(false)
    setShowUnsavedWarning(false)
    setGroupInputMode('select')
    setTestStatus('idle')
    setTestLatencyMs(0)
    setTestError('')
    setTestHistory([])
    setNewModelId('')
    setNewModelName('')

    if (mode === 'edit' && provider) {
      reset({
        name: provider.name,
        apiHost: provider.apiHost,
        apiKey: provider.apiKey,
        temperature: provider.temperature,
        timeoutMs: provider.timeoutMs,
        maxRetries: provider.maxRetries,
        maxTokens: provider.maxTokens ?? 4096,
        group: provider.group,
        enabled: provider.enabled,
      })
      setAuthMethod(provider.authMethod || 'api_key')
      setAuthParams(provider.authParams || {})
      setModels(provider.models && provider.models.length > 0 ? [...provider.models] : provider.modelId ? [{ id: provider.modelId, name: provider.modelId, enabled: true }] : [])
    } else if (template) {
      reset({
        name: template.name,
        apiHost: template.apiHost,
        apiKey: '',
        temperature: 0.7,
        timeoutMs: 30000,
        maxRetries: 3,
        maxTokens: 4096,
        group: template.type === 'local' ? '本地' : '云端',
        enabled: true,
      })
      setAuthMethod(template.authMethods?.[0] || 'api_key')
      setAuthParams({})
      setModels(template.defaultModel ? [{ id: template.defaultModel, name: template.defaultModel, enabled: true }] : [])
    } else {
      reset(defaultValues)
      setAuthMethod('api_key')
      setAuthParams({})
      setModels([])
    }
  }, [open, mode, provider, template, reset])

  const temperature = watch('temperature')
  const group = watch('group')
  const apiHostValue = watch('apiHost')

  const handleTestConnection = useCallback(async () => {
    const apiHost = getValues('apiHost')
    const apiKey = getValues('apiKey')

    if (!apiHost || !/^https?:\/\//.test(apiHost)) {
      setTestStatus('red')
      setTestError('请先填写有效的 API Host')
      return
    }

    setTestStatus('testing')
    setTestError('')

    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 10000)
    const startTime = Date.now()

    let finalStatus: Exclude<TestStatus, 'idle' | 'testing'> = 'red'
    let finalLatency = 0
    let finalError = ''
    let finalModels: ProviderModel[] = []

    try {
      const resp = await fetch(`${apiHost.replace(/\/$/, '')}/v1/models`, {
        headers: apiKey ? { Authorization: `Bearer ${apiKey}` } : {},
        signal: controller.signal,
      })
      const latency = Date.now() - startTime
      finalLatency = latency

      if (resp.status === 200) {
        const data = await resp.json()
        const fetched = (data.data || []).map((m: { id?: string; name?: string }) => ({
          id: m.id || '',
          name: m.name || m.id || '',
          enabled: true,
        }))
        finalModels = fetched
        finalStatus = latency >= 1000 ? 'yellow' : 'green'
      } else if (resp.status === 404) {
        finalStatus = 'green'
        finalError = '该服务不支持自动获取模型列表，可手动添加'
      } else if (resp.status === 401 || resp.status === 403) {
        finalStatus = 'red'
        finalError = '认证失败，请检查 API Key'
      } else {
        finalStatus = 'red'
        finalError = `连接失败（HTTP ${resp.status}）`
      }
    } catch (err) {
      if (err instanceof Error && err.name === 'AbortError') {
        finalStatus = 'red'
        finalError = '连接超时'
      } else {
        finalStatus = 'red'
        finalError = '连接失败，请检查网络或服务地址'
      }
    } finally {
      clearTimeout(timeoutId)
    }

    setTestStatus(finalStatus)
    setTestLatencyMs(finalLatency)
    setTestError(finalError)
    if (finalModels.length > 0) {
      setModels(finalModels)
    }
    setTestHistory((prev) =>
      [{ status: finalStatus, latencyMs: finalLatency, error: finalError || undefined, checkedAt: Date.now() }, ...prev].slice(0, 3)
    )
  }, [getValues])

  const handleAddModel = useCallback(() => {
    const id = newModelId.trim()
    const name = newModelName.trim() || id
    if (!id) return
    if (models.some((m) => m.id === id)) {
      setNewModelId('')
      setNewModelName('')
      return
    }
    setModels((prev) => [...prev, { id, name, enabled: true }])
    setNewModelId('')
    setNewModelName('')
  }, [newModelId, newModelName, models])

  const toggleModelEnabled = useCallback((id: string) => {
    setModels((prev) => prev.map((m) => (m.id === id ? { ...m, enabled: !m.enabled } : m)))
  }, [])

  const removeModel = useCallback((id: string) => {
    setModels((prev) => prev.filter((m) => m.id !== id))
  }, [])

  const parseMaxContextLength = (value: string): number | undefined => {
    if (value === '') return undefined
    const num = Number(value)
    if (!Number.isFinite(num) || !Number.isInteger(num)) return NaN
    return num
  }

  const isInvalidMaxContextLength = (value?: number) =>
    value !== undefined && (Number.isNaN(value) || value < 256 || value > 2000000)

  const updateModelMaxContextLength = useCallback((id: string, value: string) => {
    const num = parseMaxContextLength(value)
    setModels((prev) => prev.map((m) => (m.id === id ? { ...m, maxContextLength: num } : m)))
  }, [])

  const handleGroupChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      const val = e.target.value
      if (val === '__new__') {
        setGroupInputMode('input')
        setValue('group', '', { shouldValidate: true })
      } else {
        setValue('group', val, { shouldValidate: true })
      }
    },
    [setValue]
  )

  const handleClose = useCallback(() => {
    if (isDirty || models.length > 0) {
      setShowUnsavedWarning(true)
    } else {
      onClose()
    }
  }, [isDirty, models.length, onClose])

  const handleConfirmClose = useCallback(() => {
    setShowUnsavedWarning(false)
    onClose()
  }, [onClose])

  const onSubmit = useCallback(
    (data: ServiceFormData) => {
      if (models.some((m) => isInvalidMaxContextLength(m.maxContextLength))) {
        setActiveTab('models')
        return
      }
      if (mode === 'edit' && provider) {
        onSave({ ...data, id: provider.id, createdAt: provider.createdAt, authMethod, authParams, models })
      } else {
        onSave({ ...data, authMethod, authParams, models })
      }
      onClose()
    },
    [mode, provider, onSave, onClose, authMethod, authParams, models]
  )

  const title = mode === 'edit' ? '编辑模型服务' : template ? `添加 ${template.name}` : '添加模型服务'
  const Icon = mode === 'edit' ? Pencil : Plus

  const statusCardStyles = {
    green: 'border-emerald-500 bg-emerald-50 text-emerald-700',
    yellow: 'border-amber-500 bg-amber-50 text-amber-700',
    red: 'border-red-500 bg-red-50 text-red-700',
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      {/* 未保存警告 */}
      {showUnsavedWarning && (
        <div className="absolute inset-0 z-10 flex items-center justify-center bg-black/30">
          <div className="w-full max-w-sm mx-4 rounded-xl border border-border bg-card shadow-xl p-5 space-y-4">
            <div className="flex items-center gap-2 text-amber-600">
              <AlertTriangle className="w-5 h-5" />
              <h3 className="font-semibold text-sm">有未保存的变更</h3>
            </div>
            <p className="text-sm text-muted-foreground">确定要离开吗？已修改的内容将不会保存。</p>
            <div className="flex justify-end gap-2">
              <button onClick={() => setShowUnsavedWarning(false)} className="px-3 py-1.5 rounded-lg border border-border text-sm hover:bg-accent transition-colors">
                继续编辑
              </button>
              <button onClick={handleConfirmClose} className="px-3 py-1.5 rounded-lg bg-destructive text-destructive-foreground text-sm hover:bg-destructive/90 transition-colors">
                放弃保存
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="w-full max-w-lg mx-4 rounded-xl border border-border/60 bg-background/95 backdrop-blur-xl shadow-xl max-h-[90vh] overflow-hidden flex flex-col" role="dialog" aria-modal="true" data-testid="model-service-dialog">
        {/* 头部 */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-border/60 shrink-0">
          <div className="flex items-center gap-2">
            <Icon className="w-4 h-4 text-primary" />
            <h2 className="text-base font-semibold text-foreground">{title}</h2>
          </div>
          <button onClick={handleClose} className="p-1.5 rounded-md hover:bg-accent transition-colors" aria-label="关闭">
            <X className="w-4 h-4 text-muted-foreground" />
          </button>
        </div>

        {/* Tab 切换 */}
        <div className="flex border-b border-border shrink-0">
          <button
            onClick={() => setActiveTab('service')}
            className={`flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium transition-colors border-b-2 ${
              activeTab === 'service' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
            data-testid="tab-service"
          >
            <Settings2 className="w-3.5 h-3.5" />
            服务设置
          </button>
          <button
            onClick={() => setActiveTab('models')}
            className={`flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium transition-colors border-b-2 ${
              activeTab === 'models' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
            data-testid="tab-models"
          >
            <ListChecks className="w-3.5 h-3.5" />
            模型管理
            {models.length > 0 && (
              <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-muted text-muted-foreground">{models.length}</span>
            )}
          </button>
        </div>

        {/* 内容区 */}
        <form onSubmit={handleSubmit(onSubmit)} className="flex-1 overflow-y-auto">
          {activeTab === 'service' && (
            <div className="px-5 py-4 space-y-4">
              {/* 名称 */}
              <div className="space-y-1.5">
                <Label htmlFor="ms-name">名称 <span className="text-destructive">*</span></Label>
                <Input id="ms-name" {...register('name')} placeholder="例如：我的 Kimi" data-testid="ms-name-input" />
                {errors.name && <p className="text-xs text-destructive">{errors.name.message}</p>}
              </div>

              {/* API Host */}
              <div className="space-y-1.5">
                <Label htmlFor="ms-host">API Host <span className="text-destructive">*</span></Label>
                <Input id="ms-host" {...register('apiHost')} placeholder="https://api.example.com" data-testid="ms-host-input" />
                {errors.apiHost && <p className="text-xs text-destructive">{errors.apiHost.message}</p>}
                {apiHostValue?.startsWith('http://') && (
                  <p className="text-xs text-amber-600 flex items-center gap-1">
                    <AlertTriangle className="w-3 h-3" />
                    HTTP 传输将导致 API Key 以明文暴露，建议改用 HTTPS
                  </p>
                )}
              </div>

              {/* 认证方式 */}
              <div className="space-y-1.5">
                <Label htmlFor="ms-auth-method">认证方式</Label>
                <select
                  id="ms-auth-method"
                  value={authMethod}
                  onChange={(e) => { setAuthMethod(e.target.value as AuthMethod); setAuthParams({}) }}
                  className="w-full h-9 px-3 rounded-md border border-input bg-background text-sm focus:outline-none focus:ring-2 focus:ring-ring"
                  data-testid="ms-auth-method-select"
                >
                  {authMethodOptions.map((opt) => (
                    <option key={opt.value} value={opt.value}>{opt.label}</option>
                  ))}
                </select>
              </div>

              {/* API Key */}
              {authMethod === 'api_key' && (
                <div className="space-y-1.5">
                  <Label htmlFor="ms-key">API Key</Label>
                  <div className="relative">
                    <Input id="ms-key" type={showKey ? 'text' : 'password'} {...register('apiKey')} placeholder="sk-..." className="pr-10" data-testid="ms-key-input" maxLength={1024} />
                    <button type="button" onClick={() => setShowKey(!showKey)} className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors">
                      {showKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </button>
                  </div>
                  <p className="text-[11px] text-muted-foreground">API Key 由系统密钥环保管，不以明文存储。</p>
                </div>
              )}

              {/* CLI Token */}
              {authMethod === 'cli_token' && (
                <div className="space-y-1.5">
                  <Label htmlFor="ms-cli-path">CLI 凭证路径 <span className="text-destructive">*</span></Label>
                  <Input id="ms-cli-path" value={authParams.cliCredentialPath || ''} onChange={(e) => setAuthParams((prev) => ({ ...prev, cliCredentialPath: e.target.value }))} placeholder="~/.kimi/credentials/kimi-code.json" />
                  <p className="text-[11px] text-muted-foreground">指向 CLI 工具缓存的 OAuth token 文件路径。</p>
                </div>
              )}

              {/* OAuth Device Flow */}
              {authMethod === 'oauth_device' && (
                <div className="space-y-3">
                  <div className="space-y-1.5">
                    <Label htmlFor="ms-oauth-client-id">Client ID <span className="text-destructive">*</span></Label>
                    <Input id="ms-oauth-client-id" value={authParams.oauthClientId || ''} onChange={(e) => setAuthParams((prev) => ({ ...prev, oauthClientId: e.target.value }))} placeholder="your-client-id" />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="ms-oauth-auth-url">Auth URL</Label>
                    <Input id="ms-oauth-auth-url" value={authParams.oauthAuthUrl || ''} onChange={(e) => setAuthParams((prev) => ({ ...prev, oauthAuthUrl: e.target.value }))} placeholder="https://auth.example.com/authorize" />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="ms-oauth-token-url">Token URL <span className="text-destructive">*</span></Label>
                    <Input id="ms-oauth-token-url" value={authParams.oauthTokenUrl || ''} onChange={(e) => setAuthParams((prev) => ({ ...prev, oauthTokenUrl: e.target.value }))} placeholder="https://auth.example.com/token" />
                  </div>
                </div>
              )}

              {/* Service Account */}
              {authMethod === 'service_account' && (
                <div className="space-y-3">
                  <div className="space-y-1.5">
                    <Label htmlFor="ms-sa-project">GCP Project ID <span className="text-destructive">*</span></Label>
                    <Input id="ms-sa-project" value={authParams.gcpProjectId || ''} onChange={(e) => setAuthParams((prev) => ({ ...prev, gcpProjectId: e.target.value }))} placeholder="my-project-123" />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="ms-sa-region">Region</Label>
                    <Input id="ms-sa-region" value={authParams.gcpRegion || ''} onChange={(e) => setAuthParams((prev) => ({ ...prev, gcpRegion: e.target.value }))} placeholder="us-central1" />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="ms-sa-json">Service Account JSON <span className="text-destructive">*</span></Label>
                    <textarea id="ms-sa-json" value={authParams.saJson || ''} onChange={(e) => setAuthParams((prev) => ({ ...prev, saJson: e.target.value }))} placeholder='{"type":"service_account",...}' rows={4}
                      className="w-full px-3 py-2 rounded-md border border-input bg-background text-sm focus:outline-none focus:ring-2 focus:ring-ring resize-y font-mono" />
                  </div>
                </div>
              )}

              {/* 测试连接 */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label className="text-sm">连接测试</Label>
                  <button type="button" onClick={handleTestConnection} disabled={testStatus === 'testing'}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-border text-xs font-medium hover:bg-accent transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    data-testid="ms-test-connection-btn"
                  >
                    {testStatus === 'testing' ? <><Loader2 className="w-3.5 h-3.5 animate-spin" />测试中…</> : <><RefreshCw className="w-3.5 h-3.5" />获取模型列表</>}
                  </button>
                </div>
                {testStatus !== 'idle' && testStatus !== 'testing' && (
                  <div className={`rounded-lg border p-3 text-sm ${statusCardStyles[testStatus]}`} data-testid="ms-test-result-card" data-test-status={testStatus}>
                    <div className="flex items-center gap-2">
                      {testStatus === 'green' && <CheckCircle className="w-4 h-4 shrink-0" />}
                      {testStatus === 'yellow' && <AlertTriangle className="w-4 h-4 shrink-0" />}
                      {testStatus === 'red' && <XCircle className="w-4 h-4 shrink-0" />}
                      <span className="font-medium">
                        {testStatus === 'green' && `连通，延迟 ${testLatencyMs}ms，获取 ${models.length} 个模型`}
                        {testStatus === 'yellow' && `连通但延迟较高，${testLatencyMs}ms`}
                        {testStatus === 'red' && (testError || '无法连接')}
                      </span>
                    </div>
                  </div>
                )}
                {testHistory.length > 0 && (
                  <div className="space-y-1" data-testid="ms-test-history">
                    {testHistory.map((record, idx) => (
                      <div key={`${record.checkedAt}-${idx}`} className="flex items-center gap-2 text-xs">
                        <span className={`w-2 h-2 rounded-full shrink-0 ${record.status === 'green' ? 'bg-emerald-500' : record.status === 'yellow' ? 'bg-amber-500' : 'bg-red-500'}`} />
                        <span className="text-muted-foreground">{record.status === 'green' ? `连通 ${record.latencyMs}ms` : record.status === 'yellow' ? `延迟高 ${record.latencyMs}ms` : (record.error || '无法连接')}</span>
                        <span className="text-muted-foreground/60 ml-auto flex items-center gap-0.5"><Clock className="w-3 h-3" />{formatRelativeTime(record.checkedAt)}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* 温度参数 */}
              <div className="space-y-1.5">
                <div className="flex items-center justify-between">
                  <Label htmlFor="ms-temp">温度参数</Label>
                  <span className="text-xs text-muted-foreground font-mono">{temperature.toFixed(1)}</span>
                </div>
                <input id="ms-temp" type="range" min={0} max={2} step={0.1} {...register('temperature', { valueAsNumber: true })} className="w-full accent-primary" />
                <div className="flex justify-between text-[10px] text-muted-foreground"><span>精确</span><span>平衡</span><span>创意</span></div>
              </div>

              {/* 超时 + 重试 + 最大 Token 数 */}
              <div className="grid grid-cols-3 gap-4">
                <div className="space-y-1.5">
                  <Label htmlFor="ms-timeout">超时时间（毫秒）</Label>
                  <Input id="ms-timeout" type="number" {...register('timeoutMs', { valueAsNumber: true })} data-testid="ms-timeout-input" />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="ms-retries">重试次数</Label>
                  <Input id="ms-retries" type="number" {...register('maxRetries', { valueAsNumber: true })} data-testid="ms-retries-input" />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="ms-max-tokens">最大输出 Token</Label>
                  <Input id="ms-max-tokens" type="number" {...register('maxTokens', { valueAsNumber: true })} data-testid="ms-max-tokens-input" />
                </div>
              </div>

              {/* 分组 */}
              <div className="space-y-1.5">
                <Label htmlFor="ms-group">分组 <span className="text-destructive">*</span></Label>
                {groupInputMode === 'select' ? (
                  <select id="ms-group" data-testid="ms-group-select" value={group} onChange={handleGroupChange} className="w-full h-9 px-3 rounded-md border border-input bg-background text-sm focus:outline-none focus:ring-2 focus:ring-ring">
                    {existingGroups.map((g) => (<option key={g} value={g}>{g}</option>))}
                    <option value="__new__">+ 创建新分组</option>
                  </select>
                ) : (
                  <div className="flex gap-2">
                    <Input {...register('group')} placeholder="输入新分组名称" autoFocus data-testid="ms-group-input" />
                    <button type="button" onClick={() => { setGroupInputMode('select'); if (existingGroups.length > 0) setValue('group', existingGroups[0], { shouldValidate: true }) }}
                      className="px-3 py-1.5 rounded-lg border border-border text-sm whitespace-nowrap hover:bg-accent transition-colors">选已有</button>
                  </div>
                )}
                {errors.group && <p className="text-xs text-destructive">{errors.group.message}</p>}
              </div>

              {/* 启用开关 */}
              <div className="flex items-center justify-between p-3 rounded-lg border border-border">
                <div>
                  <div className="text-sm font-medium">启用该服务</div>
                  <div className="text-xs text-muted-foreground">禁用的服务不会出现在模型切换列表中</div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input type="checkbox" {...register('enabled')} className="sr-only peer" data-testid="ms-enabled-input" />
                  <div className="w-10 h-5 bg-muted peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-primary" />
                </label>
              </div>

              {mode === 'edit' && provider && (
                <p className="text-[11px] text-muted-foreground text-right">上次修改：{new Date(provider.updatedAt).toLocaleString('zh-CN')}</p>
              )}
            </div>
          )}

          {activeTab === 'models' && (
            <div className="px-5 py-4 space-y-4">
              {/* 模型统计 */}
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-medium">模型列表</h3>
                  <p className="text-xs text-muted-foreground">共 {models.length} 个模型，{models.filter((m) => m.enabled).length} 个已启用</p>
                </div>
                <button type="button" onClick={handleTestConnection} disabled={testStatus === 'testing'}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-border text-xs font-medium hover:bg-accent transition-colors disabled:opacity-50"
                  data-testid="ms-fetch-models-btn"
                >
                  {testStatus === 'testing' ? <Loader2 className="w-3 h-3 animate-spin" /> : <RefreshCw className="w-3 h-3" />}
                  从 API 获取
                </button>
              </div>

              {/* 模型列表 */}
              <div className="space-y-1.5 max-h-[320px] overflow-y-auto">
                {models.length === 0 ? (
                  <div className="text-center py-8 text-sm text-muted-foreground border border-dashed border-border rounded-lg">
                    暂无模型，请点击「从 API 获取」或手动添加
                  </div>
                ) : (
                  models.map((m) => (
                    <div key={m.id} className="flex items-start gap-3 px-3 py-2 rounded-lg border border-border/60 hover:bg-accent/30 transition-colors">
                      <input type="checkbox" checked={m.enabled} onChange={() => toggleModelEnabled(m.id)} className="shrink-0 w-4 h-4 mt-2 rounded border-gray-300 text-primary focus:ring-primary" data-testid={`ms-model-check-${m.id}`} />
                      <div className="flex-1 min-w-0 py-0.5">
                        <div className={`text-sm font-medium truncate ${m.enabled ? 'text-foreground' : 'text-muted-foreground line-through'}`}>{m.name}</div>
                        <div className="text-[11px] text-muted-foreground font-mono truncate">{m.id}</div>
                      </div>
                      <div className="w-28 shrink-0 space-y-1">
                        <label htmlFor={`ms-max-ctx-${m.id}`} className="block text-[10px] text-muted-foreground">最大上下文长度</label>
                        <Input
                          id={`ms-max-ctx-${m.id}`}
                          type="number"
                          min={256}
                          max={2000000}
                          value={m.maxContextLength ?? ''}
                          onChange={(e) => updateModelMaxContextLength(m.id, e.target.value)}
                          className="h-7 text-xs"
                          placeholder="默认"
                          data-testid={`ms-max-ctx-input-${m.id}`}
                        />
                        {isInvalidMaxContextLength(m.maxContextLength) && (
                          <p className="text-[10px] text-destructive">请输入 256–2000000 之间的数值</p>
                        )}
                      </div>
                      <button type="button" onClick={() => removeModel(m.id)} className="p-1 mt-1.5 rounded-md text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors" title="删除">
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  ))
                )}
              </div>

              {/* 手动添加 */}
              <div className="space-y-2 pt-2 border-t border-border">
                <p className="text-xs font-medium text-muted-foreground">手动添加模型</p>
                <div className="flex gap-2">
                  <Input placeholder="模型 ID（如 gpt-4o）" value={newModelId} onChange={(e) => setNewModelId(e.target.value)} className="flex-1" data-testid="ms-new-model-id" />
                  <Input placeholder="显示名称（可选）" value={newModelName} onChange={(e) => setNewModelName(e.target.value)} className="flex-1" data-testid="ms-new-model-name" />
                  <button type="button" onClick={handleAddModel} disabled={!newModelId.trim()}
                    className="px-3 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
                    data-testid="ms-add-model-btn"
                  >
                    <Plus className="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* 底部按钮 */}
          <div className="flex items-center justify-end gap-2 px-5 py-4 border-t border-border bg-card shrink-0">
            <button type="button" onClick={handleClose} className="px-4 py-2 rounded-lg border border-border text-sm font-medium text-foreground hover:bg-accent transition-colors">
              取消
            </button>
            <button type="submit" disabled={!isValid} className="px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed" data-testid="ms-save-btn">
              保存
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
