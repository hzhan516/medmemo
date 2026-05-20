import { useState } from 'react'
import { AuthSelector } from '@/components/auth/AuthSelector'
import { useAuth } from '@/hooks/useAuth'
import type { ProviderConfig } from '@/types/provider'

interface ModelConfigStepProps {
  onComplete: (providers: ProviderConfig[]) => void
  onBack: () => void
  onSkipAPIKey: () => void
}

/**
 * 向导第3步：认证方式智能选择。
 * 使用 AuthSelector 组件检测并推荐最佳认证方式。
 */
export function ModelConfigStep({
  onComplete,
  onBack,
  onSkipAPIKey,
}: ModelConfigStepProps) {
  const auth = useAuth()
  const [createdProviders, setCreatedProviders] = useState<ProviderConfig[]>([])

  const handleProviderCreated = (provider: ProviderConfig) => {
    setCreatedProviders((prev) => [...prev, provider])
  }

  const handleComplete = () => {
    if (createdProviders.length > 0) {
      onComplete(createdProviders)
    } else {
      // 未创建任何 Provider，视为跳过
      onSkipAPIKey()
    }
  }

  return (
    <div className="flex flex-col space-y-5">
      <div className="text-center">
        <h2 className="text-xl font-bold text-foreground mb-1">模型配置</h2>
        <p className="text-sm text-muted-foreground">选择认证方式并配置 AI 模型</p>
      </div>

      <AuthSelector
        result={auth.result}
        detecting={auth.detecting}
        error={auth.error}
        expandedPanel={auth.expandedPanel}
        ollamaPulling={auth.ollamaPulling}
        ollamaPullProgress={auth.ollamaPullProgress}
        ollamaServerStarting={auth.ollamaServerStarting}
        onDetect={auth.detect}
        onSelectMethod={auth.selectMethod}
        onProviderCreated={handleProviderCreated}
        onSkip={onSkipAPIKey}
      />

      {createdProviders.length > 0 && (
        <div className="p-3 rounded-lg bg-green-500/5 border border-green-500/20">
          <p className="text-xs text-green-700">
            已配置 {createdProviders.length} 个 Provider
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
            onClick={handleComplete}
            className="flex-1 py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
          >
            {createdProviders.length > 0 ? '完成' : '跳过'}
          </button>
        </div>
      </div>
    </div>
  )
}
