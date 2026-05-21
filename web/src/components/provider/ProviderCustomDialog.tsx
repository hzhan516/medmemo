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
  PlugZap,
  Loader2,
  CheckCircle,
  XCircle,
  ChevronDown,
  Clock,
} from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { ProviderConfig, AuthMethod, AuthParams } from '@/types/provider'

const providerFormSchema = z.object({
  name: z.string().min(1, '名称不能为空'),
  apiHost: z.string().regex(/^https?:\/\//, '必须以 http:// 或 https:// 开头'),
  apiKey: z.string(),
  modelId: z.string().min(1, 'Model ID 不能为空'),
  temperature: z.number().min(0).max(2),
  timeoutMs: z.number().min(1000).max(300000),
  maxRetries: z.number().min(0).max(10),
  group: z.string().min(1, '分组不能为空'),
  enabled: z.boolean(),
})

type ProviderFormData = z.infer<typeof providerFormSchema>

type TestStatus = 'idle' | 'testing' | 'green' | 'yellow' | 'red'

interface TestRecord {
  status: Exclude<TestStatus, 'idle' | 'testing'>
  latencyMs: number
  error?: string
  checkedAt: number
}

interface ProviderCustomDialogProps {
  mode: 'custom' | 'edit'
  provider: ProviderConfig | null
  existingGroups: string[]
  open: boolean
  onClose: () => void
  onSave: (data: ProviderFormData & { id?: string; createdAt?: number; authMethod: AuthMethod; authParams: AuthParams }) => void
}

const defaultValues: ProviderFormData = {
  name: '',
  apiHost: '',
  apiKey: '',
  modelId: '',
  temperature: 0.7,
  timeoutMs: 30000,
  maxRetries: 3,
  group: '默认',
  enabled: true,
}

const authMethodOptions: { value: AuthMethod; label: string }[] = [
  { value: 'api_key', label: 'API Key' },
  { value: 'cli_token', label: 'CLI Token' },
  { value: 'oauth_device', label: 'OAuth Device Flow' },
  { value: 'service_account', label: 'Service Account' },
]

/**
 * 格式化相对时间（"刚刚"、"N秒前"、"N分钟前"）。
 */
function formatRelativeTime(ts: number): string {
  const diff = Math.floor((Date.now() - ts) / 1000)
  if (diff < 5) return '刚刚'
  if (diff < 60) return `${diff} 秒前`
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
  return `${Math.floor(diff / 3600)} 小时前`
}

/**
 * 自定义 Provider 表单弹窗。
 * 支持从零创建（custom 模式）或编辑已有 Provider（edit 模式）。
 * 使用 react-hook-form + zod 做实时验证。
 * 集成测试连接功能：fetch /v1/models 验证连通性并拉取模型列表。
 */
export function ProviderCustomDialog({
  mode,
  provider,
  existingGroups,
  open,
  onClose,
  onSave,
}: ProviderCustomDialogProps) {
  const [showKey, setShowKey] = useState(false)
  const [showUnsavedWarning, setShowUnsavedWarning] = useState(false)
  const [groupInputMode, setGroupInputMode] = useState<'select' | 'input'>('select')

  // 认证方式相关状态
  const [authMethod, setAuthMethod] = useState<AuthMethod>('api_key')
  const [authParams, setAuthParams] = useState<AuthParams>({})

  // 测试连接相关状态
  const [testStatus, setTestStatus] = useState<TestStatus>('idle')
  const [testLatencyMs, setTestLatencyMs] = useState(0)
  const [testError, setTestError] = useState('')
  const [fetchedModels, setFetchedModels] = useState<Array<{ id: string; name: string }>>([])
  const [modelSelectMode, setModelSelectMode] = useState<'dropdown' | 'manual'>('manual')
  const [testHistory, setTestHistory] = useState<TestRecord[]>([])

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    getValues,
    formState: { errors, isDirty, isValid },
  } = useForm<ProviderFormData>({
    resolver: zodResolver(providerFormSchema),
    defaultValues,
    mode: 'onChange',
  })

  // 弹窗打开时初始化表单与测试状态
  useEffect(() => {
    if (open) {
      setShowKey(false)
      setShowUnsavedWarning(false)
      setGroupInputMode('select')
      setTestStatus('idle')
      setTestLatencyMs(0)
      setTestError('')
      setFetchedModels([])
      setModelSelectMode('manual')
      setTestHistory([])
      if (mode === 'edit' && provider) {
        reset({
          name: provider.name,
          apiHost: provider.apiHost,
          apiKey: provider.apiKey,
          modelId: provider.modelId,
          temperature: provider.temperature,
          timeoutMs: provider.timeoutMs,
          maxRetries: provider.maxRetries,
          group: provider.group,
          enabled: provider.enabled,
        })
        setAuthMethod(provider.authMethod || 'api_key')
        setAuthParams(provider.authParams || {})
      } else {
        reset(defaultValues)
        setAuthMethod('api_key')
        setAuthParams({})
      }
    }
  }, [open, mode, provider, reset])

  const temperature = watch('temperature')
  const group = watch('group')
  const currentModelId = watch('modelId')

  // 测试连接
  const handleTestConnection = useCallback(async () => {
    const apiHost = getValues('apiHost')
    const apiKey = getValues('apiKey')

    if (!apiHost || !/^https?:\/\//.test(apiHost)) {
      setTestStatus('red')
      setTestError('请先填写有效的 API Host（以 http:// 或 https:// 开头）')
      return
    }

    setTestStatus('testing')
    setTestError('')

    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 2000)
    const startTime = Date.now()

    let finalStatus: Exclude<TestStatus, 'idle' | 'testing'> = 'red'
    let finalLatency = 0
    let finalError = ''
    let finalModels: Array<{ id: string; name: string }> = []
    let finalSelectMode: 'dropdown' | 'manual' = 'manual'

    try {
      const resp = await fetch(`${apiHost.replace(/\/$/, '')}/v1/models`, {
        headers: apiKey ? { Authorization: `Bearer ${apiKey}` } : {},
        signal: controller.signal,
      })
      clearTimeout(timeoutId)
      const latency = Date.now() - startTime
      finalLatency = latency

      if (resp.status === 200) {
        const data = await resp.json()
        const models = (data.data || []).map((m: { id?: string; name?: string }) => ({
          id: m.id || '',
          name: m.name || m.id || '',
        }))
        finalModels = models

        if (latency >= 1000) {
          finalStatus = 'yellow'
        } else {
          finalStatus = 'green'
        }

        if (models.length > 0) {
          finalSelectMode = 'dropdown'
          // 若当前 Model ID 为空，自动填入第一个
          if (!getValues('modelId')) {
            setValue('modelId', models[0].id, { shouldValidate: true })
          }
        }
      } else if (resp.status === 404) {
        finalStatus = 'green'
        finalError = '该 Provider 不支持自动获取模型列表，请手动输入 Model ID'
      } else if (resp.status === 401 || resp.status === 403) {
        finalStatus = 'red'
        finalError = 'API Key 无效，请检查密钥是否正确'
      } else {
        finalStatus = 'red'
        finalError = `连接失败（HTTP ${resp.status}），请检查 API Host`
      }
    } catch (err) {
      clearTimeout(timeoutId)
      if (err instanceof Error && err.name === 'AbortError') {
        finalStatus = 'red'
        finalError = '连接超时，请检查网络或 API Host 是否正确'
      } else {
        finalStatus = 'red'
        finalError = '连接失败，请检查网络或 API Host 是否正确'
      }
    }

    setTestStatus(finalStatus)
    setTestLatencyMs(finalLatency)
    setTestError(finalError)
    setFetchedModels(finalModels)
    setModelSelectMode(finalSelectMode)
    setTestHistory((prev) =>
      [
        {
          status: finalStatus,
          latencyMs: finalLatency,
          error: finalError || undefined,
          checkedAt: Date.now(),
        },
        ...prev,
      ].slice(0, 3)
    )
  }, [getValues, setValue])

  // 从下拉选择模型
  const handleSelectModel = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      const val = e.target.value
      setValue('modelId', val, { shouldValidate: true })
    },
    [setValue]
  )

  // 分组选择变更
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

  // 关闭弹窗，检查未保存
  const handleClose = useCallback(() => {
    if (isDirty) {
      setShowUnsavedWarning(true)
    } else {
      onClose()
    }
  }, [isDirty, onClose])

  const handleConfirmClose = useCallback(() => {
    setShowUnsavedWarning(false)
    onClose()
  }, [onClose])

  const handleCancelClose = useCallback(() => {
    setShowUnsavedWarning(false)
  }, [])

  const onSubmit = useCallback(
    (data: ProviderFormData) => {
      if (mode === 'edit' && provider) {
        onSave({ ...data, id: provider.id, createdAt: provider.createdAt, authMethod, authParams })
      } else {
        onSave({ ...data, authMethod, authParams })
      }
      onClose()
    },
    [mode, provider, onSave, onClose, authMethod, authParams]
  )

  const title = mode === 'edit' ? '编辑 Provider' : '添加自定义 Provider'
  const Icon = mode === 'edit' ? Pencil : Plus

  // 状态卡片样式映射
  const statusCardStyles = {
    green: 'border-emerald-500 bg-emerald-50 text-emerald-700',
    yellow: 'border-amber-500 bg-amber-50 text-amber-700',
    red: 'border-red-500 bg-red-50 text-red-700',
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      {/* 未保存警告层 */}
      {showUnsavedWarning && (
        <div className="absolute inset-0 z-10 flex items-center justify-center bg-black/30">
          <div className="w-full max-w-sm mx-4 rounded-xl border border-border bg-card shadow-xl p-5 space-y-4">
            <div className="flex items-center gap-2 text-amber-600">
              <AlertTriangle className="w-5 h-5" />
              <h3 className="font-semibold text-sm">有未保存的变更</h3>
            </div>
            <p className="text-sm text-muted-foreground">
              确定要离开吗？已修改的内容将不会保存。
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={handleCancelClose}
                className="px-3 py-1.5 rounded-lg border border-border text-sm hover:bg-accent transition-colors"
              >
                继续编辑
              </button>
              <button
                onClick={handleConfirmClose}
                className="px-3 py-1.5 rounded-lg bg-destructive text-destructive-foreground text-sm hover:bg-destructive/90 transition-colors"
              >
                放弃保存
              </button>
            </div>
          </div>
        </div>
      )}

      <div
        className="w-full max-w-lg mx-4 rounded-xl border border-border bg-card shadow-xl max-h-[90vh] overflow-y-auto"
        role="dialog"
        aria-modal="true"
        data-testid="provider-custom-dialog"
      >
        {/* 头部 */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-border sticky top-0 bg-card z-10">
          <div className="flex items-center gap-2">
            <Icon className="w-4 h-4 text-primary" />
            <h2 className="text-base font-semibold text-foreground">{title}</h2>
          </div>
          <button
            onClick={handleClose}
            className="p-1.5 rounded-md hover:bg-accent transition-colors"
            aria-label="关闭"
          >
            <X className="w-4 h-4 text-muted-foreground" />
          </button>
        </div>

        {/* 表单 */}
        <form onSubmit={handleSubmit(onSubmit)} className="px-5 py-4 space-y-4">
          {/* 名称 */}
          <div className="space-y-1.5">
            <Label htmlFor="pc-name">
              名称 <span className="text-destructive">*</span>
            </Label>
            <Input
              id="pc-name"
              {...register('name')}
              placeholder="例如：我的 OpenAI"
              data-testid="pc-name-input"
            />
            {errors.name && (
              <p className="text-xs text-destructive">{errors.name.message}</p>
            )}
          </div>

          {/* API Host */}
          <div className="space-y-1.5">
            <Label htmlFor="pc-host">
              API Host <span className="text-destructive">*</span>
            </Label>
            <Input
              id="pc-host"
              {...register('apiHost')}
              placeholder="https://api.example.com"
              data-testid="pc-host-input"
            />
            {errors.apiHost && (
              <p className="text-xs text-destructive">{errors.apiHost.message}</p>
            )}
          </div>

          {/* 认证方式 */}
          <div className="space-y-1.5">
            <Label htmlFor="pc-auth-method">认证方式</Label>
            <select
              id="pc-auth-method"
              value={authMethod}
              onChange={(e) => {
                setAuthMethod(e.target.value as AuthMethod)
                setAuthParams({})
              }}
              className="w-full h-9 px-3 rounded-md border border-input bg-background text-sm focus:outline-none focus:ring-2 focus:ring-ring"
              data-testid="pc-auth-method-select"
            >
              {authMethodOptions.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          {/* API Key（api_key 方式） */}
          {authMethod === 'api_key' && (
            <div className="space-y-1.5">
              <Label htmlFor="pc-key">API Key</Label>
              <div className="relative">
                <Input
                  id="pc-key"
                  type={showKey ? 'text' : 'password'}
                  {...register('apiKey')}
                  placeholder="sk-..."
                  className="pr-10"
                  data-testid="pc-key-input"
                />
                <button
                  type="button"
                  onClick={() => setShowKey(!showKey)}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                  aria-label={showKey ? '隐藏 API Key' : '显示 API Key'}
                >
                  {showKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
              <p className="text-[11px] text-muted-foreground">
                API Key 由系统密钥环保管，不以明文存储。
              </p>
            </div>
          )}

          {/* CLI Token 凭证路径（cli_token 方式） */}
          {authMethod === 'cli_token' && (
            <div className="space-y-1.5">
              <Label htmlFor="pc-cli-path">CLI 凭证路径 <span className="text-destructive">*</span></Label>
              <Input
                id="pc-cli-path"
                value={authParams.cliCredentialPath || ''}
                onChange={(e) => setAuthParams((prev) => ({ ...prev, cliCredentialPath: e.target.value }))}
                placeholder="~/.kimi/credentials/kimi-code.json"
                data-testid="pc-cli-path-input"
              />
              <p className="text-[11px] text-muted-foreground">
                指向 CLI 工具缓存的 OAuth token 文件路径。
              </p>
            </div>
          )}

          {/* OAuth Device Flow 配置（oauth_device 方式） */}
          {authMethod === 'oauth_device' && (
            <div className="space-y-3">
              <div className="space-y-1.5">
                <Label htmlFor="pc-oauth-client-id">Client ID <span className="text-destructive">*</span></Label>
                <Input
                  id="pc-oauth-client-id"
                  value={authParams.oauthClientId || ''}
                  onChange={(e) => setAuthParams((prev) => ({ ...prev, oauthClientId: e.target.value }))}
                  placeholder="your-client-id"
                  data-testid="pc-oauth-client-id-input"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="pc-oauth-auth-url">Auth URL</Label>
                <Input
                  id="pc-oauth-auth-url"
                  value={authParams.oauthAuthUrl || ''}
                  onChange={(e) => setAuthParams((prev) => ({ ...prev, oauthAuthUrl: e.target.value }))}
                  placeholder="https://auth.example.com/authorize"
                  data-testid="pc-oauth-auth-url-input"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="pc-oauth-token-url">Token URL <span className="text-destructive">*</span></Label>
                <Input
                  id="pc-oauth-token-url"
                  value={authParams.oauthTokenUrl || ''}
                  onChange={(e) => setAuthParams((prev) => ({ ...prev, oauthTokenUrl: e.target.value }))}
                  placeholder="https://auth.example.com/token"
                  data-testid="pc-oauth-token-url-input"
                />
              </div>
            </div>
          )}

          {/* Service Account 配置（service_account 方式） */}
          {authMethod === 'service_account' && (
            <div className="space-y-3">
              <div className="space-y-1.5">
                <Label htmlFor="pc-sa-project">GCP Project ID <span className="text-destructive">*</span></Label>
                <Input
                  id="pc-sa-project"
                  value={authParams.gcpProjectId || ''}
                  onChange={(e) => setAuthParams((prev) => ({ ...prev, gcpProjectId: e.target.value }))}
                  placeholder="my-project-123"
                  data-testid="pc-sa-project-input"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="pc-sa-region">Region</Label>
                <Input
                  id="pc-sa-region"
                  value={authParams.gcpRegion || ''}
                  onChange={(e) => setAuthParams((prev) => ({ ...prev, gcpRegion: e.target.value }))}
                  placeholder="us-central1"
                  data-testid="pc-sa-region-input"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="pc-sa-json">Service Account JSON <span className="text-destructive">*</span></Label>
                <textarea
                  id="pc-sa-json"
                  value={authParams.saJson || ''}
                  onChange={(e) => setAuthParams((prev) => ({ ...prev, saJson: e.target.value }))}
                  placeholder='{"type":"service_account","project_id":"..."}'
                  rows={4}
                  className="w-full px-3 py-2 rounded-md border border-input bg-background text-sm focus:outline-none focus:ring-2 focus:ring-ring resize-y font-mono"
                  data-testid="pc-sa-json-input"
                />
              </div>
            </div>
          )}

          {/* 测试连接 */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label className="text-sm">连接测试</Label>
              <button
                type="button"
                onClick={handleTestConnection}
                disabled={testStatus === 'testing'}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-border text-xs font-medium hover:bg-accent transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                data-testid="pc-test-connection-btn"
              >
                {testStatus === 'testing' ? (
                  <>
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    测试中…
                  </>
                ) : (
                  <>
                    <PlugZap className="w-3.5 h-3.5" />
                    测试连接
                  </>
                )}
              </button>
            </div>

            {/* 结果卡片 */}
            {testStatus !== 'idle' && testStatus !== 'testing' && (
              <div
                className={`rounded-lg border p-3 text-sm ${statusCardStyles[testStatus]}`}
                data-testid="pc-test-result-card"
                data-test-status={testStatus}
              >
                <div className="flex items-center gap-2">
                  {testStatus === 'green' && <CheckCircle className="w-4 h-4 shrink-0" />}
                  {testStatus === 'yellow' && <AlertTriangle className="w-4 h-4 shrink-0" />}
                  {testStatus === 'red' && <XCircle className="w-4 h-4 shrink-0" />}
                  <span className="font-medium">
                    {testStatus === 'green' && `连通，延迟 ${testLatencyMs}ms`}
                    {testStatus === 'yellow' && `连通但延迟较高，${testLatencyMs}ms`}
                    {testStatus === 'red' && (testError || '无法连接')}
                  </span>
                </div>
              </div>
            )}

            {/* 测试历史 */}
            {testHistory.length > 0 && (
              <div className="space-y-1.5" data-testid="pc-test-history">
                <p className="text-[11px] text-muted-foreground font-medium">最近测试结果</p>
                <div className="space-y-1">
                  {testHistory.map((record, idx) => (
                    <div
                      key={`${record.checkedAt}-${idx}`}
                      className="flex items-center gap-2 text-xs"
                      data-testid={`pc-test-history-item-${idx}`}
                    >
                      <span
                        className={`w-2 h-2 rounded-full shrink-0 ${
                          record.status === 'green'
                            ? 'bg-emerald-500'
                            : record.status === 'yellow'
                              ? 'bg-amber-500'
                              : 'bg-red-500'
                        }`}
                      />
                      <span className="text-muted-foreground">
                        {record.status === 'green' && `连通 ${record.latencyMs}ms`}
                        {record.status === 'yellow' && `延迟高 ${record.latencyMs}ms`}
                        {record.status === 'red' && (record.error || '无法连接')}
                      </span>
                      <span className="text-muted-foreground/60 ml-auto flex items-center gap-0.5">
                        <Clock className="w-3 h-3" />
                        {formatRelativeTime(record.checkedAt)}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Model ID */}
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <Label htmlFor="pc-model">
                Model ID <span className="text-destructive">*</span>
              </Label>
              {fetchedModels.length > 0 && (
                <div className="flex items-center gap-1">
                  <button
                    type="button"
                    onClick={() => setModelSelectMode('dropdown')}
                    className={`text-[11px] px-1.5 py-0.5 rounded transition-colors ${
                      modelSelectMode === 'dropdown'
                        ? 'bg-primary text-primary-foreground'
                        : 'text-muted-foreground hover:bg-accent'
                    }`}
                    data-testid="pc-model-mode-dropdown"
                  >
                    列表选择
                  </button>
                  <button
                    type="button"
                    onClick={() => setModelSelectMode('manual')}
                    className={`text-[11px] px-1.5 py-0.5 rounded transition-colors ${
                      modelSelectMode === 'manual'
                        ? 'bg-primary text-primary-foreground'
                        : 'text-muted-foreground hover:bg-accent'
                    }`}
                    data-testid="pc-model-mode-manual"
                  >
                    手动输入
                  </button>
                </div>
              )}
            </div>

            {modelSelectMode === 'dropdown' && fetchedModels.length > 0 ? (
              <div className="relative">
                <select
                  id="pc-model"
                  value={currentModelId}
                  onChange={handleSelectModel}
                  className="w-full h-9 px-3 pr-8 rounded-md border border-input bg-background text-sm focus:outline-none focus:ring-2 focus:ring-ring appearance-none"
                  data-testid="pc-model-select"
                >
                  <option value="">请选择模型</option>
                  {fetchedModels.map((m) => (
                    <option key={m.id} value={m.id}>
                      {m.name} ({m.id})
                    </option>
                  ))}
                </select>
                <ChevronDown className="absolute right-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
              </div>
            ) : (
              <Input
                id="pc-model"
                {...register('modelId')}
                placeholder={
                  testStatus === 'green' && fetchedModels.length === 0
                    ? '该 Provider 不支持自动获取，请手动输入'
                    : '例如：gpt-4o'
                }
                data-testid="pc-model-input"
              />
            )}
            {errors.modelId && (
              <p className="text-xs text-destructive">{errors.modelId.message}</p>
            )}
          </div>

          {/* 温度参数 */}
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <Label htmlFor="pc-temp">温度参数</Label>
              <span className="text-xs text-muted-foreground font-mono">{temperature.toFixed(1)}</span>
            </div>
            <input
              id="pc-temp"
              type="range"
              min={0}
              max={2}
              step={0.1}
              {...register('temperature', { valueAsNumber: true })}
              className="w-full accent-primary"
              data-testid="pc-temp-input"
            />
            <div className="flex justify-between text-[10px] text-muted-foreground">
              <span>精确</span>
              <span>平衡</span>
              <span>创意</span>
            </div>
            {errors.temperature && (
              <p className="text-xs text-destructive">{errors.temperature.message}</p>
            )}
          </div>

          {/* 超时 + 重试 并排 */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="pc-timeout">超时时间（毫秒）</Label>
              <Input
                id="pc-timeout"
                type="number"
                {...register('timeoutMs', { valueAsNumber: true })}
                data-testid="pc-timeout-input"
              />
              {errors.timeoutMs && (
                <p className="text-xs text-destructive">{errors.timeoutMs.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="pc-retries">重试次数</Label>
              <Input
                id="pc-retries"
                type="number"
                {...register('maxRetries', { valueAsNumber: true })}
                data-testid="pc-retries-input"
              />
              {errors.maxRetries && (
                <p className="text-xs text-destructive">{errors.maxRetries.message}</p>
              )}
            </div>
          </div>

          {/* 分组 */}
          <div className="space-y-1.5">
            <Label htmlFor="pc-group">
              分组 <span className="text-destructive">*</span>
            </Label>
            {groupInputMode === 'select' ? (
              <select
                id="pc-group"
                value={group}
                onChange={handleGroupChange}
                className="w-full h-9 px-3 rounded-md border border-input bg-background text-sm focus:outline-none focus:ring-2 focus:ring-ring"
                data-testid="pc-group-select"
              >
                {existingGroups.map((g) => (
                  <option key={g} value={g}>
                    {g}
                  </option>
                ))}
                <option value="__new__">+ 创建新分组</option>
              </select>
            ) : (
              <div className="flex gap-2">
                <Input
                  {...register('group')}
                  placeholder="输入新分组名称"
                  autoFocus
                  data-testid="pc-group-input"
                />
                <button
                  type="button"
                  onClick={() => {
                    setGroupInputMode('select')
                    if (existingGroups.length > 0) {
                      setValue('group', existingGroups[0], { shouldValidate: true })
                    }
                  }}
                  className="px-3 py-1.5 rounded-lg border border-border text-sm whitespace-nowrap hover:bg-accent transition-colors"
                >
                  选已有
                </button>
              </div>
            )}
            {errors.group && (
              <p className="text-xs text-destructive">{errors.group.message}</p>
            )}
          </div>

          {/* 启用开关 */}
          <div className="flex items-center justify-between p-3 rounded-lg border border-border">
            <div>
              <div className="text-sm font-medium">启用该 Provider</div>
              <div className="text-xs text-muted-foreground">禁用的 Provider 不会出现在模型切换列表中</div>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" {...register('enabled')} className="sr-only peer" data-testid="pc-enabled-input" />
              <div className="w-10 h-5 bg-muted peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-primary" />
            </label>
          </div>

          {/* 编辑模式：上次修改时间 */}
          {mode === 'edit' && provider && (
            <p className="text-[11px] text-muted-foreground text-right">
              上次修改：{new Date(provider.updatedAt).toLocaleString('zh-CN')}
            </p>
          )}

          {/* 底部按钮 */}
          <div className="flex items-center justify-end gap-2 pt-2 border-t border-border sticky bottom-0 bg-card pb-1">
            <button
              type="button"
              onClick={handleClose}
              className="px-4 py-2 rounded-lg border border-border text-sm font-medium text-foreground hover:bg-accent transition-colors"
            >
              取消
            </button>
            <button
              type="submit"
              disabled={!isValid}
              className="px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              data-testid="pc-save-btn"
            >
              保存
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
