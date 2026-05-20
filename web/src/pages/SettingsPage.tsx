import { useState } from 'react'
import { useSettingsStore } from '@/stores/settingsStore'
import { useOnboardingStore } from '@/stores/onboardingStore'
import { useProviderStore } from '@/stores/providerStore'
import { useTheme } from '@/hooks/useTheme'
import { useWails } from '@/hooks/useWails'
import { Card, CardContent } from '@/components/ui/card'
import { ProviderTemplateList, ProviderAddDialog } from '@/components/provider'
import type { ProviderTemplate } from '@/types/provider'
import { Monitor, Moon, Sun, Check, Bell, BellDot, BellOff, RefreshCw, Shield, FlaskConical, ShieldCheck, ShieldOff, Eye, Trash2, RotateCcw, Cloud, Server, Trash } from 'lucide-react'

/**
 * 设置页面：支持主题切换、模型选择与合规提示条模式。
 * 使用 shadcn/ui 组件验证 light/dark 主题兼容性。
 */
export function SettingsPage() {
  const { theme, setTheme } = useTheme()
  const { selectedModel, setSelectedModel, complianceBarMode, setComplianceBarMode, autoCheckUpdate, setAutoCheckUpdate, updateChannel, setUpdateChannel, desensitizationLevel, setDesensitizationLevel, dataRetentionDays, setDataRetentionDays } = useSettingsStore()
  const onboardingCompleted = useOnboardingStore((s) => s.completed)
  const analytics = useOnboardingStore((s) => s.analytics)
  const resetOnboarding = useOnboardingStore((s) => s.reset)
  const clearAnalytics = useOnboardingStore((s) => s.clearAnalytics)
  const [showAnalytics, setShowAnalytics] = useState(false)
  const [showAddDialog, setShowAddDialog] = useState(false)
  const [selectedTemplate, setSelectedTemplate] = useState<ProviderTemplate | null>(null)

  const providers = useProviderStore((s) => s.providers)
  const addProvider = useProviderStore((s) => s.addProvider)
  const removeProvider = useProviderStore((s) => s.removeProvider)
  const hasProvider = useProviderStore((s) => s.hasProvider)
  const { saveAPIKey } = useWails()

  const handleSelectTemplate = (template: ProviderTemplate) => {
    setSelectedTemplate(template)
    setShowAddDialog(true)
  }

  const handleSaveProvider = (config: Parameters<typeof addProvider>[0]) => {
    addProvider(config)
    if (config.apiKey) {
      saveAPIKey(config.templateId, config.apiKey).catch((err) => {
        console.error('Failed to save API key:', err)
      })
    }
  }

  const models = [
    { id: 'kimi-lite', name: 'Kimi Lite', provider: 'kimi' },
    { id: 'gpt-4o-mini', name: 'GPT-4o Mini', provider: 'openai' },
    { id: 'qwen-turbo', name: '通义千问 Turbo', provider: 'qwen' },
    { id: 'llama3.1-8b', name: 'Llama 3.1 8B (本地)', provider: 'ollama' },
  ]

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

        {/* 模型设置 */}
        <section>
          <h2 className="text-sm font-medium text-muted-foreground mb-3 uppercase tracking-wider">
            默认模型
          </h2>
          <div className="space-y-2">
            {models.map((m) => {
              const isActive = selectedModel === m.id
              return (
                <Card
                  key={m.id}
                  className={`cursor-pointer transition-all ${
                    isActive
                      ? 'border-primary bg-primary/5'
                      : 'border-border hover:border-primary/30 hover:bg-accent'
                  }`}
                  onClick={() => setSelectedModel(m.id)}
                >
                  <CardContent className="p-4 flex items-center justify-between">
                    <div>
                      <div className={`text-sm font-medium ${isActive ? 'text-primary' : 'text-foreground'}`}>
                        {m.name}
                      </div>
                      <div className="text-xs text-muted-foreground capitalize">{m.provider}</div>
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

        {/* 模型提供商 */}
        <section>
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-medium text-muted-foreground uppercase tracking-wider">
              模型提供商
            </h2>
          </div>

          {/* 已添加的 Provider 列表 */}
          {providers.length > 0 && (
            <div className="space-y-2 mb-4">
              {providers.map((p) => (
                <Card key={p.id} className="border-border">
                  <CardContent className="p-3 flex items-center justify-between">
                    <div className="flex items-center gap-2.5 min-w-0">
                      <div className={`shrink-0 w-7 h-7 rounded-md flex items-center justify-center ${p.group === '本地' ? 'bg-amber-500/10 text-amber-600' : 'bg-blue-500/10 text-blue-600'}`}>
                        {p.group === '本地' ? <Server className="w-3.5 h-3.5" /> : <Cloud className="w-3.5 h-3.5" />}
                      </div>
                      <div className="min-w-0">
                        <div className="text-sm font-medium text-foreground truncate">{p.name}</div>
                        <div className="text-[11px] text-muted-foreground truncate">{p.modelId || '未选择模型'}</div>
                      </div>
                    </div>
                    <button
                      onClick={() => removeProvider(p.id)}
                      className="p-1.5 rounded-md text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors shrink-0"
                      title="删除"
                      aria-label={`删除 ${p.name}`}
                    >
                      <Trash className="w-3.5 h-3.5" />
                    </button>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}

          {/* 模板列表 */}
          <ProviderTemplateList
            onSelectTemplate={handleSelectTemplate}
            isAddedCheck={(templateId) => hasProvider(templateId)}
          />
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

      <ProviderAddDialog
        template={selectedTemplate}
        open={showAddDialog}
        onClose={() => setShowAddDialog(false)}
        onSave={handleSaveProvider}
      />
    </div>
  )
}
