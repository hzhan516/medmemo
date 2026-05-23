import { useState } from 'react'
import { Shield, ShieldCheck, ShieldOff } from 'lucide-react'

interface PrivacyStepProps {
  initialLevel: string
  initialRetention: number
  onNext: (level: string, retentionDays: number) => void
  onBack: () => void
}

type DesensitizationLevel = 'standard' | 'strict' | 'off'

const levels: { value: DesensitizationLevel; label: string; desc: string; icon: typeof Shield }[] = [
  {
    value: 'standard',
    label: '标准',
    desc: '启用规则脱敏 + NER 模型识别，平衡隐私与回答质量',
    icon: Shield,
  },
  {
    value: 'strict',
    label: '严格',
    desc: '三重脱敏兜底，最大程度保护敏感信息，可能略微影响回答完整性',
    icon: ShieldCheck,
  },
  {
    value: 'off',
    label: '关闭',
    desc: '不进行脱敏处理，数据以明文传输。请确认您了解相关风险',
    icon: ShieldOff,
  },
]

const retentionOptions = [
  { value: 7, label: '7 天' },
  { value: 30, label: '30 天' },
  { value: 90, label: '90 天' },
  { value: 365, label: '1 年' },
  { value: 0, label: '永久保留' },
]

/**
 * 向导第2步：隐私设置。
 * 脱敏级别与数据留存期限选择。
 */
export function PrivacyStep({ initialLevel, initialRetention, onNext, onBack }: PrivacyStepProps) {
  const [level, setLevel] = useState<DesensitizationLevel>(initialLevel as DesensitizationLevel)
  const [retention, setRetention] = useState<number>(initialRetention)

  return (
    <div className="flex flex-col space-y-5">
      <div className="text-center">
        <h2 className="text-xl font-bold text-foreground mb-1">隐私设置</h2>
        <p className="text-sm text-muted-foreground">配置数据保护策略，您可以随时在设置中更改</p>
      </div>

      {/* 脱敏级别 */}
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">脱敏级别</label>
        <div className="space-y-2">
          {levels.map((l) => (
            <button
              key={l.value}
              onClick={() => setLevel(l.value)}
              className={`w-full flex items-start gap-3 p-3 rounded-lg border text-left transition-all ${
                level === l.value
                  ? 'border-primary bg-primary/5'
                  : 'border-border bg-card hover:bg-accent'
              }`}
            >
              <l.icon
                className={`w-5 h-5 shrink-0 mt-0.5 ${
                  level === l.value ? 'text-primary' : 'text-muted-foreground'
                }`}
              />
              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between">
                  <span className={`text-sm font-medium ${level === l.value ? 'text-primary' : 'text-foreground'}`}>
                    {l.label}
                  </span>
                  {level === l.value && (
                    <svg className="w-4 h-4 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                  )}
                </div>
                <p className="text-xs text-muted-foreground mt-0.5 leading-relaxed">{l.desc}</p>
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* 数据留存期限 */}
      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">数据留存期限</label>
        <div className="flex flex-wrap gap-2">
          {retentionOptions.map((opt) => (
            <button
              key={opt.value}
              onClick={() => setRetention(opt.value)}
              className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                retention === opt.value
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-muted-foreground hover:bg-accent'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
        <p className="text-xs text-muted-foreground">
          超过期限的历史对话将自动清理。设置为"永久保留"则不会自动删除。
        </p>
      </div>

      {/* 导航按钮 */}
      <div className="flex gap-3 pt-2">
        <button
          onClick={onBack}
          className="flex-1 py-2.5 px-4 rounded-lg border border-border text-sm font-medium text-foreground hover:bg-accent transition-colors"
        >
          上一步
        </button>
        <button
          onClick={() => onNext(level, retention)}
          className="flex-1 py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
        >
          下一步
        </button>
      </div>
    </div>
  )
}
