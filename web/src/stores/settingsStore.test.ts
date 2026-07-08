import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('settingsStore', () => {
  const setDesensitizationLevelMock = vi.fn().mockResolvedValue(undefined)
  const setDataRetentionDaysMock = vi.fn().mockResolvedValue(undefined)
  const getDesensitizationLevelMock = vi.fn().mockResolvedValue('standard')

  beforeEach(() => {
    localStorage.clear()
    setDesensitizationLevelMock.mockReset().mockResolvedValue(undefined)
    setDataRetentionDaysMock.mockReset().mockResolvedValue(undefined)
    getDesensitizationLevelMock.mockReset().mockResolvedValue('standard')

    // Wails 运行时通过 window.go.main.WailsApp 暴露 Go 方法
    Object.defineProperty(window, 'go', {
      value: {
        main: {
          WailsApp: {
            SetDesensitizationLevel: setDesensitizationLevelMock,
            SetDataRetentionDays: setDataRetentionDaysMock,
            GetDesensitizationLevel: getDesensitizationLevelMock,
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

  it('should roll back local state when backend save fails', async () => {
    setDesensitizationLevelMock.mockRejectedValueOnce(new Error('backend down'))
    const { useSettingsStore } = await import('./settingsStore')
    const store = useSettingsStore.getState()

    // 基线为默认 standard；尝试切换到 strict 但后端失败。
    expect(useSettingsStore.getState().desensitizationLevel).toBe('standard')
    await expect(store.setDesensitizationLevel('strict')).rejects.toThrow('backend down')

    // 失败后应回滚到 standard，保持前后端一致。
    expect(useSettingsStore.getState().desensitizationLevel).toBe('standard')
  })

  it('should read back desensitization level from backend on startup', async () => {
    getDesensitizationLevelMock.mockResolvedValueOnce('strict')
    const { useSettingsStore } = await import('./settingsStore')
    const store = useSettingsStore.getState()

    // 本地默认 standard，后端权威值为 strict，回读后应校正为 strict。
    expect(useSettingsStore.getState().desensitizationLevel).toBe('standard')
    await store.syncDesensitizationLevelFromBackend()

    expect(getDesensitizationLevelMock).toHaveBeenCalledTimes(1)
    expect(useSettingsStore.getState().desensitizationLevel).toBe('strict')
  })
})
