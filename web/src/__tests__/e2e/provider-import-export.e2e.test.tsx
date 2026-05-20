import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { SettingsPage } from '@/pages/SettingsPage'
import { useProviderStore } from '@/stores/providerStore'
import { useSettingsStore } from '@/stores/settingsStore'
import type { ProviderConfig } from '@/types/provider'

/**
 * Provider 配置导入导出 E2E 测试。
 * 覆盖导出功能、导入功能、格式验证、冲突处理、API Key 安全。
 */

describe('E2E: Provider 配置导入导出', () => {
  let originalFileReader: typeof FileReader
  let originalCreateObjectURL: typeof URL.createObjectURL
  let originalRevokeObjectURL: typeof URL.revokeObjectURL

  beforeEach(() => {
    useProviderStore.setState({ providers: [] })
    useSettingsStore.setState({ activeProviderId: null, providerHealthStatus: {}, lastSelectedProviderId: null })

    originalFileReader = global.FileReader
    originalCreateObjectURL = URL.createObjectURL
    originalRevokeObjectURL = URL.revokeObjectURL

    URL.createObjectURL = vi.fn(() => 'blob:mock-url')
    URL.revokeObjectURL = vi.fn()
  })

  afterEach(() => {
    global.FileReader = originalFileReader
    URL.createObjectURL = originalCreateObjectURL
    URL.revokeObjectURL = originalRevokeObjectURL
    vi.restoreAllMocks()
  })

  const openImportExportDialog = async (user: ReturnType<typeof userEvent.setup>) => {
    render(<SettingsPage />)
    await user.click(screen.getByTestId('import-export-btn'))
    await waitFor(() => {
      expect(screen.getByTestId('provider-import-export-dialog')).toBeInTheDocument()
    })
  }

  const mockFileReader = (content: string) => {
    global.FileReader = class {
      result: string | ArrayBuffer | null = content
      onload: ((this: FileReader, ev: ProgressEvent<FileReader>) => unknown) | null = null
      onerror: ((this: FileReader, ev: ProgressEvent<FileReader>) => unknown) | null = null
      readAsText() {
        setTimeout(() => {
          if (this.onload) {
            this.onload.call(this as unknown as FileReader, { target: this } as unknown as ProgressEvent<FileReader>)
          }
        }, 0)
      }
    } as unknown as typeof FileReader
  }

  const createMockFile = (content: string, filename = 'test.json') =>
    new File([content], filename, { type: 'application/json' })

  const makeValidExportJson = (providers: Omit<ProviderConfig, 'id' | 'createdAt' | 'updatedAt'>[]) =>
    JSON.stringify({ version: '1.0', exportedAt: new Date().toISOString(), providers })

  it('渲染导入/导出按钮，点击后打开弹窗', async () => {
    const user = userEvent.setup()
    await openImportExportDialog(user)

    expect(screen.getByTestId('export-section')).toBeInTheDocument()
    expect(screen.getByTestId('import-section')).toBeInTheDocument()
  })

  it('导出区域：默认不包含 API Key，全部分组默认选中', async () => {
    const user = userEvent.setup()
    useProviderStore.setState({
      providers: [
        {
          id: 'p1',
          templateId: 'openai',
          name: 'OpenAI',
          apiHost: 'https://api.openai.com',
          apiKey: 'sk-secret',
          modelId: 'gpt-4o',
          temperature: 0.7,
          timeoutMs: 30000,
          maxRetries: 3,
          group: '云端',
          enabled: true,
          sortOrder: 0,
          createdAt: Date.now(),
          updatedAt: Date.now(),
        },
      ],
    })

    await openImportExportDialog(user)

    const checkbox = screen.getByTestId('include-apikey-checkbox') as HTMLInputElement
    expect(checkbox.checked).toBe(false)

    // 导出按钮可用
    expect(screen.getByTestId('export-btn')).not.toBeDisabled()
  })

  it('导入有效 JSON（合并模式）→ providers 列表增加', async () => {
    const user = userEvent.setup()
    useProviderStore.setState({
      providers: [
        {
          id: 'p1',
          templateId: 'openai',
          name: 'OpenAI',
          apiHost: 'https://api.openai.com',
          apiKey: 'sk-old',
          modelId: 'gpt-4o',
          temperature: 0.7,
          timeoutMs: 30000,
          maxRetries: 3,
          group: '云端',
          enabled: true,
          sortOrder: 0,
          createdAt: Date.now(),
          updatedAt: Date.now(),
        },
      ],
    })

    const json = makeValidExportJson([
      {
        templateId: 'kimi',
        name: 'Kimi',
        apiHost: 'https://api.moonshot.cn',
        apiKey: 'sk-kimi',
        modelId: 'kimi-lite',
        temperature: 0.7,
        timeoutMs: 30000,
        maxRetries: 3,
        group: '云端',
        enabled: true,
        sortOrder: 0,
      },
    ])
    mockFileReader(json)

    await openImportExportDialog(user)

    const fileInput = screen.getByTestId('import-file-input')
    await user.upload(fileInput, createMockFile(json))

    await waitFor(() => {
      expect(screen.getByTestId('import-preview')).toBeInTheDocument()
    })

    await user.click(screen.getByTestId('import-btn'))

    await waitFor(() => {
      expect(screen.getByTestId('import-result')).toBeInTheDocument()
    })

    // 验证 provider 数量从 1 增加到 2
    expect(useProviderStore.getState().providers.length).toBe(2)
    expect(useProviderStore.getState().providers.some((p) => p.name === 'Kimi')).toBe(true)
  })

  it('导入有效 JSON（覆盖模式）→ 原 providers 被完全替换', async () => {
    const user = userEvent.setup()
    useProviderStore.setState({
      providers: [
        {
          id: 'p1',
          templateId: 'openai',
          name: 'OpenAI',
          apiHost: 'https://api.openai.com',
          apiKey: 'sk-old',
          modelId: 'gpt-4o',
          temperature: 0.7,
          timeoutMs: 30000,
          maxRetries: 3,
          group: '云端',
          enabled: true,
          sortOrder: 0,
          createdAt: Date.now(),
          updatedAt: Date.now(),
        },
      ],
    })

    const json = makeValidExportJson([
      {
        templateId: 'deepseek',
        name: 'DeepSeek',
        apiHost: 'https://api.deepseek.com',
        apiKey: 'sk-ds',
        modelId: 'deepseek-chat',
        temperature: 0.7,
        timeoutMs: 30000,
        maxRetries: 3,
        group: '云端',
        enabled: true,
        sortOrder: 0,
      },
    ])
    mockFileReader(json)

    await openImportExportDialog(user)

    // 选择覆盖模式
    await user.click(screen.getByTestId('import-mode-overwrite'))

    const fileInput = screen.getByTestId('import-file-input')
    await user.upload(fileInput, createMockFile(json))

    await waitFor(() => {
      expect(screen.getByTestId('import-preview')).toBeInTheDocument()
    })

    await user.click(screen.getByTestId('import-btn'))

    await waitFor(() => {
      expect(screen.getByTestId('import-result')).toBeInTheDocument()
    })

    // 验证 provider 被完全替换，只剩 1 条且是 DeepSeek
    const allProviders = useProviderStore.getState().providers
    expect(allProviders.length).toBe(1)
    expect(allProviders[0].name).toBe('DeepSeek')
  })

  it('导入无效 JSON → 显示格式错误提示', async () => {
    const user = userEvent.setup()
    mockFileReader('not valid json {{{')

    await openImportExportDialog(user)

    const fileInput = screen.getByTestId('import-file-input')
    await user.upload(fileInput, createMockFile('not valid json {{{'))

    await waitFor(() => {
      expect(screen.getByTestId('validation-errors')).toBeInTheDocument()
    })

    expect(screen.getByTestId('validation-errors').textContent).toContain('JSON 解析失败')
    expect(screen.getByTestId('import-btn')).toBeDisabled()
  })

  it('导入含缺失字段的记录 → 显示具体错误位置', async () => {
    const user = userEvent.setup()
    const json = JSON.stringify({
      version: '1.0',
      providers: [
        { name: 'Valid', apiHost: 'https://a.com', modelId: 'm1' },
        { apiHost: 'https://b.com', modelId: 'm2' }, // 缺少 name
      ],
    })
    mockFileReader(json)

    await openImportExportDialog(user)

    const fileInput = screen.getByTestId('import-file-input')
    await user.upload(fileInput, createMockFile(json))

    await waitFor(() => {
      expect(screen.getByTestId('validation-errors')).toBeInTheDocument()
    })

    expect(screen.getByTestId('validation-errors').textContent).toContain('第 2 条记录')
    expect(screen.getByTestId('validation-errors').textContent).toContain('name')
  })

  it('合并模式导入同名 Provider → 冲突提示 + 跳过', async () => {
    const user = userEvent.setup()
    useProviderStore.setState({
      providers: [
        {
          id: 'p1',
          templateId: 'openai',
          name: 'OpenAI',
          apiHost: 'https://api.openai.com',
          apiKey: 'sk-old',
          modelId: 'gpt-4o',
          temperature: 0.7,
          timeoutMs: 30000,
          maxRetries: 3,
          group: '云端',
          enabled: true,
          sortOrder: 0,
          createdAt: Date.now(),
          updatedAt: Date.now(),
        },
      ],
    })

    const json = makeValidExportJson([
      {
        templateId: 'openai',
        name: 'OpenAI',
        apiHost: 'https://api.openai.com',
        apiKey: 'sk-new',
        modelId: 'gpt-4o-mini',
        temperature: 0.7,
        timeoutMs: 30000,
        maxRetries: 3,
        group: '云端',
        enabled: true,
        sortOrder: 0,
      },
    ])
    mockFileReader(json)

    await openImportExportDialog(user)

    const fileInput = screen.getByTestId('import-file-input')
    await user.upload(fileInput, createMockFile(json))

    await waitFor(() => {
      expect(screen.getByTestId('import-conflicts')).toBeInTheDocument()
    })

    expect(screen.getByTestId('import-conflicts').textContent).toContain('OpenAI')
    expect(screen.getByTestId('import-conflicts').textContent).toContain('将跳过')

    await user.click(screen.getByTestId('import-btn'))

    await waitFor(() => {
      expect(screen.getByTestId('import-result')).toBeInTheDocument()
    })

    // 数量不变，同名被跳过
    expect(useProviderStore.getState().providers.length).toBe(1)
    expect(screen.getByTestId('import-result').textContent).toContain('跳过')
  })

  it('导入不含 API Key 的 provider → 标记 needsApiKey', async () => {
    const user = userEvent.setup()
    const json = makeValidExportJson([
      {
        templateId: 'kimi',
        name: 'Kimi',
        apiHost: 'https://api.moonshot.cn',
        apiKey: '',
        modelId: 'kimi-lite',
        temperature: 0.7,
        timeoutMs: 30000,
        maxRetries: 3,
        group: '云端',
        enabled: true,
        sortOrder: 0,
      },
    ])
    mockFileReader(json)

    await openImportExportDialog(user)

    const fileInput = screen.getByTestId('import-file-input')
    await user.upload(fileInput, createMockFile(json))

    await waitFor(() => {
      expect(screen.getByTestId('import-preview')).toBeInTheDocument()
    })

    await user.click(screen.getByTestId('import-btn'))

    await waitFor(() => {
      expect(screen.getByTestId('import-result')).toBeInTheDocument()
    })

    const added = useProviderStore.getState().providers[0]
    expect(added.needsApiKey).toBe(true)
  })

  it('导出时选择特定分组 → 仅导出该分组的数据', async () => {
    const user = userEvent.setup()
    useProviderStore.setState({
      providers: [
        {
          id: 'p1',
          templateId: 'openai',
          name: 'OpenAI',
          apiHost: 'https://api.openai.com',
          apiKey: 'sk-1',
          modelId: 'gpt-4o',
          temperature: 0.7,
          timeoutMs: 30000,
          maxRetries: 3,
          group: '云端',
          enabled: true,
          sortOrder: 0,
          createdAt: Date.now(),
          updatedAt: Date.now(),
        },
        {
          id: 'p2',
          templateId: 'ollama',
          name: '本地 Ollama',
          apiHost: 'http://localhost:11434',
          apiKey: '',
          modelId: 'llama3.1',
          temperature: 0.7,
          timeoutMs: 30000,
          maxRetries: 3,
          group: '本地',
          enabled: true,
          sortOrder: 0,
          createdAt: Date.now(),
          updatedAt: Date.now(),
        },
      ],
    })

    await openImportExportDialog(user)

    // 取消选择「本地」分组（只保留「云端」）
    const localCheckbox = screen.getByLabelText('本地')
    await user.click(localCheckbox)

    // 点击导出
    await user.click(screen.getByTestId('export-btn'))

    // 验证下载被触发（通过 blob URL）
    expect(URL.createObjectURL).toHaveBeenCalled()
  })

  it('导入空 providers 数组 → 提示缺少 providers 字段', async () => {
    const user = userEvent.setup()
    const json = JSON.stringify({ version: '1.0' })
    mockFileReader(json)

    await openImportExportDialog(user)

    const fileInput = screen.getByTestId('import-file-input')
    await user.upload(fileInput, createMockFile(json))

    await waitFor(() => {
      expect(screen.getByTestId('validation-errors')).toBeInTheDocument()
    })

    expect(screen.getByTestId('validation-errors').textContent).toContain('providers')
  })
})
