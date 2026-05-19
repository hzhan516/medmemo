import { useSettingsStore } from '@/stores/settingsStore'
import { useTheme } from '@/hooks/useTheme'
import { Card, CardContent } from '@/components/ui/card'
import { Monitor, Moon, Sun, Check } from 'lucide-react'

/**
 * 设置页面：支持主题切换与模型选择。
 * 使用 shadcn/ui 组件验证 light/dark 主题兼容性。
 */
export function SettingsPage() {
  const { theme, setTheme } = useTheme()
  const { selectedModel, setSelectedModel } = useSettingsStore()

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
      </div>
    </div>
  )
}
