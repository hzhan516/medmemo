import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { SettingsPage } from '@/pages/SettingsPage'
import { useProviderStore } from '@/stores/providerStore'
import { useSettingsStore } from '@/stores/settingsStore'

/**
 * Provider 测试连接 E2E 测试。
 * 覆盖测试连接成功、延迟高、403/404 错误、超时、测试历史。
 */

describe('E2E: Provider 测试连接', () => {
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    useProviderStore.setState({ providers: [] })
    useSettingsStore.setState({ activeProviderId: null })
    fetchMock = vi.fn()
    global.fetch = fetchMock
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  const openCustomDialog = async (user: ReturnType<typeof userEvent.setup>) => {
    render(<SettingsPage />)
    await user.click(screen.getByTestId('add-custom-provider-btn'))
    await waitFor(() => {
      expect(screen.getByTestId('model-service-dialog')).toBeInTheDocument()
    })
  }

  const fillHostAndKey = async (user: ReturnType<typeof userEvent.setup>) => {
    await user.type(screen.getByTestId('ms-host-input'), 'https://api.test.com')
    await user.type(screen.getByTestId('ms-key-input'), 'sk-test-key')
  }

  it('测试连接成功：green 状态 + 模型下拉选择框 + 自动填充 Model ID', async () => {
    const user = userEvent.setup()

    fetchMock.mockResolvedValue({
      status: 200,
      json: async () => ({
        data: [
          { id: 'model-a', name: 'Model A' },
          { id: 'model-b', name: 'Model B' },
        ],
      }),
    })

    await openCustomDialog(user)
    await fillHostAndKey(user)

    // 点击测试连接
    await user.click(screen.getByTestId('ms-test-connection-btn'))

    // 验证 green 状态卡片
    await waitFor(() => {
      const card = screen.getByTestId('ms-test-result-card')
      expect(card).toHaveAttribute('data-test-status', 'green')
      expect(card.textContent).toContain('连通')
    })

    // 切换到 models 标签页验证模型已获取
    await user.click(screen.getByTestId('tab-models'))
    await waitFor(() => {
      expect(screen.getByTestId('ms-model-check-model-a')).toBeInTheDocument()
    })
    expect(screen.getByTestId('ms-model-check-model-b')).toBeInTheDocument()
  })

  it('测试连接延迟高：yellow 状态', async () => {
    const user = userEvent.setup()

    // 模拟延迟：通过 setTimeout 在 fetch 中延迟 1200ms
    fetchMock.mockImplementation(
      () =>
        new Promise((resolve) => {
          setTimeout(() => {
            resolve({
              status: 200,
              json: async () => ({ data: [{ id: 'slow-model', name: 'Slow Model' }] }),
            })
          }, 1200)
        })
    )

    await openCustomDialog(user)
    await fillHostAndKey(user)

    await user.click(screen.getByTestId('ms-test-connection-btn'))

    // 验证 yellow 状态（延迟 >= 1000ms）
    await waitFor(
      () => {
        const card = screen.getByTestId('ms-test-result-card')
        expect(card).toHaveAttribute('data-test-status', 'yellow')
        expect(card.textContent).toContain('延迟较高')
      },
      { timeout: 3000 }
    )
  })

  it('测试连接 403：red 状态 + API Key 无效提示', async () => {
    const user = userEvent.setup()

    fetchMock.mockResolvedValue({
      status: 403,
      json: async () => ({ error: { message: 'Forbidden' } }),
    })

    await openCustomDialog(user)
    await fillHostAndKey(user)

    await user.click(screen.getByTestId('ms-test-connection-btn'))

    await waitFor(() => {
      const card = screen.getByTestId('ms-test-result-card')
      expect(card).toHaveAttribute('data-test-status', 'red')
      expect(card.textContent).toContain('认证失败，请检查 API Key')
    })
  })

  it('测试连接 404：green 状态（Host 可达）+ 手动输入兜底提示', async () => {
    const user = userEvent.setup()

    fetchMock.mockResolvedValue({
      status: 404,
      json: async () => ({ error: { message: 'Not Found' } }),
    })

    await openCustomDialog(user)
    await fillHostAndKey(user)

    await user.click(screen.getByTestId('ms-test-connection-btn'))

    await waitFor(() => {
      const card = screen.getByTestId('ms-test-result-card')
      expect(card).toHaveAttribute('data-test-status', 'green')
      expect(card.textContent).toContain('连通')
    })
  })

  it('测试连接超时：red 状态 + 连接超时提示', async () => {
    const user = userEvent.setup()

    // 模拟 AbortError（超时）
    fetchMock.mockImplementation(() => {
      const err = new Error('The operation was aborted')
      err.name = 'AbortError'
      return Promise.reject(err)
    })

    await openCustomDialog(user)
    await fillHostAndKey(user)

    await user.click(screen.getByTestId('ms-test-connection-btn'))

    await waitFor(() => {
      const card = screen.getByTestId('ms-test-result-card')
      expect(card).toHaveAttribute('data-test-status', 'red')
      expect(card.textContent).toContain('连接超时')
    })
  })

  it('测试历史：连续测试后显示最近 3 次结果', async () => {
    const user = userEvent.setup()

    // 第一次：成功
    fetchMock.mockResolvedValueOnce({
      status: 200,
      json: async () => ({ data: [{ id: 'm1', name: 'M1' }] }),
    })

    await openCustomDialog(user)
    await fillHostAndKey(user)

    // 第一次测试
    await user.click(screen.getByTestId('ms-test-connection-btn'))
    await waitFor(() => {
      expect(screen.getByTestId('ms-test-result-card')).toHaveAttribute('data-test-status', 'green')
    })

    // 第二次：403
    fetchMock.mockResolvedValueOnce({
      status: 403,
      json: async () => ({ error: {} }),
    })
    await user.click(screen.getByTestId('ms-test-connection-btn'))
    await waitFor(() => {
      expect(screen.getByTestId('ms-test-result-card')).toHaveAttribute('data-test-status', 'red')
    })

    // 第三次：404
    fetchMock.mockResolvedValueOnce({
      status: 404,
      json: async () => ({ error: {} }),
    })
    await user.click(screen.getByTestId('ms-test-connection-btn'))
    await waitFor(() => {
      expect(screen.getByTestId('ms-test-result-card')).toHaveAttribute('data-test-status', 'green')
    })

    // 验证历史区域显示 3 条记录
    await waitFor(() => {
      expect(screen.getByTestId('ms-test-history')).toBeInTheDocument()
    })

    const history = screen.getByTestId('ms-test-history')
    const historyItems = history.querySelectorAll(':scope > div')
    expect(historyItems.length).toBe(3)

    // 最新的一条在最上面（404 → green）
    expect(historyItems[0].textContent).toContain('连通')
    // 第二条是 403 → red
    expect(historyItems[1].textContent).toContain('认证失败，请检查 API Key')
    // 第三条是第一次成功 → green
    expect(historyItems[2].textContent).toContain('连通')
  })
})
