import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * 单步埋点记录。
 * 纯本地存储，不上传服务器，仅用于用户自助查看向导完成统计。
 */
export interface StepAnalytics {
  step: number
  startedAt: number
  completedAt?: number
  skipped: boolean
}

interface OnboardingState {
  completed: boolean
  currentStep: number
  skipped: boolean
  analytics: StepAnalytics[]

  start: () => void
  goToStep: (step: number) => void
  complete: () => void
  skip: () => void
  reset: () => void
  clearAnalytics: () => void
}

export const useOnboardingStore = create<OnboardingState>()(
  persist(
    (set, get) => ({
      completed: false,
      currentStep: 1,
      skipped: false,
      analytics: [],

      start: () => {
        const state = get()
        if (state.analytics.length === 0) {
          set({
            analytics: [{ step: 1, startedAt: Date.now(), skipped: false }],
          })
        }
      },

      goToStep: (step) => {
        const state = get()
        const analytics = [...state.analytics]
        // 标记当前步完成
        const current = analytics.find((a) => a.step === state.currentStep)
        if (current && !current.completedAt) {
          current.completedAt = Date.now()
        }
        // 新步骤开始
        if (!analytics.find((a) => a.step === step)) {
          analytics.push({ step, startedAt: Date.now(), skipped: false })
        }
        set({ currentStep: step, analytics })
      },

      complete: () => {
        const state = get()
        const analytics = [...state.analytics]
        const current = analytics.find((a) => a.step === state.currentStep)
        if (current && !current.completedAt) {
          current.completedAt = Date.now()
        }
        set({ completed: true, skipped: false, analytics })
      },

      skip: () => {
        const state = get()
        const analytics = [...state.analytics]
        const current = analytics.find((a) => a.step === state.currentStep)
        if (current) {
          current.skipped = true
        }
        set({ skipped: true, analytics })
      },

      reset: () => {
        set({
          completed: false,
          currentStep: 1,
          skipped: false,
          analytics: [],
        })
      },

      clearAnalytics: () => {
        set({ analytics: [] })
      },
    }),
    {
      name: 'medmemo-onboarding',
    }
  )
)
