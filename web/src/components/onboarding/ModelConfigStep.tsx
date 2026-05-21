import { useState, useCallback, useMemo } from 'react'
import { ProviderTemplateList } from '@/components/provider/ProviderTemplateList'
import { ModelServiceDialog } from '@/components/provider/ModelServiceDialog'
import { useProviderStore } from '@/stores/providerStore'
import { useSettingsStore } from '@/stores/settingsStore'
import { useWails } from '@/hooks/useWails'
import type { ProviderTemplate, ProviderConfig, ProviderModel, AuthParams } from '@/types/provider'
import { CheckCircle2, Server, Cloud } from 'lucide-react'

interface ModelConfigStepProps {
  onComplete: () => void
  onBack: () => void
  onSkip: () => void
}

/**
 * 向导第3步：模型提供商配置。
 * 采用与 SettingsPage 一致的「选择模板 → 配置服务」流程。
 */
export function ModelConfigStep({ onComplete, onBack, onSkip }: ModelConfigStepProps) {
  const { saveAPIKey, createProvider } = useWails()
  const showToast = useCallback((message: string) => {
    console.error('[Toast]', message)
  }, [])
  const addProvider = useProviderStore((s) => s.addProvider)
  const providers = useProviderStore((s) => s.providers)
  const setActiveProviderId = useSettingsStore((s) => s.setActiveProviderId)
  const setActiveModelId = useSettingsStore((s) => s.setActiveModelId)
  const setSelectedModel = useSettingsStore((s) => s.setSelectedModel)

  const [createdProviders, setCreatedProviders] = useState<ProviderConfig[]>([])
  const [serviceDialogOpen, setServiceDialogOpen] = useState(false)
  const [selectedTemplate, setSelectedTemplate] = useState<ProviderTemplate | null>(null)

  // 已有分组列表（给 ModelServiceDialog 用）
  const existingGroups = useMemo(() => {
    const groups = new Set<string>()
    for (const p of providers) {
      groups.add(p.group)
    }
    return Array.from(groups).sort()
  }, [providers])

  const handleSelectTemplate = useCallback((template: ProviderTemplate) => {
    setSelectedTemplate(template)
    setServiceDialogOpen(true)
  }, [])

  const handleSaveService = useCallback(
    async (data: {
      name: string
      apiHost: string
      apiKey: string
      temperature: number
      timeoutMs: number
      maxRetries: number
      group: string
      enabled: boolean
      authMethod: string
      authParams: AuthParams
      models: ProviderModel[]
    }) => {
      const templateId = selectedTemplate?.id || 'custom'
      const now = Date.now()
      const newProvider: ProviderConfig = {
        templateId,
        name: data.name,
        apiHost: data.apiHost,
        apiKey: data.apiKey,
        modelId: data.models[0]?.id || '',
        models: data.models,
        temperature: data.temperature,
        timeoutMs: data.timeoutMs,
        maxRetries: data.maxRetries,
        group: data.group,
        enabled: data.enabled,
        authMethod: data.authMethod as ProviderConfig['authMethod'],
        authParams: data.authParams as ProviderConfig['authParams'],
        sortOrder: 0,
        id: `${templateId}_${now}_${Math.random().toString(36).slice(2, 6)}`,
        createdAt: now,
        updatedAt: now,
      }

      try {
        await createProvider(newProvider)
        addProvider(newProvider)
        setCreatedProviders((prev) => [...prev, newProvider])

        // 保存 API Key 到系统密钥环
        if (data.apiKey && templateId !== 'custom') {
          saveAPIKey(templateId, data.apiKey).catch((err: unknown) => {
            console.error('Failed to save API key:', err)
          })
        }

        // 首次添加时设为活跃 provider
        if (providers.length === 0 && createdProviders.length === 0) {
          setActiveProviderId(newProvider.id)
          const firstModel = newProvider.models?.find((m) => m.enabled)?.id || newProvider.modelId
          setActiveModelId(firstModel)
          setSelectedModel(firstModel)
        }

        setServiceDialogOpen(false)
        setSelectedTemplate(null)
      } catch (err) {
        showToast(`添加失败: ${err instanceof Error ? err.message : String(err)}`)
      }
    },
    [selectedTemplate, addProvider, saveAPIKey, providers.length, createdProviders.length, setActiveProviderId, setActiveModelId, setSelectedModel, createProvider, showToast]
  )

  const handleDialogClose = useCallback(() => {
    setServiceDialogOpen(false)
    setSelectedTemplate(null)
  }, [])

  const isAddedCheck = useCallback(
    (templateId: string) => {
      return (
        providers.some((p) => p.templateId === templateId) ||
        createdProviders.some((p) => p.templateId === templateId)
      )
    },
    [providers, createdProviders]
  )

  return (
    <div className="flex flex-col space-y-5">
      <div className="text-center">
        <h2 className="text-xl font-bold text-foreground mb-1">模型配置</h2>
        <p className="text-sm text-muted-foreground">选择模型提供商并配置服务</p>
      </div>

      {/* 已添加 provider 摘要 */}
      {createdProviders.length > 0 && (
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-xs text-green-700">
            <CheckCircle2 className="w-4 h-4" />
            <span>已配置 {createdProviders.length} 个 Provider</span>
          </div>
          <div className="space-y-1.5">
            {createdProviders.map((p) => (
              <div
                key={p.id}
                className="flex items-center gap-2 px-3 py-2 rounded-lg bg-green-500/5 border border-green-500/20 text-xs"
              >
                {p.group === '本地' ? (
                  <Server className="w-3.5 h-3.5 text-amber-600" />
                ) : (
                  <Cloud className="w-3.5 h-3.5 text-blue-600" />
                )}
                <span className="font-medium text-foreground">{p.name}</span>
                <span className="text-muted-foreground ml-auto">
                  {p.models?.find((m) => m.enabled)?.name || p.modelId}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 提供商模板列表 */}
      <div className="max-h-[40vh] overflow-y-auto pr-1">
        <ProviderTemplateList
          onSelectTemplate={handleSelectTemplate}
          isAddedCheck={isAddedCheck}
        />
      </div>

      {/* 导航按钮 */}
      <div className="flex flex-col gap-2 pt-2">
        <div className="flex gap-3">
          <button
            onClick={onBack}
            className="flex-1 py-2.5 px-4 rounded-lg border border-border text-sm font-medium text-foreground hover:bg-accent transition-colors"
          >
            上一步
          </button>
          <button
            onClick={createdProviders.length > 0 ? onComplete : onSkip}
            className="flex-1 py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
          >
            {createdProviders.length > 0 ? '完成' : '跳过'}
          </button>
        </div>
      </div>

      {/* 模型服务配置弹窗 */}
      <ModelServiceDialog
        mode="add"
        template={selectedTemplate}
        provider={null}
        existingGroups={existingGroups.length > 0 ? existingGroups : ['默认']}
        open={serviceDialogOpen}
        onClose={handleDialogClose}
        onSave={handleSaveService}
      />
    </div>
  )
}
