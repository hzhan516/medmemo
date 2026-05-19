import { useEffect, useCallback } from 'react'
import { useOnboarding } from '@/hooks/useOnboarding'
import { useSettingsStore } from '@/stores/settingsStore'
import { StepIndicator } from './StepIndicator'
import { WelcomeStep } from './WelcomeStep'
import { PrivacyStep } from './PrivacyStep'
import { ModelConfigStep } from './ModelConfigStep'

interface OnboardingWizardProps {
  onClose: () => void
}

/**
 * 安装向导全屏覆盖层容器。
 * 3 步引导：欢迎 → 隐私设置 → 模型配置。
 * 支持 Escape 键退出、步骤切换动画。
 */
export function OnboardingWizard({ onClose }: OnboardingWizardProps) {
  const onboarding = useOnboarding()
  const settings = useSettingsStore()

  // Escape 键退出（保存当前进度）
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  const handleWelcomeNext = useCallback(() => {
    onboarding.goToStep(2)
  }, [onboarding])

  const handlePrivacyNext = useCallback(
    async (level: string, retentionDays: number) => {
      await onboarding.savePrivacySettings(level, retentionDays)
      onboarding.goToStep(3)
    },
    [onboarding]
  )

  const handlePrivacyBack = useCallback(() => {
    onboarding.goToStep(1)
  }, [onboarding])

  const handleModelComplete = useCallback(
    async (model: string, apiKey: string) => {
      await onboarding.saveModelConfig(model, apiKey)
      onboarding.complete()
      onClose()
    },
    [onboarding, onClose]
  )

  const handleModelBack = useCallback(() => {
    onboarding.goToStep(2)
  }, [onboarding])

  const handleSkipAPIKey = useCallback(() => {
    // 仅保存模型选择，不保存 API Key
    onboarding.saveModelConfig(settings.selectedModel, '')
    onboarding.complete()
    onClose()
  }, [onboarding, settings.selectedModel, onClose])

  const handleSkipWizard = useCallback(() => {
    onboarding.skip()
    onClose()
  }, [onboarding, onClose])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div className="w-full max-w-md mx-4 bg-card rounded-2xl shadow-2xl border border-border p-6 animate-in fade-in zoom-in-95 duration-200">
        {/* 顶部：步骤指示器 + 跳过按钮 */}
        <div className="flex items-center justify-between mb-2">
          <StepIndicator currentStep={onboarding.currentStep} />
          <button
            onClick={handleSkipWizard}
            className="text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            跳过
          </button>
        </div>

        {/* 步骤内容 */}
        <div className="mt-2">
          {onboarding.currentStep === 1 && <WelcomeStep onNext={handleWelcomeNext} />}
          {onboarding.currentStep === 2 && (
            <PrivacyStep
              initialLevel={settings.desensitizationLevel}
              initialRetention={settings.dataRetentionDays}
              onNext={handlePrivacyNext}
              onBack={handlePrivacyBack}
            />
          )}
          {onboarding.currentStep === 3 && (
            <ModelConfigStep
              initialModel={settings.selectedModel}
              onComplete={handleModelComplete}
              onBack={handleModelBack}
              onSkipAPIKey={handleSkipAPIKey}
            />
          )}
        </div>
      </div>
    </div>
  )
}
