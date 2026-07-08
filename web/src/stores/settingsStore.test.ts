import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('settingsStore', () => {
  const setDesensitizationLevelMock = vi.fn().mockResolvedValue(undefined)
  const setDataRetentionDaysMock = vi.fn().mockResolvedValue(undefined)

  beforeEach(() => {
    localStorage.clear()
    setDesensitizationLevelMock.mockClear()
    setDataRetentionDaysMock.mockClear()

    // Wails 运行时通过 window.go.main.WailsApp 暴露 Go 方法
    Object.defineProperty(window, 'go', {
      value: {
        main: {
          WailsApp: {
            SetDesensitizationLevel: setDesensitizationLevelMock,
            SetDataRetentionDays: setDataRetentionDaysMock,
          },
        },
      },
      configurable: true,
    })

    vi.resetModules()
  })

  it('should persist desensitization level locally and sync to backend', async () => {
    const { useSettingsStore } = await import('./settingsStore')
    const store = useSettingsStore.getState()

    await store.setDesensitizationLevel('off')

    expect(useSettingsStore.getState().desensitizationLevel).toBe('off')
    expect(setDesensitizationLevelMock).toHaveBeenCalledTimes(1)
    expect(setDesensitizationLevelMock).toHaveBeenCalledWith('off')
  })

  it('should sync strict desensitization level to backend', async () => {
    const { useSettingsStore } = await import('./settingsStore')
    const store = useSettingsStore.getState()

    await store.setDesensitizationLevel('strict')

    expect(useSettingsStore.getState().desensitizationLevel).toBe('strict')
    expect(setDesensitizationLevelMock).toHaveBeenCalledTimes(1)
    expect(setDesensitizationLevelMock).toHaveBeenCalledWith('strict')
  })
})
