import { useState, useEffect, useCallback } from 'react'
import { Eye, EyeOff, X, Server } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { ProviderTemplate, ProviderConfig } from '@/types/provider'

interface ProviderAddDialogProps {
  template: ProviderTemplate | null
  open: boolean
  onClose: () => void
  onSave: (config: Omit<ProviderConfig, 'id' | 'createdAt' | 'updatedAt'>) => void
}

/** Provider 一键添加弹窗 */
export function ProviderAddDialog({ template, open, onClose, onSave }: ProviderAddDialogProps) {
  const [apiKey, setApiKey] = useState('')
  const [showKey, setShowKey] = useState(false)
  const [temperature, setTemperature] = useState(0.7)
  const [error, setError] = useState<string | null>(null)

  // 弹窗打开时重置表单
  useEffect(() => {
    if (open) {
      setApiKey('')
      setShowKey(false)
      setTemperature(0.7)
      setError(null)
    }
  }, [open])

  const isLocal = template?.type === 'local'

  const handleSave = useCallback(() => {
    if (!template) return

    if (!isLocal && !apiKey.trim()) {
      setError('请输入 API Key')
      return
    }

    onSave({
      templateId: template.id,
      name: template.name,
      apiHost: template.apiHost,
      apiKey: apiKey.trim(),
      modelId: template.defaultModel,
      temperature,
      timeoutMs: 30000,
      maxRetries: 3,
      group: isLocal ? '本地' : '云端',
      enabled: true,
      sortOrder: 0,
      authMethod: 'api_key',
      authParams: {},
      models: [{ id: template.defaultModel, name: template.defaultModel, enabled: true }],
    })

    onClose()
  }, [template, isLocal, apiKey, temperature, onSave, onClose])

  if (!open || !template) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div
        className="w-full max-w-md mx-4 rounded-xl border border-border bg-card shadow-xl"
        role="dialog"
        aria-modal="true"
        data-testid="provider-add-dialog"
      >
        {/* 头部 */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <div className="flex items-center gap-2">
            <h2 className="text-base font-semibold text-foreground">添加 Provider</h2>
            {isLocal && (
              <span className="text-[10px] font-medium px-1.5 py-0.5 rounded-full bg-amber-500/10 text-amber-600">
                本地
              </span>
            )}
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-md hover:bg-accent transition-colors"
            aria-label="关闭"
          >
            <X className="w-4 h-4 text-muted-foreground" />
          </button>
        </div>

        {/* 表单 */}
        <div className="px-5 py-4 space-y-4">
          {/* 名称（只读） */}
          <div className="space-y-1.5">
            <Label htmlFor="provider-name">名称</Label>
            <Input id="provider-name" value={template.name} readOnly className="bg-muted" />
          </div>

          {/* API Host（只读） */}
          <div className="space-y-1.5">
            <Label htmlFor="provider-host">API Host</Label>
            <Input id="provider-host" value={template.apiHost} readOnly className="bg-muted" />
          </div>

          {/* Model ID（只读） */}
          <div className="space-y-1.5">
            <Label htmlFor="provider-model">Model ID</Label>
            <Input id="provider-model" value={template.defaultModel} readOnly className="bg-muted" />
            {template.models.length > 0 && (
              <p className="text-[11px] text-muted-foreground">
                可用模型: {template.models.join(', ')}
              </p>
            )}
          </div>

          {/* API Key（用户输入） */}
          {!isLocal && (
            <div className="space-y-1.5">
              <Label htmlFor="provider-apikey">API Key</Label>
              <div className="relative">
                <Input
                  id="provider-apikey"
                  type={showKey ? 'text' : 'password'}
                  value={apiKey}
                  onChange={(e) => {
                    setApiKey(e.target.value)
                    setError(null)
                  }}
                  placeholder="请输入您的 API Key"
                  className="pr-10"
                  data-testid="provider-apikey-input"
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
                API Key 将通过系统密钥环保管，不会以明文存储在本地文件中。
              </p>
            </div>
          )}

          {/* 本地 Provider 提示 */}
          {isLocal && (
            <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-500/5 border border-amber-500/20">
              <Server className="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
              <p className="text-xs text-amber-700">
                本地 Provider 无需 API Key，请确保服务已在本地运行。
              </p>
            </div>
          )}

          {/* 温度参数 */}
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <Label htmlFor="provider-temp">温度参数</Label>
              <span className="text-xs text-muted-foreground font-mono">{temperature.toFixed(1)}</span>
            </div>
            <input
              id="provider-temp"
              type="range"
              min={0}
              max={2}
              step={0.1}
              value={temperature}
              onChange={(e) => setTemperature(parseFloat(e.target.value))}
              className="w-full accent-primary"
            />
            <div className="flex justify-between text-[10px] text-muted-foreground">
              <span>精确</span>
              <span>平衡</span>
              <span>创意</span>
            </div>
          </div>

          {/* 错误提示 */}
          {error && (
            <p className="text-xs text-destructive" data-testid="provider-add-error">
              {error}
            </p>
          )}
        </div>

        {/* 底部按钮 */}
        <div className="flex items-center justify-end gap-2 px-5 py-4 border-t border-border">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-lg border border-border text-sm font-medium text-foreground hover:bg-accent transition-colors"
          >
            取消
          </button>
          <button
            onClick={handleSave}
            className="px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
            data-testid="provider-save-btn"
          >
            保存
          </button>
        </div>
      </div>
    </div>
  )
}
