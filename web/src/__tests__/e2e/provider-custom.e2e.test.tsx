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

    const addCustomBtn = screen.getByTestId('add-custom-provider-btn')
    await user.click(addCustomBtn)

    await waitFor(() => {
      expect(screen.getByTestId('model-service-dialog')).toBeInTheDocument()
    })

    await user.type(screen.getByTestId('ms-name-input'), '我的自定义模型')
    await user.type(screen.getByTestId('ms-host-input'), 'https://api.custom.com')
    await user.type(screen.getByTestId('ms-key-input'), 'sk-custom-key')
    await user.click(screen.getByTestId('tab-models'))
    await user.type(screen.getByTestId('ms-new-model-id'), 'custom-model-v1')
    await user.click(screen.getByTestId('ms-add-model-btn'))

    const saveBtn = screen.getByTestId('ms-save-btn')
    await user.click(saveBtn)

    await waitFor(() => {
      expect(screen.queryByTestId('model-service-dialog')).not.toBeInTheDocument()
    })

    await waitFor(() => {
      const state = useProviderStore.getState()
      expect(state.providers.length).toBe(1)
      expect(state.providers[0].name).toBe('我的自定义模型')
      expect(state.providers[0].apiHost).toBe('https://api.custom.com')
      expect(state.providers[0].modelId).toBe('custom-model-v1')
      expect(state.providers[0].group).toBe('默认')
      expect(state.providers[0].templateId).toBe('custom')
    })

    await waitFor(() => {
      expect(screen.getByTestId('provider-group-list')).toBeInTheDocument()
    })
  })

  it('表单验证：无效 API Host 提示错误', async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)

    await user.click(screen.getByTestId('add-custom-provider-btn'))

    await waitFor(() => {
      expect(screen.getByTestId('model-service-dialog')).toBeInTheDocument()
    })

    await user.type(screen.getByTestId('ms-host-input'), 'not-a-url')
    await user.type(screen.getByTestId('ms-name-input'), '测试')
    // 切换到模型标签页添加模型
    await user.click(screen.getByTestId('tab-models'))
    await user.type(screen.getByTestId('ms-new-model-id'), 'test')
    await user.click(screen.getByTestId('ms-add-model-btn'))

    await user.click(screen.getByTestId('tab-service'))

    await user.click(screen.getByTestId('ms-name-input'))

    // 点击保存：无效 API Host 应阻止提交并显示错误（不再依赖按钮禁用）
    await user.click(screen.getByTestId('ms-save-btn'))

    await waitFor(() => {
      expect(screen.getByText('必须以 http:// 或 https:// 开头')).toBeInTheDocument()
    })
    // 校验失败时弹窗保持打开（未保存）
    expect(screen.getByTestId('model-service-dialog')).toBeInTheDocument()
  })

  it('表单验证：空必填字段提示错误', async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)

    await user.click(screen.getByTestId('add-custom-provider-btn'))

    await waitFor(() => {
      expect(screen.getByTestId('model-service-dialog')).toBeInTheDocument()
    })

    const saveBtn = screen.getByTestId('ms-save-btn')
    // 点击保存：空必填字段应阻止提交并提示错误（不再依赖按钮禁用）
    await user.click(saveBtn)

    await waitFor(() => {
      expect(screen.getByText('名称不能为空')).toBeInTheDocument()
    })
    expect(screen.getByText('必须以 http:// 或 https:// 开头')).toBeInTheDocument()
    // 弹窗保持打开（未保存）
    expect(screen.getByTestId('model-service-dialog')).toBeInTheDocument()
  })

  it('编辑模式：点击编辑 → 表单预填 → 修改保存', async () => {
    const user = userEvent.setup()

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

    await waitFor(() => {
      expect(screen.getByTestId('provider-group-list')).toBeInTheDocument()
    })

    const state = useProviderStore.getState()
    const providerId = state.providers[0].id

    const editBtn = screen.getByTestId(`provider-edit-btn-${providerId}`)
    await user.click(editBtn)

    await waitFor(() => {
      expect(screen.getByTestId('model-service-dialog')).toBeInTheDocument()
    })

    expect(screen.getByDisplayValue('原始名称')).toBeInTheDocument()
    expect(screen.getByDisplayValue('https://api.original.com')).toBeInTheDocument()

    await user.click(screen.getByTestId('tab-models'))
    expect(screen.getByTestId('ms-model-check-original-model')).toBeInTheDocument()
    await user.click(screen.getByTestId('tab-service'))

    const nameInput = screen.getByTestId('ms-name-input')
    await user.clear(nameInput)
    await user.type(nameInput, '修改后的名称')

    await user.click(screen.getByTestId('ms-save-btn'))

    await waitFor(() => {
      const updated = useProviderStore.getState().providers[0]
      expect(updated.name).toBe('修改后的名称')
      expect(updated.apiHost).toBe('https://api.original.com')
    })
  })

  it('分组展示：不同分组的 Provider 按分组折叠', async () => {
    const user = userEvent.setup()

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

    expect(screen.getByTestId('group-header-个人')).toBeInTheDocument()
    expect(screen.getByTestId('group-header-工作')).toBeInTheDocument()

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

    const deleteBtn = screen.getByTestId(`provider-delete-btn-${providerId}`)
    await user.click(deleteBtn)

    await waitFor(() => {
      expect(screen.getByTestId('delete-confirm-dialog')).toBeInTheDocument()
    })

    await user.click(screen.getByTestId('delete-confirm-btn'))

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

    await user.click(screen.getByTestId(`provider-delete-btn-${providerId}`))

    await waitFor(() => {
      expect(screen.getByTestId('delete-confirm-dialog')).toBeInTheDocument()
    })

    const confirmBtn = screen.getByTestId('delete-confirm-btn')
    expect(confirmBtn).toBeDisabled()

    await user.click(screen.getByLabelText('关闭'))

    expect(useProviderStore.getState().providers.length).toBe(1)
  })

  it('分组创建：添加 Provider 时创建新分组', async () => {
    const user = userEvent.setup()
    render(<SettingsPage />)

    await user.click(screen.getByTestId('add-custom-provider-btn'))

    await waitFor(() => {
      expect(screen.getByTestId('model-service-dialog')).toBeInTheDocument()
    })

    await user.type(screen.getByTestId('ms-name-input'), '新分组模型')
    await user.type(screen.getByTestId('ms-host-input'), 'https://api.new.com')
    // 切换到模型标签页添加模型
    await user.click(screen.getByTestId('tab-models'))
    await user.type(screen.getByTestId('ms-new-model-id'), 'new-model')
    await user.click(screen.getByTestId('ms-add-model-btn'))

    // 切换回 service 标签页配置分组
    await user.click(screen.getByTestId('tab-service'))

    const groupSelect = screen.getByTestId('ms-group-select')
    await user.selectOptions(groupSelect, '__new__')

    await waitFor(() => {
      expect(screen.getByTestId('ms-group-input')).toBeInTheDocument()
    })

    await user.type(screen.getByTestId('ms-group-input'), '我的新分组')

    await user.click(screen.getByTestId('ms-save-btn'))

    await waitFor(() => {
      expect(screen.queryByTestId('model-service-dialog')).not.toBeInTheDocument()
    })

    await waitFor(() => {
      expect(screen.getByTestId('group-header-我的新分组')).toBeInTheDocument()
    })

    const provider = useProviderStore.getState().providers[0]
    expect(provider.group).toBe('我的新分组')
  })
})
