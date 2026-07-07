import { useMemo } from 'react'
import type { ProviderConfig } from '@/types/provider'
import type { CompressionSettings } from '@/stores/settingsStore'

interface Props {
  value: CompressionSettings
  providers: ProviderConfig[]
  onChange: (next: CompressionSettings) => void
  onTest: () => Promise<void>
  testStatus?: 'idle' | 'testing' | 'ok' | 'error'
}

export function CompressionSettingsForm({
  value,
  providers,
  onChange,
  onTest,
  testStatus = 'idle',
}: Props) {
  const enabledProviders = useMemo(
    () => providers.filter((p) => p.enabled),
    [providers]
  )

  const selectedProvider = useMemo(
    () => enabledProviders.find((p) => p.id === value.providerId),
    [enabledProviders, value.providerId]
  )

  const modelOptions = useMemo(() => {
    if (!selectedProvider) return []
    return selectedProvider.models?.filter((m) => m.enabled) ?? []
  }, [selectedProvider])

  return (
    <div className="space-y-4">
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={value.useModel}
          onChange={(e) => onChange({ ...value, useModel: e.target.checked })}
        />
        <span>启用模型压缩摘要</span>
      </label>

      {value.useModel && (
        <div className="pl-6 space-y-3 text-sm">
          <p className="text-muted-foreground">
            选择用于生成摘要的模型。本地模型（Ollama）数据不出设备；云端模型会先脱敏再发送，仅上传占位符。
            未选择或不可用时自动使用内置的无模型摘要。
          </p>

          <div className="space-y-1">
            <label className="text-muted-foreground">Provider</label>
            <select
              className="w-full rounded border px-2 py-1 bg-background"
              value={value.providerId}
              onChange={(e) =>
                onChange({
                  ...value,
                  providerId: e.target.value,
                  modelId: '',
                })
              }
            >
              <option value="">（复用当前会话模型）</option>
              {enabledProviders.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </div>

          {value.providerId && (
            <div className="space-y-1">
              <label className="text-muted-foreground">Model</label>
              <select
                className="w-full rounded border px-2 py-1 bg-background"
                value={value.modelId}
                onChange={(e) =>
                  onChange({
                    ...value,
                    modelId: e.target.value,
                  })
                }
              >
                <option value="">（使用 provider 默认模型）</option>
                {modelOptions.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.name}
                  </option>
                ))}
              </select>
            </div>
          )}

          <button
            type="button"
            className="text-primary underline disabled:opacity-50"
            onClick={() => void onTest()}
            disabled={testStatus === 'testing' || !value.providerId}
          >
            {testStatus === 'testing' ? '测试中…' : '测试连接'}
          </button>
          {testStatus === 'ok' && (
            <span className="ml-2 text-emerald-500 text-xs">连接成功</span>
          )}
          {testStatus === 'error' && (
            <span className="ml-2 text-red-500 text-xs">连接失败</span>
          )}
        </div>
      )}
    </div>
  )
}
