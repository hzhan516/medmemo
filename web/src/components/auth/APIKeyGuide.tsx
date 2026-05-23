import { useState, useCallback } from 'react'
import { ChevronDown, ChevronUp, ExternalLink } from 'lucide-react'

interface Step {
  text: string
  highlight?: string
}

interface ProviderGuide {
  id: string
  name: string
  docsUrl: string
  steps: Step[]
}

const providerGuides: ProviderGuide[] = [
  {
    id: 'openai',
    name: 'OpenAI',
    docsUrl: 'https://platform.openai.com/api-keys',
    steps: [
      { text: '访问 platform.openai.com 并登录你的账号' },
      { text: '点击左侧菜单 API Keys', highlight: 'API Keys' },
      { text: '点击 Create new secret key 按钮', highlight: 'Create new secret key' },
      { text: '输入密钥名称，选择权限范围，点击 Create secret key', highlight: 'Create secret key' },
      { text: '立即复制生成的 Key（关闭后将无法再次查看）', highlight: '立即复制' },
    ],
  },
  {
    id: 'gemini',
    name: 'Gemini (Google AI Studio)',
    docsUrl: 'https://aistudio.google.com/app/apikey',
    steps: [
      { text: '访问 aistudio.google.com 并使用 Google 账号登录' },
      { text: '点击左侧菜单 Get API key', highlight: 'Get API key' },
      { text: '点击 Create API key in new project', highlight: 'Create API key' },
      { text: '复制生成的 API Key' },
    ],
  },
  {
    id: 'kimi',
    name: 'Kimi (Moonshot)',
    docsUrl: 'https://platform.moonshot.cn/console/api-keys',
    steps: [
      { text: '访问 platform.moonshot.cn 并登录账号' },
      { text: '进入左侧 用户中心 → API Key 管理', highlight: 'API Key 管理' },
      { text: '点击 新建 按钮', highlight: '新建' },
      { text: '输入名称和可选的限额设置，点击 确定', highlight: '确定' },
      { text: '复制生成的 API Key' },
    ],
  },
  {
    id: 'deepseek',
    name: 'DeepSeek',
    docsUrl: 'https://platform.deepseek.com/api_keys',
    steps: [
      { text: '访问 platform.deepseek.com 并登录账号' },
      { text: '点击左侧 API Keys', highlight: 'API Keys' },
      { text: '点击 Create API Key 按钮', highlight: 'Create API Key' },
      { text: '输入名称后点击 Create', highlight: 'Create' },
      { text: '复制生成的 Key' },
    ],
  },
  {
    id: 'claude',
    name: 'Claude (Anthropic)',
    docsUrl: 'https://console.anthropic.com/settings/keys',
    steps: [
      { text: '访问 console.anthropic.com 并登录账号' },
      { text: '点击左侧 Settings → API Keys', highlight: 'API Keys' },
      { text: '点击 Create Key 按钮', highlight: 'Create Key' },
      { text: '输入名称后点击 Create', highlight: 'Create' },
      { text: '复制生成的 Key' },
    ],
  },
]

interface APIKeyGuideProps {
  providerId: string
  onOpenURL: (url: string) => void
}

/**
 * API Key 获取引导组件。
 * 展示指定厂商的分步图文指引，支持折叠展开和一键直达厂商页面。
 */
export function APIKeyGuide({ providerId, onOpenURL }: APIKeyGuideProps) {
  const [expanded, setExpanded] = useState(false)

  const guide = providerGuides.find((g) => g.id === providerId)

  const handleOpenURL = useCallback(() => {
    if (guide) {
      onOpenURL(guide.docsUrl)
    }
  }, [guide, onOpenURL])

  if (!guide) {
    return null
  }

  return (
    <div className="rounded-lg border border-border overflow-hidden">
      {/* 折叠头部 */}
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center justify-between px-3 py-2.5 text-sm hover:bg-muted/50 transition-colors"
        data-testid="apikey-guide-toggle"
      >
        <span className="font-medium text-foreground">如何获取 {guide.name} 的 API Key？</span>
        {expanded ? (
          <ChevronUp className="w-4 h-4 text-muted-foreground shrink-0" />
        ) : (
          <ChevronDown className="w-4 h-4 text-muted-foreground shrink-0" />
        )}
      </button>

      {/* 展开内容 */}
      {expanded && (
        <div className="px-3 pb-3 border-t border-border bg-muted/20" data-testid="apikey-guide-content">
          {/* 步骤列表 */}
          <ol className="mt-3 space-y-2.5">
            {guide.steps.map((step, index) => (
              <li key={index} className="flex items-start gap-2.5">
                <span className="flex items-center justify-center w-5 h-5 rounded-full bg-primary/10 text-primary text-xs font-semibold shrink-0 mt-0.5">
                  {index + 1}
                </span>
                <p className="text-sm text-foreground leading-relaxed">
                  {step.highlight ? (
                    <>
                      {step.text.split(step.highlight).map((part, i, arr) => (
                        <span key={i}>
                          {part}
                          {i < arr.length - 1 && (
                            <code className="px-1 py-0.5 rounded bg-amber-100 dark:bg-amber-900/30 text-amber-800 dark:text-amber-200 text-xs font-medium">
                              {step.highlight}
                            </code>
                          )}
                        </span>
                      ))}
                    </>
                  ) : (
                    step.text
                  )}
                </p>
              </li>
            ))}
          </ol>

          {/* 一键直达 */}
          <button
            type="button"
            onClick={handleOpenURL}
            className="mt-3 w-full flex items-center justify-center gap-1.5 py-2 px-3 rounded-md bg-primary/5 hover:bg-primary/10 text-primary text-sm font-medium transition-colors"
            data-testid="apikey-guide-open-url"
          >
            <ExternalLink className="w-3.5 h-3.5" />
            去获取 API Key
          </button>
        </div>
      )}
    </div>
  )
}
