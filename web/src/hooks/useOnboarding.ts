import { useCallback } from 'react'
import { useOnboardingStore } from '@/stores/onboardingStore'
import { useSettingsStore } from '@/stores/settingsStore'
import * as WailsApp from '@wails/go/main/WailsApp'

/**
 * 封装安装向导的完整业务逻辑：状态管理 + 后端调用 + 埋点。
 */
export function useOnboarding() {
  const store = useOnboardingStore()
  const settings = useSettingsStore()

  const launch = useCallback(() => {
    store.start()
  }, [store])

  const goToStep = useCallback(
    (step: number) => {
      store.goToStep(step)
    },
    [store]
  )

  const savePrivacySettings = useCallback(
    async (level: string, retentionDays: number) => {
      settings.setDesensitizationLevel(level as 'standard' | 'strict' | 'off')
      settings.setDataRetentionDays(retentionDays)
    },
    [settings]
  )

  const saveModelConfig = useCallback(
    async (model: string, apiKey: string) => {
      settings.setSelectedModel(model)
      if (apiKey.trim()) {
        // 解析 provider 前缀：模型 ID 格式为 "provider-model"
        const provider = model.split('-')[0] || 'kimi'
        await WailsApp.SaveAPIKey(provider, apiKey.trim())
      }
    },
    [settings]
  )

  const checkHasAPIKey = useCallback(
    async (model: string): Promise<boolean> => {
      const provider = model.split('-')[0] || 'kimi'
      return await WailsApp.HasAPIKey(provider)
    },
    []
  )

  const complete = useCallback(() => {
    store.complete()
  }, [store])

  const skip = useCallback(() => {
    store.skip()
  }, [store])

  const reset = useCallback(() => {
    store.reset()
  }, [store])

  return {
    completed: store.completed,
    skipped: store.skipped,
    currentStep: store.currentStep,
    analytics: store.analytics,
    launch,
    goToStep,
    savePrivacySettings,
    saveModelConfig,
    checkHasAPIKey,
    complete,
    skip,
    reset,
  }
}
