import { useState, useEffect, useCallback } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Eye, EyeOff, X, AlertTriangle, Pencil, Plus } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { ProviderConfig } from '@/types/provider'

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

interface ProviderCustomDialogProps {
  mode: 'custom' | 'edit'
  provider: ProviderConfig | null
  existingGroups: string[]
  open: boolean
  onClose: () => void
  onSave: (data: ProviderFormData & { id?: string; createdAt?: number }) => void
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

/**
 * 自定义 Provider 表单弹窗。
 * 支持从零创建（custom 模式）或编辑已有 Provider（edit 模式）。
 * 使用 react-hook-form + zod 做实时验证。
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

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors, isDirty, isValid },
  } = useForm<ProviderFormData>({
    resolver: zodResolver(providerFormSchema),
    defaultValues,
    mode: 'onChange',
  })

  // 弹窗打开时初始化表单
  useEffect(() => {
    if (open) {
      setShowKey(false)
      setShowUnsavedWarning(false)
      setGroupInputMode('select')
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
      } else {
        reset(defaultValues)
      }
    }
  }, [open, mode, provider, reset])

  const temperature = watch('temperature')
  const group = watch('group')

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
        onSave({ ...data, id: provider.id, createdAt: provider.createdAt })
      } else {
        onSave(data)
      }
      onClose()
    },
    [mode, provider, onSave, onClose]
  )

  const title = mode === 'edit' ? '编辑 Provider' : '添加自定义 Provider'
  const Icon = mode === 'edit' ? Pencil : Plus

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

          {/* API Key */}
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
              本地模型可留空。API Key 由系统密钥环保管，不以明文存储。
            </p>
          </div>

          {/* Model ID */}
          <div className="space-y-1.5">
            <Label htmlFor="pc-model">
              Model ID <span className="text-destructive">*</span>
            </Label>
            <Input
              id="pc-model"
              {...register('modelId')}
              placeholder="例如：gpt-4o"
              data-testid="pc-model-input"
            />
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
