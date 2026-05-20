import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useAuth } from './useAuth'
import { setMockHandlers, EventsEmit, resetWailsMock } from '@/test/mocks/wails'

describe('useAuth', () => {
  beforeEach(() => {
    resetWailsMock()
  })

  it('初始状态：未检测', () => {
    const { result } = renderHook(() => useAuth())
    expect(result.current.detecting).toBe(false)
    expect(result.current.detected).toBe(false)
    expect(result.current.result).toBeNull()
    expect(result.current.error).toBeNull()
  })

  it('detect() 调用后进入检测中状态', async () => {
    setMockHandlers({
      DetectAuthMethods: vi.fn(() =>
        Promise.resolve({
          results: [
            { method: 'cli_token', available: true, connected: true, tier: 1, detail: '已连接' },
          ],
          recommended: 'cli_token',
          all_unavailable: false,
        })
      ),
    })

    const { result } = renderHook(() => useAuth())

    act(() => {
      result.current.detect()
    })

    expect(result.current.detecting).toBe(true)

    await waitFor(() => {
      expect(result.current.detected).toBe(true)
    })

    expect(result.current.result?.recommended).toBe('cli_token')
    expect(result.current.result?.results[0].available).toBe(true)
    expect(result.current.expandedPanel).toBe('cli_token')
  })

  it('detect() 失败时设置 error', async () => {
    setMockHandlers({
      DetectAuthMethods: vi.fn(() => Promise.reject(new Error('网络超时'))),
    })

    const { result } = renderHook(() => useAuth())

    await act(async () => {
      await result.current.detect()
    })

    expect(result.current.detecting).toBe(false)
    expect(result.current.error).toBe('网络超时')
  })

  it('selectMethod() 切换展开面板', () => {
    const { result } = renderHook(() => useAuth())

    act(() => {
      result.current.selectMethod('api_key')
    })
    expect(result.current.expandedPanel).toBe('api_key')

    act(() => {
      result.current.selectMethod('api_key')
    })
    expect(result.current.expandedPanel).toBeNull()
  })

  it('监听 ollama:pull_progress 事件更新进度', () => {
    const { result } = renderHook(() => useAuth())

    act(() => {
      EventsEmit('ollama:pull_progress', { model: 'smollm2:135m', progress: 'pulling manifest' })
    })

    expect(result.current.ollamaPulling).toBe(true)
    expect(result.current.ollamaPullProgress).toBe('pulling manifest')
  })

  it('监听 ollama:pull_done 事件结束下载', () => {
    const { result } = renderHook(() => useAuth())

    act(() => {
      EventsEmit('ollama:pull_progress', { model: 'smollm2:135m', progress: 'downloading' })
    })
    expect(result.current.ollamaPulling).toBe(true)

    act(() => {
      EventsEmit('ollama:pull_done', { model: 'smollm2:135m' })
    })
    expect(result.current.ollamaPulling).toBe(false)
  })

  it('reset() 重置所有状态', async () => {
    setMockHandlers({
      DetectAuthMethods: vi.fn(() =>
        Promise.resolve({
          results: [],
          recommended: '',
          all_unavailable: true,
        })
      ),
    })

    const { result } = renderHook(() => useAuth())

    await act(async () => {
      await result.current.detect()
    })

    expect(result.current.detected).toBe(true)

    act(() => {
      result.current.reset()
    })

    expect(result.current.detected).toBe(false)
    expect(result.current.result).toBeNull()
    expect(result.current.expandedPanel).toBeNull()
  })
})
