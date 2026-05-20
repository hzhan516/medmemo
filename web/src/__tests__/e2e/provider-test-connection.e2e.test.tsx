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
      expect(screen.getByTestId('provider-custom-dialog')).toBeInTheDocument()
    })
  }

  const fillHostAndKey = async (user: ReturnType<typeof userEvent.setup>) => {
    await user.type(screen.getByTestId('pc-host-input'), 'https://api.test.com')
    await user.type(screen.getByTestId('pc-key-input'), 'sk-test-key')
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
    await user.click(screen.getByTestId('pc-test-connection-btn'))

    // 验证 green 状态卡片
    await waitFor(() => {
      const card = screen.getByTestId('pc-test-result-card')
      expect(card).toHaveAttribute('data-test-status', 'green')
      expect(card.textContent).toContain('连通')
    })

    // 验证下拉选择框出现
    await waitFor(() => {
      expect(screen.getByTestId('pc-model-select')).toBeInTheDocument()
    })

    // 验证 Model ID 被自动填充为第一个模型
    await waitFor(() => {
      expect(screen.getByTestId('pc-model-select')).toHaveValue('model-a')
    })

    // 验证下拉选项包含两个模型
    const select = screen.getByTestId('pc-model-select') as HTMLSelectElement
    expect(select.options.length).toBe(3) // 包括空选项
    expect(select.options[1].value).toBe('model-a')
    expect(select.options[2].value).toBe('model-b')
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

    await user.click(screen.getByTestId('pc-test-connection-btn'))

    // 验证 yellow 状态（延迟 >= 1000ms）
    await waitFor(
      () => {
        const card = screen.getByTestId('pc-test-result-card')
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

    await user.click(screen.getByTestId('pc-test-connection-btn'))

    await waitFor(() => {
      const card = screen.getByTestId('pc-test-result-card')
      expect(card).toHaveAttribute('data-test-status', 'red')
      expect(card.textContent).toContain('API Key 无效')
    })

    // Model ID 应保持手动输入模式
    expect(screen.getByTestId('pc-model-input')).toBeInTheDocument()
  })

  it('测试连接 404：green 状态（Host 可达）+ 手动输入兜底提示', async () => {
    const user = userEvent.setup()

    fetchMock.mockResolvedValue({
      status: 404,
      json: async () => ({ error: { message: 'Not Found' } }),
    })

    await openCustomDialog(user)
    await fillHostAndKey(user)

    await user.click(screen.getByTestId('pc-test-connection-btn'))

    await waitFor(() => {
      const card = screen.getByTestId('pc-test-result-card')
      expect(card).toHaveAttribute('data-test-status', 'green')
      expect(card.textContent).toContain('连通')
    })

    // 手动输入模式，placeholder 提示不支持自动获取
    const modelInput = screen.getByTestId('pc-model-input')
    expect(modelInput).toBeInTheDocument()
    expect(modelInput).toHaveAttribute('placeholder', expect.stringContaining('不支持自动获取'))
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

    await user.click(screen.getByTestId('pc-test-connection-btn'))

    await waitFor(() => {
      const card = screen.getByTestId('pc-test-result-card')
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
    await user.click(screen.getByTestId('pc-test-connection-btn'))
    await waitFor(() => {
      expect(screen.getByTestId('pc-test-result-card')).toHaveAttribute('data-test-status', 'green')
    })

    // 第二次：403
    fetchMock.mockResolvedValueOnce({
      status: 403,
      json: async () => ({ error: {} }),
    })
    await user.click(screen.getByTestId('pc-test-connection-btn'))
    await waitFor(() => {
      expect(screen.getByTestId('pc-test-result-card')).toHaveAttribute('data-test-status', 'red')
    })

    // 第三次：404
    fetchMock.mockResolvedValueOnce({
      status: 404,
      json: async () => ({ error: {} }),
    })
    await user.click(screen.getByTestId('pc-test-connection-btn'))
    await waitFor(() => {
      expect(screen.getByTestId('pc-test-result-card')).toHaveAttribute('data-test-status', 'green')
    })

    // 验证历史区域显示 3 条记录
    await waitFor(() => {
      expect(screen.getByTestId('pc-test-history')).toBeInTheDocument()
    })

    expect(screen.getByTestId('pc-test-history-item-0')).toBeInTheDocument()
    expect(screen.getByTestId('pc-test-history-item-1')).toBeInTheDocument()
    expect(screen.getByTestId('pc-test-history-item-2')).toBeInTheDocument()

    // 最新的一条在最上面（404 → green）
    expect(screen.getByTestId('pc-test-history-item-0').textContent).toContain('连通')
    // 第二条是 403 → red
    expect(screen.getByTestId('pc-test-history-item-1').textContent).toContain('API Key 无效')
    // 第三条是第一次成功 → green
    expect(screen.getByTestId('pc-test-history-item-2').textContent).toContain('连通')
  })
})
