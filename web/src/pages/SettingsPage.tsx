import { useState, useMemo, useCallback, useEffect } from 'react'
import { logger } from '@/lib/logger'
import { useSettingsStore } from '@/stores/settingsStore'
import { useOnboardingStore } from '@/stores/onboardingStore'
import { useProviderStore } from '@/stores/providerStore'
import { useTheme } from '@/hooks/useTheme'
import { useWails } from '@/hooks/useWails'
import { Card, CardContent } from '@/components/ui/card'
import {
  ProviderTemplateList,
  ModelServiceDialog,
  ProviderGroupList,
  DeleteConfirmDialog,
  ProviderImportExport,
} from '@/components/provider'
import type { ProviderTemplate, ProviderConfig, ProviderModel, AuthParams } from '@/types/provider'
import {
  Monitor, Moon, Sun, Check, Bell, BellDot, BellOff,
  RefreshCw, Shield, FlaskConical, ShieldCheck, ShieldOff,
  Eye, Trash2, RotateCcw, Plus, FileJson,
  Brain, FolderOpen, Sparkles, AlertCircle, ExternalLink,
} from 'lucide-react'

/**
 * 设置页面：支持主题切换、模型选择、Provider 管理与合规提示条模式。
 * 使用 shadcn/ui 组件验证 light/dark 主题兼容性。
 */
