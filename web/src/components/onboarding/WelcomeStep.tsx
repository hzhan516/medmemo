import { Shield, MessageCircle, Users } from 'lucide-react'

interface WelcomeStepProps {
  onNext: () => void
}

/**
 * 向导第1步：欢迎与功能介绍。
 */
export function WelcomeStep({ onNext }: WelcomeStepProps) {
  const features = [
    {
      icon: Shield,
      title: '本地隐私优先',
      desc: '健康数据原则上仅存储在本地设备，云端交互前经过本地脱敏处理。',
    },
    {
      icon: MessageCircle,
      title: '智能健康咨询',
      desc: '基于大语言模型的自然语言交互，帮助您了解症状、科室建议与健康科普信息。',
    },
    {
      icon: Users,
      title: '家族健康图谱',
      desc: '记录家族成员健康信息，构建个性化风险参考（即将上线）。',
    },
  ]

  return (
    <div className="flex flex-col items-center text-center space-y-6">
      <div className="w-16 h-16 rounded-2xl bg-primary/10 flex items-center justify-center">
        <span className="text-3xl font-bold text-primary">M</span>
      </div>

      <div>
        <h2 className="text-2xl font-bold text-foreground mb-2">欢迎使用 MedMemo</h2>
        <p className="text-muted-foreground text-sm max-w-xs mx-auto">
          您的开源桌面端健康咨询信息助手
        </p>
      </div>

      <div className="w-full space-y-3">
        {features.map((f) => (
          <div
            key={f.title}
            className="flex items-start gap-3 p-3 rounded-lg bg-muted/50 text-left"
          >
            <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center shrink-0 mt-0.5">
              <f.icon className="w-4 h-4 text-primary" />
            </div>
            <div>
              <h3 className="text-sm font-medium text-foreground">{f.title}</h3>
              <p className="text-xs text-muted-foreground mt-0.5 leading-relaxed">{f.desc}</p>
            </div>
          </div>
        ))}
      </div>

      <button
        onClick={onNext}
        className="w-full py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
      >
        开始设置
      </button>
    </div>
  )
}
