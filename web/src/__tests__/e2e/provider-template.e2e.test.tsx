import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { SettingsPage } from '@/pages/SettingsPage'
import { useProviderStore } from '@/stores/providerStore'
import { setMockHandlers, EventsEmit, resetWailsMock } from '@/test/mocks/wails'

/**
 * Provider 模板列表 E2E 测试。
 * 覆盖模板加载、搜索过滤、卡片点击、弹窗预填、保存流程。
 */

describe('E2E: Provider 模板列表', () => {
  beforeEach(() => {
    resetWailsMock()
    useProviderStore.setState({ providers: [] })
  })

  it('加载并展示 Provider 模板卡片网格', async () => {
    render(<SettingsPage />)

    // 滚动到模型提供商区域
    const heading = screen.getByText('模型提供商')
    expect(heading).toBeInTheDocument()

    // 等待模板加载
    await waitFor(() => {
      // 至少应展示部分模板卡片（OpenAI、Kimi 等）
      expect(screen.getByTestId('provider-card-openai')).toBeInTheDocument()
    }, { timeout: 3000 })

    expect(screen.getByTestId('provider-card-kimi')).toBeInTheDocument()
    expect(screen.getByTestId('provider-card-deepseek')).toBeInTheDocument()
  })

  it('搜索过滤：输入关键词后仅展示匹配的模板', async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('provider-card-openai')).toBeInTheDocument()
    }, { timeout: 3000 })

    const searchInput = screen.getByTestId('provider-search-input')
    await user.type(searchInput, 'Kimi')

    // Kimi 应存在
    expect(screen.getByTestId('provider-card-kimi')).toBeInTheDocument()
    // OpenAI 应被过滤掉
    expect(screen.queryByTestId('provider-card-openai')).not.toBeInTheDocument()
  })

  it('搜索无结果时展示空状态提示', async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('provider-card-openai')).toBeInTheDocument()
    }, { timeout: 3000 })

    const searchInput = screen.getByTestId('provider-search-input')
    await user.type(searchInput, '不存在的Provider')

    await waitFor(() => {
      expect(screen.getByText('未找到匹配的 Provider')).toBeInTheDocument()
    })
  })

  it('点击模板卡片 → 弹出预填表单 → 输入 API Key → 保存 → 列表更新', async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('provider-card-openai')).toBeInTheDocument()
    }, { timeout: 3000 })

    // 点击 OpenAI 卡片
    const openaiCard = screen.getByTestId('provider-card-openai')
    await user.click(openaiCard)

    // 验证弹窗出现
    await waitFor(() => {
      expect(screen.getByTestId('model-service-dialog')).toBeInTheDocument()
    })

    // 验证预填字段
    expect(screen.getByDisplayValue('OpenAI')).toBeInTheDocument()
    expect(screen.getByDisplayValue('https://api.openai.com')).toBeInTheDocument()

    // 输入 API Key
    const apiKeyInput = screen.getByTestId('ms-key-input')
    await user.type(apiKeyInput, 'sk-test-key-123')

    // 点击保存
    const saveBtn = screen.getByTestId('ms-save-btn')
    await user.click(saveBtn)

    // 验证弹窗关闭
    await waitFor(() => {
      expect(screen.queryByTestId('model-service-dialog')).not.toBeInTheDocument()
    })

    // 验证 store 中已添加
    await waitFor(() => {
      const state = useProviderStore.getState()
      expect(state.providers.length).toBe(1)
      expect(state.providers[0].name).toBe('OpenAI')
      expect(state.providers[0].templateId).toBe('openai')
    })

    // 验证已添加标识
    await waitFor(() => {
      expect(screen.getByText('已添加')).toBeInTheDocument()
    })
  })

  it('本地 Provider 无需 API Key 可直接保存', async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('provider-card-ollama')).toBeInTheDocument()
    }, { timeout: 3000 })

    // 点击 Ollama 卡片
    const ollamaCard = screen.getByTestId('provider-card-ollama')
    await user.click(ollamaCard)

    await waitFor(() => {
      expect(screen.getByTestId('model-service-dialog')).toBeInTheDocument()
    })

    // 直接保存（ModelServiceDialog 中 API Key 非必填，本地 provider 可直接保存）
    const saveBtn = screen.getByTestId('ms-save-btn')
    await user.click(saveBtn)

    // 验证添加成功
    await waitFor(() => {
      const state = useProviderStore.getState()
      expect(state.providers.length).toBe(1)
      expect(state.providers[0].templateId).toBe('ollama')
    })
  })

  it('云端 Provider 未输入 API Key 仍可保存，needsApiKey 自动标记为 true', async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('provider-card-openai')).toBeInTheDocument()
    }, { timeout: 3000 })

    await user.click(screen.getByTestId('provider-card-openai'))

    await waitFor(() => {
      expect(screen.getByTestId('model-service-dialog')).toBeInTheDocument()
    })

    // 不输入 API Key 直接保存（新版 ModelServiceDialog 允许空 API Key）
    const saveBtn = screen.getByTestId('ms-save-btn')
    await user.click(saveBtn)

    // 验证弹窗关闭且添加成功
    await waitFor(() => {
      expect(screen.queryByTestId('model-service-dialog')).not.toBeInTheDocument()
    })

    // 验证已添加，且 needsApiKey 为 true
    const state = useProviderStore.getState()
    expect(state.providers.length).toBe(1)
    expect(state.providers[0].needsApiKey).toBe(true)
  })

  it('删除已添加的 Provider', async () => {
    const user = userEvent.setup()

    // 预置一个 Provider
    useProviderStore.getState().addProvider({
      templateId: 'openai',
      name: 'OpenAI',
      apiHost: 'https://api.openai.com',
      apiKey: '',
      modelId: 'gpt-4o',
      temperature: 0.7,
      timeoutMs: 30000,
      maxRetries: 3,
      group: '云端',
      enabled: true,
      sortOrder: 0,
    })

    render(<SettingsPage />)

    // 验证已添加列表中存在（使用 aria-label 定位删除按钮来间接验证）
    const deleteBtn = await waitFor(() => screen.getByLabelText('删除 OpenAI'))

    // 点击删除按钮 → 弹出确认
    await user.click(deleteBtn)

    await waitFor(() => {
      expect(screen.getByTestId('delete-confirm-dialog')).toBeInTheDocument()
    })

    // 确认删除
    await user.click(screen.getByTestId('delete-confirm-btn'))

    // 验证已删除
    await waitFor(() => {
      expect(useProviderStore.getState().providers.length).toBe(0)
    })
  })
})