export function SettingsPage() {
  const { theme, setTheme } = useTheme()
  const {
    complianceBarMode, setComplianceBarMode,
    showConfidenceBar, setShowConfidenceBar,
    confidenceBarMode, setConfidenceBarMode,
    autoCheckUpdate, setAutoCheckUpdate,
    updateChannel, setUpdateChannel: setUpdateChannelStore,
    desensitizationLevel, setDesensitizationLevel,
    dataRetentionDays, setDataRetentionDays,
    activeProviderId, setActiveProviderId,
  } = useSettingsStore()
  const onboardingCompleted = useOnboardingStore((s) => s.completed)
  const analytics = useOnboardingStore((s) => s.analytics)
  const resetOnboarding = useOnboardingStore((s) => s.reset)
  const clearAnalytics = useOnboardingStore((s) => s.clearAnalytics)
  const [showAnalytics, setShowAnalytics] = useState(false)

  // 模型服务弹窗状态（统一处理添加和编辑）
  const [serviceDialogOpen, setServiceDialogOpen] = useState(false)
  const [serviceDialogMode, setServiceDialogMode] = useState<'add' | 'edit'>('add')
  const [serviceDialogTemplate, setServiceDialogTemplate] = useState<ProviderTemplate | null>(null)
  const [editingProvider, setEditingProvider] = useState<ProviderConfig | null>(null)

  // 删除确认弹窗状态
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [deletingProvider, setDeletingProvider] = useState<ProviderConfig | null>(null)

  // 导入导出弹窗状态
  const [importExportOpen, setImportExportOpen] = useState(false)

  const providers = useProviderStore((s) => s.providers)
  const addProvider = useProviderStore((s) => s.addProvider)
  const updateProvider = useProviderStore((s) => s.updateProvider)
  const removeProvider = useProviderStore((s) => s.removeProvider)
  const hasProvider = useProviderStore((s) => s.hasProvider)
  const { saveAPIKey, createProvider, updateProvider: updateProviderApi, deleteProvider: deleteProviderApi, setUpdateSettings, getEmbeddingStatus, getEmbeddingModelDirPath, openEmbeddingModelDir } = useWails()

  const [embeddingStatus, setEmbeddingStatus] = useState<{ available: boolean; model_path: string; model_name: string; download_url: string } | null>(null)
  const [modelDirPath, setModelDirPath] = useState<string>('')
  const [toastMsg, setToastMsg] = useState<string | null>(null)

  useEffect(() => {
    getEmbeddingStatus()
      .then((status) => setEmbeddingStatus(status))
      .catch((err) => logger.error('Failed to get embedding status:', err))
    getEmbeddingModelDirPath()
      .then((path) => setModelDirPath(path))
      .catch((err) => logger.error('Failed to get model dir path:', err))
  }, [getEmbeddingStatus, getEmbeddingModelDirPath])

  const showToast = useCallback((message: string) => {
    setToastMsg(message)
    setTimeout(() => setToastMsg(null), 3000)
  }, [])

  /**
   * 切换更新通道时同步到后端 updater 服务。
   */
  const setUpdateChannel = useCallback(
    async (channel: 'stable' | 'beta') => {
      setUpdateChannelStore(channel)
      try {
        await setUpdateSettings({
          check_enabled: autoCheckUpdate,
          channel,
          skip_version: '',
        })
      } catch (err) {
        logger.error('Failed to sync update channel to backend:', err)
      }
    },
    [setUpdateChannelStore, setUpdateSettings, autoCheckUpdate]
  )

  // 已有分组列表
  const existingGroups = useMemo(() => {
    const groups = new Set<string>()
    for (const p of providers) {
      groups.add(p.group)
    }
    return Array.from(groups).sort()
  }, [providers])

  const handleSelectTemplate = (template: ProviderTemplate) => {
    setServiceDialogTemplate(template)
    setServiceDialogMode('add')
    setEditingProvider(null)
    setServiceDialogOpen(true)
  }

  const handleOpenCustomDialog = useCallback(() => {
    setServiceDialogTemplate(null)
    setServiceDialogMode('add')
    setEditingProvider(null)
    setServiceDialogOpen(true)
  }, [])

  const handleEditProvider = useCallback((provider: ProviderConfig) => {
    setServiceDialogTemplate(null)
    setServiceDialogMode('edit')
    setEditingProvider(provider)
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
      maxTokens: number
      group: string
      enabled: boolean
      id?: string
      createdAt?: number
      authMethod: string
      authParams: AuthParams
      models: ProviderModel[]
    }) => {
      const providerData = {
        name: data.name,
        apiHost: data.apiHost,
        apiKey: data.apiKey,
        modelId: data.models[0]?.id || '',
        models: data.models,
        temperature: data.temperature,
        timeoutMs: data.timeoutMs,
        maxRetries: data.maxRetries,
        maxTokens: data.maxTokens,
        group: data.group,
        enabled: data.enabled,
        authMethod: data.authMethod as ProviderConfig['authMethod'],
        authParams: data.authParams as ProviderConfig['authParams'],
      }
      if (serviceDialogMode === 'edit' && data.id) {
        const updated: ProviderConfig = {
          ...editingProvider!,
          ...providerData,
          id: data.id,
          createdAt: data.createdAt ?? editingProvider!.createdAt,
          updatedAt: Date.now(),
        }
        try {
          await updateProviderApi(updated)
          updateProvider(updated)
          showToast('Provider 已更新')
        } catch (err) {
          showToast(`更新失败: ${err instanceof Error ? err.message : String(err)}`)
        }
      } else {
        const templateId = serviceDialogTemplate?.id || 'custom'
        const now = Date.now()
        const newProvider: ProviderConfig = {
          templateId,
          ...providerData,
          id: `${templateId}_${now}_${Math.random().toString(36).slice(2, 6)}`,
          sortOrder: 0,
          createdAt: now,
          updatedAt: now,
        }
        try {
          await createProvider(newProvider)
          addProvider(newProvider)
          if (providers.length === 0) {
            setActiveProviderId(newProvider.id)
          }
          if (data.apiKey && templateId !== 'custom') {
            saveAPIKey(templateId, data.apiKey).catch((err) => {
              logger.error('Failed to save API key:', err)
            })
          }
          showToast('Provider 已添加')
        } catch (err) {
          showToast(`添加失败: ${err instanceof Error ? err.message : String(err)}`)
        }
      }
    },
    [serviceDialogMode, editingProvider, addProvider, updateProvider, providers.length, setActiveProviderId, serviceDialogTemplate, saveAPIKey, createProvider, updateProviderApi, showToast]
  )

  const handleDeleteClick = useCallback((provider: ProviderConfig) => {
    setDeletingProvider(provider)
    setDeleteDialogOpen(true)
  }, [])

  const handleConfirmDelete = useCallback(async () => {
    if (deletingProvider) {
      try {
        await deleteProviderApi(deletingProvider.id)
        removeProvider(deletingProvider.id)
        // 如果删除的是活跃 Provider，清空 activeProviderId
        if (activeProviderId === deletingProvider.id) {
          setActiveProviderId(null)
        }
        showToast('Provider 已删除')
      } catch (err) {
        showToast(`删除失败: ${err instanceof Error ? err.message : String(err)}`)
      }
      setDeleteDialogOpen(false)
      setDeletingProvider(null)
    }
  }, [deletingProvider, removeProvider, activeProviderId, setActiveProviderId, deleteProviderApi, showToast])

  const themes = [
    { id: 'light' as const, label: '亮色', icon: Sun },
    { id: 'dark' as const, label: '暗色', icon: Moon },
    { id: 'system' as const, label: '跟随系统', icon: Monitor },
  ]

  const complianceModes = [
    { id: 'always' as const, label: '始终展示', icon: BellDot, desc: '每次进入会话都展示，可手动关闭' },
    { id: 'first' as const, label: '首次展示', icon: Bell, desc: '新会话首次进入时展示，关闭后不再显示' },
    { id: 'off' as const, label: '关闭', icon: BellOff, desc: '完全不展示合规提示条' },
  ]

  const updateChannels = [
    { id: 'stable' as const, label: '稳定版', icon: Shield, desc: '仅接收正式版本更新' },
    { id: 'beta' as const, label: '测试版', icon: FlaskConical, desc: '包含预发布版本，优先体验新功能' },
  ]

  const desensitizationLevels = [
    { id: 'standard' as const, label: '标准', icon: Shield, desc: '规则脱敏 + NER 模型识别' },
    { id: 'strict' as const, label: '严格', icon: ShieldCheck, desc: '三重脱敏兜底，最大程度保护' },
    { id: 'off' as const, label: '关闭', icon: ShieldOff, desc: '不进行脱敏，明文传输' },
  ]

  const retentionOptions = [
    { value: 7, label: '7 天' },
    { value: 30, label: '30 天' },
    { value: 90, label: '90 天' },
    { value: 365, label: '1 年' },
    { value: 0, label: '永久保留' },
  ]

  return (
    <div className="h-full flex flex-col bg-background">
      <div className="h-14 flex items-center px-4 border-b border-border">
        <h1 className="text-lg font-semibold">设置</h1>
      </div>

      <div className="flex-1 overflow-y-auto p-6 max-w-2xl mx-auto w-full space-y-8">
        {/* 主题设置 */}
        <section>
          <h2 className="text-sm font-medium text-muted-foreground mb-3 uppercase tracking-wider">
            外观
          </h2>
          <div className="grid grid-cols-3 gap-3">
            {themes.map((t) => {
              const Icon = t.icon
              const isActive = theme === t.id
              return (
                <Card
                  key={t.id}
                  className={`cursor-pointer transition-all ${
                    isActive
                      ? 'border-primary bg-primary/5'
                      : 'border-border hover:border-primary/30 hover:bg-accent'
                  }`}
                  onClick={() => setTheme(t.id)}
                >
                  <CardContent className="p-4 flex flex-col items-center gap-2 relative">
                    <Icon size={20} className={isActive ? 'text-primary' : 'text-foreground'} />
                    <span className={`text-sm ${isActive ? 'text-primary font-medium' : 'text-foreground'}`}>
                      {t.label}
                    </span>
                    {isActive && (
                      <div className="absolute top-2 right-2 w-4 h-4 rounded-full bg-primary flex items-center justify-center">
                        <Check size={10} className="text-primary-foreground" />
                      </div>
                    )}
                  </CardContent>
                </Card>
              )
            })}
          </div>
        </section>

        {/* 模型服务 */}
        <section>
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-medium text-muted-foreground uppercase tracking-wider">
              模型提供商
            </h2>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setImportExportOpen(true)}
                className="flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-medium border border-border hover:bg-accent transition-colors"
                data-testid="import-export-btn"
              >
                <FileJson className="w-3 h-3" />
                导入 / 导出
              </button>
              <button
                onClick={handleOpenCustomDialog}
                className="flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-medium border border-border hover:bg-accent transition-colors"
                data-testid="add-custom-provider-btn"
              >
                <Plus className="w-3 h-3" />
                自定义
              </button>
            </div>
          </div>

          {/* 已添加的 Provider 列表（按分组折叠） */}
          <div className="mb-4">
            <ProviderGroupList
              providers={providers}
              activeProviderId={activeProviderId}
              onEdit={handleEditProvider}
              onDelete={handleDeleteClick}
            />
          </div>

          {/* 从模板添加 */}
          <div className="border-t border-border pt-4">
            <p className="text-xs text-muted-foreground mb-3">从预置模板快速添加</p>
            <ProviderTemplateList
              onSelectTemplate={handleSelectTemplate}
              isAddedCheck={(templateId) => hasProvider(templateId)}
            />
          </div>
        </section>

        {/* 合规提示条设置 */}
        <section>
          <h2 className="text-sm font-medium text-muted-foreground mb-3 uppercase tracking-wider">
            合规提示条
          </h2>
          <div className="space-y-2">
            {complianceModes.map((m) => {
              const Icon = m.icon
              const isActive = complianceBarMode === m.id
              return (
                <Card
                  key={m.id}
                  className={`cursor-pointer transition-all ${
                    isActive
                      ? 'border-primary bg-primary/5'
                      : 'border-border hover:border-primary/30 hover:bg-accent'
                  }`}
                  onClick={() => setComplianceBarMode(m.id)}
                >
                  <CardContent className="p-4 flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <Icon size={18} className={isActive ? 'text-primary' : 'text-muted-foreground'} />
                      <div>
                        <div className={`text-sm font-medium ${isActive ? 'text-primary' : 'text-foreground'}`}>
                          {m.label}
                        </div>
                        <div className="text-xs text-muted-foreground">{m.desc}</div>
                      </div>
                    </div>
                    {isActive && (
                      <div className="w-4 h-4 rounded-full bg-primary flex items-center justify-center">
                        <div className="w-1.5 h-1.5 rounded-full bg-primary-foreground" />
                      </div>
                    )}
                  </CardContent>
                </Card>
              )
            })}
          </div>
        </section>

        {/* 置信度条设置 */}
        <section>
          <h2 className="text-sm font-medium text-muted-foreground mb-3 uppercase tracking-wider">
            置信度条
          </h2>
          <div
            className={`flex items-center justify-between p-4 rounded-lg border cursor-pointer transition-all ${
              showConfidenceBar
                ? 'border-primary bg-primary/5'
                : 'border-border hover:border-primary/30 hover:bg-accent'
            }`}
            onClick={() => {
              const next = !showConfidenceBar
              setShowConfidenceBar(next)
              if (next && confidenceBarMode === 'hidden') {
                setConfidenceBarMode('compact')
              }
            }}
          >
            <div className="flex items-center gap-3">
              <Monitor size={18} className={showConfidenceBar ? 'text-primary' : 'text-muted-foreground'} />
              <div>
                <div className={`text-sm font-medium ${showConfidenceBar ? 'text-primary' : 'text-foreground'}`}>
                  展示置信度
                </div>
                <div className="text-xs text-muted-foreground">在 AI 回复底部显示回答可信度评估</div>
              </div>
            </div>
            <div
              className={`w-10 h-5 rounded-full transition-colors relative ${
                showConfidenceBar ? 'bg-primary' : 'bg-muted'
              }`}
            >
              <div
                className={`w-4 h-4 rounded-full bg-white absolute top-0.5 transition-transform ${
                  showConfidenceBar ? 'translate-x-5' : 'translate-x-0.5'
                }`}
              />
            </div>
          </div>
        </section>

        {/* 隐私设置 */}
        <section>
          <h2 className="text-sm font-medium text-muted-foreground mb-3 uppercase tracking-wider">
            隐私
          </h2>
          <div className="space-y-4">
            {/* 脱敏级别 */}
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground">脱敏级别</label>
              <div className="space-y-2">
                {desensitizationLevels.map((l) => {
                  const Icon = l.icon
                  const isActive = desensitizationLevel === l.id
                  return (
                    <Card
                      key={l.id}
                      className={`cursor-pointer transition-all ${
                        isActive
                          ? 'border-primary bg-primary/5'
                          : 'border-border hover:border-primary/30 hover:bg-accent'
                      }`}
                      onClick={() => setDesensitizationLevel(l.id)}
                    >
                      <CardContent className="p-4 flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <Icon size={18} className={isActive ? 'text-primary' : 'text-muted-foreground'} />
                          <div>
                            <div className={`text-sm font-medium ${isActive ? 'text-primary' : 'text-foreground'}`}>
                              {l.label}
                            </div>
                            <div className="text-xs text-muted-foreground">{l.desc}</div>
                          </div>
                        </div>
                        {isActive && (
                          <div className="w-4 h-4 rounded-full bg-primary flex items-center justify-center">
                            <div className="w-1.5 h-1.5 rounded-full bg-primary-foreground" />
                          </div>
                        )}
                      </CardContent>
                    </Card>
                  )
                })}
              </div>
            </div>

            {/* 数据留存期限 */}
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground">数据留存期限</label>
              <div className="flex flex-wrap gap-2">
                {retentionOptions.map((opt) => (
                  <button
                    key={opt.value}
                    onClick={() => setDataRetentionDays(opt.value)}
                    className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                      dataRetentionDays === opt.value
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted text-muted-foreground hover:bg-accent'
                    }`}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* 记忆召回模式 */}
        <section>
          <h2 className="text-sm font-medium text-muted-foreground mb-3 uppercase tracking-wider">
            记忆召回模式
          </h2>
          <div className="p-4 rounded-lg border border-border bg-card space-y-4">
            {embeddingStatus?.available ? (
              /* 语义搜索已启用 */
              <div className="flex items-start gap-3">
                <div className="mt-0.5 w-8 h-8 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center shrink-0">
                  <Sparkles size={16} className="text-green-600 dark:text-green-400" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium text-foreground">语义搜索（智能模式）</div>
                  <div className="text-xs text-muted-foreground mt-1">
                    AI 通过语义理解来召回你的历史记忆。"意思相近"的内容也会被关联，例如问"我多重"也能找到"体重是110公斤"。
                  </div>
                  <div className="text-xs text-muted-foreground mt-2">
                    模型路径：{embeddingStatus.model_path}
                  </div>
                </div>
              </div>
            ) : (
              /* 关键词匹配基础模式 */
              <>
                <div className="flex items-start gap-3">
                  <div className="mt-0.5 w-8 h-8 rounded-full bg-amber-100 dark:bg-amber-900/30 flex items-center justify-center shrink-0">
                    <Brain size={16} className="text-amber-600 dark:text-amber-400" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium text-foreground">关键词匹配（基础模式）</div>
                    <div className="text-xs text-muted-foreground mt-1">
                      当前 AI 通过关键词字面匹配来召回你的历史记忆。例如你说"我体重多少"能找到"体重是110公斤"，但说"我多重"可能找不到。
                    </div>
                  </div>
                </div>

                <div className="pt-3 border-t border-border">
                  <div className="flex items-start gap-3">
                    <div className="mt-0.5 w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                      <Sparkles size={16} className="text-primary" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium text-foreground">可选升级：语义搜索（智能模式）</div>
                      <div className="text-xs text-muted-foreground mt-1">
                        安装 Embedding 模型后，AI 能理解"意思相近"。"多重"和"体重"也会被关联，召回更自然准确。
                      </div>
                    </div>
                  </div>

                  <div className="mt-3 flex items-center gap-2">
                    <button
                      onClick={() => {
                        openEmbeddingModelDir()
                          .then(() => showToast('已打开模型目录'))
                          .catch((err) => {
                            logger.error('Failed to open model dir:', err)
                            showToast('无法自动打开目录，请手动前往')
                          })
                      }}
                      className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border border-border hover:bg-accent transition-colors"
                    >
                      <FolderOpen size={14} />
                      打开模型目录
                    </button>
                    {embeddingStatus?.download_url && (
                      <button
                        onClick={() => {
                          if (embeddingStatus.download_url) {
                            window.open(embeddingStatus.download_url, '_blank')
                          }
                        }}
                        className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border border-border hover:bg-accent transition-colors"
                      >
                        <ExternalLink size={14} />
                        下载模型
                      </button>
                    )}
                  </div>

                  {modelDirPath && (
                    <div className="mt-2 text-xs text-muted-foreground">
                      模型目录：
                      <span className="font-mono bg-muted px-1 rounded">{modelDirPath}</span>
                    </div>
                  )}

                  <div className="mt-3 p-3 rounded-md bg-muted/50 space-y-2">
                    <div className="flex items-center gap-2">
                      <AlertCircle size={14} className="text-muted-foreground shrink-0" />
                      <span className="text-xs font-medium text-foreground">如何升级到语义搜索</span>
                    </div>
                    <ol className="text-xs text-muted-foreground space-y-1 list-decimal list-inside">
                      <li>
                        点击上方"下载模型"按钮获取{' '}
                        <span className="font-mono text-xs bg-muted px-1 rounded">all-MiniLM-L6-v2</span>{' '}
                        ONNX 模型（约 50MB）
                      </li>
                      <li>
                        将 <span className="font-mono text-xs bg-muted px-1 rounded">model.onnx</span> 和{' '}
                        <span className="font-mono text-xs bg-muted px-1 rounded">tokenizer.json</span>{' '}
                        放入模型目录
                      </li>
                      <li>重启应用即可生效（不安装也能继续用关键词匹配）</li>
                    </ol>
                  </div>
                </div>
              </>
            )}
          </div>
        </section>

        {/* 安装向导 */}
        <section>
          <h2 className="text-sm font-medium text-muted-foreground mb-3 uppercase tracking-wider">
            安装向导
          </h2>
          <div className="p-4 rounded-lg border border-border bg-card space-y-3">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-foreground">
                  {onboardingCompleted ? '向导已完成' : '向导未完成'}
                </div>
                <div className="text-xs text-muted-foreground">
                  {onboardingCompleted
                    ? '您已完成首次安装引导配置'
                    : '首次安装引导尚未完成，部分功能可能未配置'}
                </div>
              </div>
              <button
                onClick={resetOnboarding}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border border-border hover:bg-accent transition-colors"
              >
                <RotateCcw size={14} />
                重新运行
              </button>
            </div>
          </div>
        </section>

        {/* 本地埋点 */}
        <section>
          <h2 className="text-sm font-medium text-muted-foreground mb-3 uppercase tracking-wider">
            本地埋点
          </h2>
          <div className="p-4 rounded-lg border border-border bg-card space-y-3">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-foreground">向导完成统计</div>
                <div className="text-xs text-muted-foreground">
                  {analytics.length > 0
                    ? `已记录 ${analytics.length} 个步骤的数据（纯本地存储，不上传）`
                    : '暂无记录'}
                </div>
              </div>
              <button
                onClick={() => setShowAnalytics(!showAnalytics)}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border border-border hover:bg-accent transition-colors"
              >
                <Eye size={14} />
                {showAnalytics ? '隐藏' : '查看'}
              </button>
            </div>
            {showAnalytics && analytics.length > 0 && (
              <div className="space-y-2 pt-2 border-t border-border">
                {analytics.map((a) => (
                  <div key={a.step} className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground">步骤 {a.step}</span>
                    <div className="flex items-center gap-3">
                      <span className={a.completedAt ? 'text-green-600' : 'text-amber-600'}>
                        {a.completedAt ? '已完成' : a.skipped ? '已跳过' : '进行中'}
                      </span>
                      {a.completedAt && (
                        <span className="text-muted-foreground">
                          {((a.completedAt - a.startedAt) / 1000).toFixed(1)}s
                        </span>
                      )}
                    </div>
                  </div>
                ))}
                <button
                  onClick={clearAnalytics}
                  className="flex items-center gap-1.5 text-xs text-destructive hover:text-destructive/80 transition-colors mt-2"
                >
                  <Trash2 size={12} />
                  清除埋点数据
                </button>
              </div>
            )}
          </div>
        </section>

        {/* 更新设置 */}
        <section>
          <h2 className="text-sm font-medium text-muted-foreground mb-3 uppercase tracking-wider">
            自动更新
          </h2>
          <div className="space-y-4">
            {/* 自动检测开关 */}
            <div
              className={`flex items-center justify-between p-4 rounded-lg border cursor-pointer transition-all ${
                autoCheckUpdate
                  ? 'border-primary bg-primary/5'
                  : 'border-border hover:border-primary/30 hover:bg-accent'
              }`}
              onClick={() => setAutoCheckUpdate(!autoCheckUpdate)}
            >
              <div className="flex items-center gap-3">
                <RefreshCw size={18} className={autoCheckUpdate ? 'text-primary' : 'text-muted-foreground'} />
                <div>
                  <div className={`text-sm font-medium ${autoCheckUpdate ? 'text-primary' : 'text-foreground'}`}>
                    自动检测更新
                  </div>
                  <div className="text-xs text-muted-foreground">应用启动时自动检查 GitHub Releases 新版本</div>
                </div>
              </div>
              <div
                className={`w-10 h-5 rounded-full transition-colors relative ${
                  autoCheckUpdate ? 'bg-primary' : 'bg-muted'
                }`}
              >
                <div
                  className={`w-4 h-4 rounded-full bg-white absolute top-0.5 transition-transform ${
                    autoCheckUpdate ? 'translate-x-5' : 'translate-x-0.5'
                  }`}
                />
              </div>
            </div>

            {/* 更新通道 */}
            <div className="space-y-2">
              {updateChannels.map((ch) => {
                const Icon = ch.icon
                const isActive = updateChannel === ch.id
                return (
                  <Card
                    key={ch.id}
                    className={`cursor-pointer transition-all ${
                      isActive
                        ? 'border-primary bg-primary/5'
                        : 'border-border hover:border-primary/30 hover:bg-accent'
                    }`}
                    onClick={() => setUpdateChannel(ch.id)}
                  >
                    <CardContent className="p-4 flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <Icon size={18} className={isActive ? 'text-primary' : 'text-muted-foreground'} />
                        <div>
                          <div className={`text-sm font-medium ${isActive ? 'text-primary' : 'text-foreground'}`}>
                            {ch.label}
                          </div>
                          <div className="text-xs text-muted-foreground">{ch.desc}</div>
                        </div>
                      </div>
                      {isActive && (
                        <div className="w-4 h-4 rounded-full bg-primary flex items-center justify-center">
                          <div className="w-1.5 h-1.5 rounded-full bg-primary-foreground" />
                        </div>
                      )}
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          </div>
        </section>
      </div>

      {/* Toast 提示 */}
      {toastMsg && (
        <div className="fixed bottom-8 left-1/2 -translate-x-1/2 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium shadow-lg z-50 animate-in fade-in slide-in-from-bottom-2 duration-200">
          {toastMsg}
        </div>
      )}

      {/* 模型服务配置弹窗（CherryStudio 风格） */}
      <ModelServiceDialog
        mode={serviceDialogMode}
        template={serviceDialogTemplate}
        provider={editingProvider}
        existingGroups={existingGroups.length > 0 ? existingGroups : ['默认']}
        open={serviceDialogOpen}
        onClose={() => setServiceDialogOpen(false)}
        onSave={handleSaveService}
      />

      {/* 删除确认弹窗 */}
      <DeleteConfirmDialog
        provider={deletingProvider}
        isActiveProvider={deletingProvider ? activeProviderId === deletingProvider.id : false}
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
        onConfirm={handleConfirmDelete}
      />

      {/* 导入导出弹窗 */}
      <ProviderImportExport
        open={importExportOpen}
        onClose={() => setImportExportOpen(false)}
      />
    </div>
  )
}
