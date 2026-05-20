import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { SettingsPage } from '@/pages/SettingsPage'
import { useProviderStore } from '@/stores/providerStore'
import { useSettingsStore } from '@/stores/settingsStore'
import { resetWailsMock } from '@/test/mocks/wails'

/**
 * Provider 自定义表单 E2E 测试。
 * 覆盖自定义添加、表单验证、编辑模式、分组展示、删除确认、活跃模型保护。
 */

describe('E2E: Provider 自定义表单', () => {
  beforeEach(() => {
    resetWailsMock()
    useProviderStore.setState({ providers: [] })
    useSettingsStore.setState({ activeProviderId: null })
  })

  it('自定义添加 Provider：填写表单 → 保存 → 列表更新', async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)

    // 打开自定义表单
    const addCustomBtn = screen.getByTestId('add-custom-provider-btn')
    await user.click(addCustomBtn)

    await waitFor(() => {
      expect(screen.getByTestId('provider-custom-dialog')).toBeInTheDocument()
    })

    // 填写表单
    await user.type(screen.getByTestId('pc-name-input'), '我的自定义模型')
    await user.type(screen.getByTestId('pc-host-input'), 'https://api.custom.com')
    await user.type(screen.getByTestId('pc-key-input'), 'sk-custom-key')
    await user.type(screen.getByTestId('pc-model-input'), 'custom-model-v1')

    // 保存
    const saveBtn = screen.getByTestId('pc-save-btn')
    await user.click(saveBtn)

    // 验证弹窗关闭
    await waitFor(() => {
      expect(screen.queryByTestId('provider-custom-dialog')).not.toBeInTheDocument()
    })

    // 验证 store 中已添加
    await waitFor(() => {
      const state = useProviderStore.getState()
      expect(state.providers.length).toBe(1)
      expect(state.providers[0].name).toBe('我的自定义模型')
      expect(state.providers[0].apiHost).toBe('https://api.custom.com')
      expect(state.providers[0].modelId).toBe('custom-model-v1')
      expect(state.providers[0].group).toBe('默认')
      expect(state.providers[0].templateId).toBe('custom')
    })

    // 验证分组列表中出现
    await waitFor(() => {
      expect(screen.getByTestId('provider-group-list')).toBeInTheDocument()
    })
  })

  it('表单验证：无效 API Host 提示错误', async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)

    await user.click(screen.getByTestId('add-custom-provider-btn'))

    await waitFor(() => {
      expect(screen.getByTestId('provider-custom-dialog')).toBeInTheDocument()
    })

    // 输入无效 URL
    await user.type(screen.getByTestId('pc-host-input'), 'not-a-url')
    await user.type(screen.getByTestId('pc-name-input'), '测试')
    await user.type(screen.getByTestId('pc-model-input'), 'test')

    // 触发验证（blur）
    await user.click(screen.getByTestId('pc-name-input'))

    // 保存按钮应被禁用
    const saveBtn = screen.getByTestId('pc-save-btn')
    expect(saveBtn).toBeDisabled()

    // 验证错误提示
    await waitFor(() => {
      expect(screen.getByText('必须以 http:// 或 https:// 开头')).toBeInTheDocument()
    })
  })

  it('表单验证：空必填字段提示错误', async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)

    await user.click(screen.getByTestId('add-custom-provider-btn'))

    await waitFor(() => {
      expect(screen.getByTestId('provider-custom-dialog')).toBeInTheDocument()
    })

    // 直接点保存（所有必填为空）
    const saveBtn = screen.getByTestId('pc-save-btn')
    expect(saveBtn).toBeDisabled()
  })

  it('编辑模式：点击编辑 → 表单预填 → 修改保存', async () => {
    const user = userEvent.setup()

    // 预置一个 Provider
    useProviderStore.getState().addProvider({
      templateId: 'custom',
      name: '原始名称',
      apiHost: 'https://api.original.com',
      apiKey: 'sk-original',
      modelId: 'original-model',
      temperature: 0.5,
      timeoutMs: 15000,
      maxRetries: 1,
      group: '测试分组',
      enabled: true,
      sortOrder: 0,
    })

    render(<SettingsPage />)

    // 展开分组
    await waitFor(() => {
      expect(screen.getByTestId('provider-group-list')).toBeInTheDocument()
    })

    const state = useProviderStore.getState()
    const providerId = state.providers[0].id

    // 点击编辑
    const editBtn = screen.getByTestId(`provider-edit-btn-${providerId}`)
    await user.click(editBtn)

    // 验证弹窗出现且预填
    await waitFor(() => {
      expect(screen.getByTestId('provider-custom-dialog')).toBeInTheDocument()
    })

    expect(screen.getByDisplayValue('原始名称')).toBeInTheDocument()
    expect(screen.getByDisplayValue('https://api.original.com')).toBeInTheDocument()
    expect(screen.getByDisplayValue('original-model')).toBeInTheDocument()

    // 修改名称
    const nameInput = screen.getByTestId('pc-name-input')
    await user.clear(nameInput)
    await user.type(nameInput, '修改后的名称')

    // 保存
    await user.click(screen.getByTestId('pc-save-btn'))

    // 验证更新
    await waitFor(() => {
      const updated = useProviderStore.getState().providers[0]
      expect(updated.name).toBe('修改后的名称')
      expect(updated.apiHost).toBe('https://api.original.com')
    })
  })

  it('分组展示：不同分组的 Provider 按分组折叠', async () => {
    const user = userEvent.setup()

    // 预置两个不同分组的 Provider
    useProviderStore.getState().addProvider({
      templateId: 'custom',
      name: '工作模型A',
      apiHost: 'https://api.work.com',
      apiKey: '',
      modelId: 'work-model',
      temperature: 0.7,
      timeoutMs: 30000,
      maxRetries: 3,
      group: '工作',
      enabled: true,
      sortOrder: 0,
    })
    useProviderStore.getState().addProvider({
      templateId: 'custom',
      name: '个人模型B',
      apiHost: 'https://api.personal.com',
      apiKey: '',
      modelId: 'personal-model',
      temperature: 0.7,
      timeoutMs: 30000,
      maxRetries: 3,
      group: '个人',
      enabled: true,
      sortOrder: 0,
    })

    render(<SettingsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('provider-group-list')).toBeInTheDocument()
    })

    // 验证两个分组头都存在
    expect(screen.getByTestId('group-header-个人')).toBeInTheDocument()
    expect(screen.getByTestId('group-header-工作')).toBeInTheDocument()

    // 验证 Provider 在正确的分组下
    const personalItems = screen.getAllByTestId(/^provider-item-/)
    expect(personalItems.length).toBe(2)
  })

  it('删除确认：点击删除 → 弹窗确认 → 删除成功', async () => {
    const user = userEvent.setup()

    useProviderStore.getState().addProvider({
      templateId: 'custom',
      name: '待删除',
      apiHost: 'https://api.delete.com',
      apiKey: '',
      modelId: 'delete-model',
      temperature: 0.7,
      timeoutMs: 30000,
      maxRetries: 3,
      group: '默认',
      enabled: true,
      sortOrder: 0,
    })

    render(<SettingsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('provider-group-list')).toBeInTheDocument()
    })

    const state = useProviderStore.getState()
    const providerId = state.providers[0].id

    // 点击删除
    const deleteBtn = screen.getByTestId(`provider-delete-btn-${providerId}`)
    await user.click(deleteBtn)

    // 确认弹窗出现
    await waitFor(() => {
      expect(screen.getByTestId('delete-confirm-dialog')).toBeInTheDocument()
    })

    // 点击确认删除
    await user.click(screen.getByTestId('delete-confirm-btn'))

    // 验证已删除
    await waitFor(() => {
      expect(useProviderStore.getState().providers.length).toBe(0)
    })
  })

  it('活跃模型保护：设为活跃后删除按钮被禁用', async () => {
    const user = userEvent.setup()

    useProviderStore.getState().addProvider({
      templateId: 'custom',
      name: '活跃模型',
      apiHost: 'https://api.active.com',
      apiKey: '',
      modelId: 'active-model',
      temperature: 0.7,
      timeoutMs: 30000,
      maxRetries: 3,
      group: '默认',
      enabled: true,
      sortOrder: 0,
    })

    const providerId = useProviderStore.getState().providers[0].id
    useSettingsStore.setState({ activeProviderId: providerId })

    render(<SettingsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('provider-group-list')).toBeInTheDocument()
    })

    // 点击删除
    await user.click(screen.getByTestId(`provider-delete-btn-${providerId}`))

    // 确认弹窗出现，且删除按钮被禁用（因是活跃模型）
    await waitFor(() => {
      expect(screen.getByTestId('delete-confirm-dialog')).toBeInTheDocument()
    })

    const confirmBtn = screen.getByTestId('delete-confirm-btn')
    expect(confirmBtn).toBeDisabled()

    // 关闭弹窗
    await user.click(screen.getByLabelText('关闭'))

    // 验证未删除
    expect(useProviderStore.getState().providers.length).toBe(1)
  })

  it('分组创建：添加 Provider 时创建新分组', async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)

    await user.click(screen.getByTestId('add-custom-provider-btn'))

    await waitFor(() => {
      expect(screen.getByTestId('provider-custom-dialog')).toBeInTheDocument()
    })

    // 填写表单
    await user.type(screen.getByTestId('pc-name-input'), '新分组模型')
    await user.type(screen.getByTestId('pc-host-input'), 'https://api.new.com')
    await user.type(screen.getByTestId('pc-model-input'), 'new-model')

    // 切换到创建新分组
    const groupSelect = screen.getByTestId('pc-group-select')
    await user.selectOptions(groupSelect, '__new__')

    // 输入新分组名
    await waitFor(() => {
      expect(screen.getByTestId('pc-group-input')).toBeInTheDocument()
    })

    await user.type(screen.getByTestId('pc-group-input'), '我的新分组')

    // 保存
    await user.click(screen.getByTestId('pc-save-btn'))

    await waitFor(() => {
      expect(screen.queryByTestId('provider-custom-dialog')).not.toBeInTheDocument()
    })

    // 验证分组列表中有新分组
    await waitFor(() => {
      expect(screen.getByTestId('group-header-我的新分组')).toBeInTheDocument()
    })

    const provider = useProviderStore.getState().providers[0]
    expect(provider.group).toBe('我的新分组')
  })
})
