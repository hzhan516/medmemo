import { useEffect } from 'react'
import { useSettingsStore } from '@/stores/settingsStore'
import {
  WindowSetDarkTheme,
  WindowSetLightTheme,
  WindowSetSystemDefaultTheme,
} from '@wails/runtime'

/**
 * 主题管理 Hook，监听系统主题偏好并支持手动切换。
 * 同步 Wails 运行时窗口主题，确保原生标题栏等系统组件跟随变化。
 */
export function useTheme() {
  const { theme, setTheme } = useSettingsStore()

  useEffect(() => {
    const root = document.documentElement

    const applyTheme = (t: string) => {
      if (t === 'dark') {
        root.classList.add('dark')
        WindowSetDarkTheme()
      } else if (t === 'light') {
        root.classList.remove('dark')
        WindowSetLightTheme()
      } else {
        // system
        WindowSetSystemDefaultTheme()
        const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
        if (prefersDark) {
          root.classList.add('dark')
        } else {
          root.classList.remove('dark')
        }
      }
    }

    applyTheme(theme)

    if (theme === 'system') {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
      const handler = (e: MediaQueryListEvent) => {
        if (e.matches) {
          root.classList.add('dark')
        } else {
          root.classList.remove('dark')
        }
      }
      mediaQuery.addEventListener('change', handler)
      return () => mediaQuery.removeEventListener('change', handler)
    }
  }, [theme])

  return { theme, setTheme }
}
