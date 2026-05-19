import { useState } from 'react'
import { Eye, EyeOff, Sparkles } from 'lucide-react'

interface ModelConfigStepProps {
  initialModel: string
  onComplete: (model: string, apiKey: string) => void
  onBack: () => void
  onSkipAPIKey: () => void
}

const models = [
  { id: 'kimi-lite', name: 'Kimi Lite', provider: 'kimi', desc: 'Moonshot 出品，中文场景表现出色' },
  { id: 'gpt-4o-mini', name: 'GPT-4o Mini', provider: 'openai', desc: 'OpenAI 轻量模型，速度快、成本低' },
  { id: 'qwen-turbo', name: '通义千问 Turbo', provider: 'qwen', desc: '阿里云出品，国内 API 稳定' },
  { id: 'llama3.1-8b', name: 'Llama 3.1 8B', provider: 'ollama', desc: '本地运行，无需联网，需先安装 Ollama' },
]

/**
 * 向导第3步：模型配置引导。
 * 选择默认模型并输入 API Key（可选）。
 */
export function ModelConfigStep({
  initialModel,
  onComplete,
  onBack,
  onSkipAPIKey,
}: ModelConfigStepProps) {
  const [selected, setSelected] = useState(initialModel)
  const [apiKey, setApiKey] = useState('')
  const [showKey, setShowKey] = useState(false)

  const selectedModel = models.find((m) => m.id === selected)
  const needsAPIKey = selectedModel?.provider !== 'ollama'

  return (
    <div className="flex flex-col space-y-5">
      <div className="text-center">
        <h2 className="text-xl font-bold text-foreground mb-1">模型配置</h2>
        <p className="text-sm text-muted-foreground">选择默认使用的 AI 模型并配置 API Key</p>
      </div>

      {/* 模型选择 */}
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">默认模型</label>
        <div className="space-y-2">
          {models.map((m) => (
            <button
              key={m.id}
              onClick={() => setSelected(m.id)}
              className={`w-full flex items-start gap-3 p-3 rounded-lg border text-left transition-all ${
                selected === m.id
                  ? 'border-primary bg-primary/5'
                  : 'border-border bg-card hover:bg-accent'
              }`}
            >
              <div
                className={`w-8 h-8 rounded-lg flex items-center justify-center shrink-0 mt-0.5 ${
                  selected === m.id ? 'bg-primary/10' : 'bg-muted'
                }`}
              >
                <Sparkles
                  className={`w-4 h-4 ${selected === m.id ? 'text-primary' : 'text-muted-foreground'}`}
                />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between">
                  <span className={`text-sm font-medium ${selected === m.id ? 'text-primary' : 'text-foreground'}`}>
                    {m.name}
                  </span>
                  {selected === m.id && (
                    <svg className="w-4 h-4 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                  )}
                </div>
                <p className="text-xs text-muted-foreground mt-0.5">{m.desc}</p>
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* API Key 输入 */}
      {needsAPIKey && (
        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">API Key</label>
          <div className="relative">
            <input
              type={showKey ? 'text' : 'password'}
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="请输入您的 API Key"
              className="w-full px-3 py-2 pr-10 rounded-lg border border-border bg-background text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/30"
            />
            <button
              onClick={() => setShowKey(!showKey)}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              type="button"
            >
              {showKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            </button>
          </div>
          <p className="text-xs text-muted-foreground">
            API Key 将通过系统密钥环保管，不会以明文形式存储在本地文件中。
          </p>
        </div>
      )}

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
            onClick={() => onComplete(selected, apiKey)}
            className="flex-1 py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
          >
            完成
          </button>
        </div>
        {needsAPIKey && (
          <button
            onClick={onSkipAPIKey}
            className="w-full py-2 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            稍后再说，不输入 API Key
          </button>
        )}
      </div>
    </div>
  )
}
