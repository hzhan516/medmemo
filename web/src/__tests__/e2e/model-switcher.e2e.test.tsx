import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { ChatPage } from '@/pages/ChatPage'
import { useProviderStore } from '@/stores/providerStore'
import { useSettingsStore } from '@/stores/settingsStore'
import type { ProviderConfig } from '@/types/provider'
import { resetWailsMock } from '@/test/mocks/wails'

/**
 * 运行时模型切换器 E2E 测试。
 * 覆盖顶部选择器渲染、下拉列表、切换交互、键盘快捷键、空状态、回退逻辑。
 */

type MockProviderInput = Omit<ProviderConfig, 'id' | 'createdAt' | 'updatedAt'>

function mockFetchWithStatus(status: number) {
  return vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve({ data: [{ id: 'test-model', name: 'Test Model' }] }),
  })
}

function createMockProvider(overrides: Partial<MockProviderInput> = {}) {
  const base = {
    templateId: 'kimi',
    name: 'Kimi Test',
    apiHost: 'https://api.moonshot.cn',
    apiKey: 'test-key',
    modelId: 'kimi-lite',
    temperature: 0.7,
    timeoutMs: 30000,
    maxRetries: 2,
    group: '工作',
    enabled: true,
    sortOrder: 0,
    ...overrides,
  }
  return useProviderStore.getState().addProvider(base)
}

describe('E2E: 运行时模型切换器', () => {
  beforeEach(() => {
    resetWailsMock()
    useProviderStore.setState({ providers: [] })
    useSettingsStore.setState({
      activeProviderId: null,
      providerHealthStatus: {},
      lastSelectedProviderId: null,
    })
    vi.restoreAllMocks()
  })

  it('空状态：无 Provider 时显示"添加模型"引导按钮', () => {
    render(<ChatPage />)

    const emptyBtn = screen.getByTestId('ms-empty-btn')
    expect(emptyBtn).toBeInTheDocument()
    expect(emptyBtn).toHaveTextContent('添加模型')
  })

  it('渲染顶部选择器：展示当前 Provider 名称+状态圆点+下拉箭头', async () => {
    global.fetch = mockFetchWithStatus(200)
    const provider = createMockProvider({ name: 'Kimi Pro', modelId: 'kimi-pro' })
    useSettingsStore.setState({ activeProviderId: provider.id, activeModelId: provider.modelId })

    render(<ChatPage />)

    await waitFor(() => {
      expect(screen.getByTestId('ms-trigger')).toBeInTheDocument()
    })

    expect(screen.getByTestId('ms-current-name')).toHaveTextContent('kimi-pro')
    expect(screen.getByTestId('ms-status-dot')).toBeInTheDocument()
  })

  it('下拉列表展开：点击箭头展示按分组折叠的 Provider 列表', async () => {
    const user = userEvent.setup()
    global.fetch = mockFetchWithStatus(200)
    const p1 = createMockProvider({ name: 'Kimi A', group: '工作' })
    createMockProvider({ name: 'GPT-4', group: '云端', templateId: 'openai', apiHost: 'https://api.openai.com' })
    useSettingsStore.setState({ activeProviderId: p1.id, activeModelId: p1.modelId })

    render(<ChatPage />)

    await waitFor(() => {
      expect(screen.getByTestId('ms-trigger')).toBeInTheDocument()
    })

    await user.click(screen.getByTestId('ms-trigger'))

    expect(screen.getByTestId('ms-dropdown')).toBeInTheDocument()
    expect(screen.getByTestId('ms-group-工作')).toBeInTheDocument()
    expect(screen.getByTestId('ms-group-云端')).toBeInTheDocument()
    expect(screen.getByTestId(`ms-model-${p1.id}-${p1.modelId}`)).toBeInTheDocument()
  })

  it('切换 Provider：点击 Green Provider 后顶部更新并显示 Toast', async () => {
    const user = userEvent.setup()
    global.fetch = mockFetchWithStatus(200)
    const p1 = createMockProvider({ name: 'Kimi A', group: '工作' })
    const p2 = createMockProvider({ name: 'Kimi B', group: '工作', templateId: 'kimi2', apiHost: 'https://api2.moonshot.cn' })
    useSettingsStore.setState({ activeProviderId: p1.id })
    // 预设 health status
    useSettingsStore.setState({
      providerHealthStatus: { [p1.id]: 'green', [p2.id]: 'green' },
    })

    render(<ChatPage />)

    await waitFor(() => {
      expect(screen.getByTestId('ms-trigger')).toBeInTheDocument()
    })

    await user.click(screen.getByTestId('ms-trigger'))
    await user.click(screen.getByTestId(`ms-model-${p2.id}-${p2.modelId}`))

    await waitFor(() => {
      expect(screen.getByTestId('ms-current-name')).toHaveTextContent('kimi-lite')
    })

    // Toast 显示
    await waitFor(() => {
      expect(screen.getByTestId('ms-toast')).toHaveTextContent('已切换至 kimi-lite')
    })

    // 下拉关闭
    expect(screen.queryByTestId('ms-dropdown')).not.toBeInTheDocument()

    // activeProviderId 已更新
    expect(useSettingsStore.getState().activeProviderId).toBe(p2.id)
    expect(useSettingsStore.getState().lastSelectedProviderId).toBe(p2.id)
  })

  it('Yellow Provider 置灰不可点击', async () => {
    const user = userEvent.setup()
    global.fetch = vi.fn().mockImplementation(() => new Promise(() => {}))
    const p1 = createMockProvider({ name: 'Kimi A' })
    const p2 = createMockProvider({ name: 'Kimi Slow', templateId: 'kimi2', apiHost: 'https://slow.moonshot.cn' })
    useSettingsStore.setState({
      activeProviderId: p1.id,
      activeModelId: p1.modelId,
      providerHealthStatus: { [p1.id]: 'green', [p2.id]: 'yellow' },
    })

    render(<ChatPage />)

    await waitFor(() => {
      expect(screen.getByTestId('ms-trigger')).toBeInTheDocument()
    })

    await user.click(screen.getByTestId('ms-trigger'))

    const slowItem = screen.getByTestId(`ms-model-${p2.id}-${p2.modelId}`)
    expect(slowItem).toHaveClass('opacity-50')
    expect(slowItem).toHaveAttribute('disabled')
  })

  it('Red Provider 不展示在下拉列表中', async () => {
    const user = userEvent.setup()
    global.fetch = vi.fn().mockImplementation(() => new Promise(() => {}))
    const p1 = createMockProvider({ name: 'Kimi A' })
    const p2 = createMockProvider({ name: 'Kimi Dead', templateId: 'kimi2', apiHost: 'https://dead.moonshot.cn' })
    useSettingsStore.setState({
      activeProviderId: p1.id,
      activeModelId: p1.modelId,
      providerHealthStatus: { [p1.id]: 'green', [p2.id]: 'red' },
    })

    render(<ChatPage />)

    await waitFor(() => {
      expect(screen.getByTestId('ms-trigger')).toBeInTheDocument()
    })

    await user.click(screen.getByTestId('ms-trigger'))

    expect(screen.getByTestId(`ms-model-${p1.id}-${p1.modelId}`)).toBeInTheDocument()
    expect(screen.queryByTestId(`ms-model-${p2.id}-${p2.modelId}`)).not.toBeInTheDocument()
  })

  it('键盘快捷键 Ctrl+Shift+↓ 循环切换可用 Provider', async () => {
    const p1 = createMockProvider({ name: 'Kimi A' })
    const p2 = createMockProvider({ name: 'Kimi B', templateId: 'kimi2', apiHost: 'https://api2.moonshot.cn' })
    useSettingsStore.setState({
      activeProviderId: p1.id,
      activeModelId: p1.modelId,
      providerHealthStatus: { [p1.id]: 'green', [p2.id]: 'green' },
    })

    render(<ChatPage />)

    await waitFor(() => {
      expect(screen.getByTestId('ms-current-name')).toHaveTextContent('kimi-lite')
    })

    // Ctrl+Shift+↓ → 切换到下一个
    const event = new KeyboardEvent('keydown', {
      key: 'ArrowDown',
      ctrlKey: true,
      shiftKey: true,
      bubbles: true,
    })
    window.dispatchEvent(event)

    await waitFor(() => {
      expect(useSettingsStore.getState().activeProviderId).toBe(p2.id)
    })

    // Ctrl+Shift+↑ → 切换回上一个
    const eventUp = new KeyboardEvent('keydown', {
      key: 'ArrowUp',
      ctrlKey: true,
      shiftKey: true,
      bubbles: true,
    })
    window.dispatchEvent(eventUp)

    await waitFor(() => {
      expect(useSettingsStore.getState().activeProviderId).toBe(p1.id)
    })
  })

  it('偏好持久化：lastSelectedProviderId 随 activeProviderId 同步更新', async () => {
    const user = userEvent.setup()
    const p1 = createMockProvider({ name: 'Kimi A' })
    const p2 = createMockProvider({ name: 'Kimi B', templateId: 'kimi2', apiHost: 'https://api2.moonshot.cn' })
    useSettingsStore.setState({
      activeProviderId: p1.id,
      providerHealthStatus: { [p1.id]: 'green', [p2.id]: 'green' },
    })

    render(<ChatPage />)

    await waitFor(() => {
      expect(screen.getByTestId('ms-trigger')).toBeInTheDocument()
    })

    await user.click(screen.getByTestId('ms-trigger'))
    await user.click(screen.getByTestId(`ms-model-${p2.id}-${p2.modelId}`))

    await waitFor(() => {
      expect(useSettingsStore.getState().lastSelectedProviderId).toBe(p2.id)
    })
  })

  it('点击外部关闭下拉面板', async () => {
    const user = userEvent.setup()
    const p1 = createMockProvider({ name: 'Kimi A' })
    useSettingsStore.setState({
      activeProviderId: p1.id,
      providerHealthStatus: { [p1.id]: 'green' },
    })

    render(<ChatPage />)

    await waitFor(() => {
      expect(screen.getByTestId('ms-trigger')).toBeInTheDocument()
    })

    await user.click(screen.getByTestId('ms-trigger'))
    expect(screen.getByTestId('ms-dropdown')).toBeInTheDocument()

    // 点击面板外部（body）
    await user.click(document.body)

    await waitFor(() => {
      expect(screen.queryByTestId('ms-dropdown')).not.toBeInTheDocument()
    })
  })

  it('空状态下拉：无可用 Provider 时显示"添加模型"按钮', async () => {
    const user = userEvent.setup()
    global.fetch = vi.fn().mockImplementation(() => new Promise(() => {}))
    const p1 = createMockProvider({ name: 'Kimi A' })
    useSettingsStore.setState({
      activeProviderId: p1.id,
      providerHealthStatus: { [p1.id]: 'red' },
    })

    render(<ChatPage />)

    await waitFor(() => {
      expect(screen.getByTestId('ms-trigger')).toBeInTheDocument()
    })

    await user.click(screen.getByTestId('ms-trigger'))

    // red 状态的 Provider 不展示，所以下拉内为空状态
    expect(screen.getByText('暂无可用模型')).toBeInTheDocument()
    expect(screen.getByTestId('ms-dropdown-add-btn')).toBeInTheDocument()
  })
})
