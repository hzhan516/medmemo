import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { APIKeyPanel } from './APIKeyPanel'
import { setMockHandlers, resetWailsMock } from '@/test/mocks/wails'
import type { AuthMethodDetectStatus } from '@/types/provider'

describe('APIKeyPanel', () => {
  const mockStatus: AuthMethodDetectStatus = {
    method: 'api_key',
    available: true,
    connected: false,
    tier: 3,
    detail: '可手动输入 API Key',
  }

  // 符合 Kimi 格式的测试 Key: sk- + 48 位十六进制
  const validKimiKey = 'sk-1234567890abcdef1234567890abcdef1234567890abcdef'
  // 符合 OpenAI 格式的测试 Key
  const validOpenAIKey = 'sk-proj-1234567890123456789012345678901234567890abcdef'

  beforeEach(() => {
    resetWailsMock()
  })

  it('渲染时包含 APIKeyGuide 组件', () => {
    render(<APIKeyPanel status={mockStatus} onProviderCreated={vi.fn()} />)
    expect(screen.getByTestId('apikey-guide-toggle')).toBeInTheDocument()
  })

  it('展开/折叠引导面板', () => {
    render(<APIKeyPanel status={mockStatus} onProviderCreated={vi.fn()} />)
    expect(screen.queryByTestId('apikey-guide-content')).not.toBeInTheDocument()
    fireEvent.click(screen.getByTestId('apikey-guide-toggle'))
    expect(screen.getByTestId('apikey-guide-content')).toBeInTheDocument()
    fireEvent.click(screen.getByTestId('apikey-guide-toggle'))
    expect(screen.queryByTestId('apikey-guide-content')).not.toBeInTheDocument()
  })

  it('连通性验证成功路径', async () => {
    const onProviderCreated = vi.fn()
    render(<APIKeyPanel status={mockStatus} onProviderCreated={onProviderCreated} />)

    const input = screen.getByPlaceholderText(/Kimi/)
    fireEvent.change(input, { target: { value: validKimiKey } })

    const saveBtn = screen.getByRole('button', { name: /保存并验证/ })
    fireEvent.click(saveBtn)

    await waitFor(() => {
      expect(onProviderCreated).toHaveBeenCalledTimes(1)
    })

    const created = onProviderCreated.mock.calls[0][0]
    expect(created.templateId).toBe('kimi')
    expect(created.authMethod).toBe('api_key')
  })

  it('连通性验证失败时显示错误和“仍然保存”按钮', async () => {
    setMockHandlers({
      TestAPIKey: async () => ({ valid: false, message: 'API Key 无效' }),
    })
    const onProviderCreated = vi.fn()
    render(<APIKeyPanel status={mockStatus} onProviderCreated={onProviderCreated} />)

    const input = screen.getByPlaceholderText(/Kimi/)
    fireEvent.change(input, { target: { value: validKimiKey } })

    const saveBtn = screen.getByRole('button', { name: /保存并验证/ })
    fireEvent.click(saveBtn)

    await waitFor(() => {
      expect(screen.getByText('API Key 验证失败')).toBeInTheDocument()
    })

    // 点击“仍然保存”
    const forceSaveBtn = screen.getByText('仍然保存（跳过验证）')
    fireEvent.click(forceSaveBtn)

    await waitFor(() => {
      expect(onProviderCreated).toHaveBeenCalledTimes(1)
    })
  })

  it('验证中显示 loading 状态', async () => {
    setMockHandlers({
      TestAPIKey: () => new Promise((resolve) => setTimeout(() => resolve({ valid: true, message: 'ok' }), 100)),
    })
    render(<APIKeyPanel status={mockStatus} onProviderCreated={vi.fn()} />)

    const input = screen.getByPlaceholderText(/Kimi/)
    fireEvent.change(input, { target: { value: validKimiKey } })

    const saveBtn = screen.getByRole('button', { name: /保存并验证/ })
    fireEvent.click(saveBtn)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /验证中/ })).toBeInTheDocument()
    })
  })

  it('空 API Key 时保存按钮被禁用', () => {
    render(<APIKeyPanel status={mockStatus} onProviderCreated={vi.fn()} />)
    const saveBtn = screen.getByRole('button', { name: /保存并验证/ })
    expect(saveBtn).toBeDisabled()
  })

  it('前缀格式不正确时阻止保存', async () => {
    render(<APIKeyPanel status={mockStatus} onProviderCreated={vi.fn()} />)
    const input = screen.getByPlaceholderText(/Kimi/)
    fireEvent.change(input, { target: { value: 'invalid-key-format' } })

    const saveBtn = screen.getByRole('button', { name: /保存并验证/ })
    fireEvent.click(saveBtn)

    await waitFor(() => {
      expect(screen.getByText('API Key 前缀格式不正确')).toBeInTheDocument()
    })
  })

  it('切换厂商时重置验证状态和输入', () => {
    render(<APIKeyPanel status={mockStatus} onProviderCreated={vi.fn()} />)
    const input = screen.getByPlaceholderText(/Kimi/)
    fireEvent.change(input, { target: { value: 'sk-test' } })
    expect(input).toHaveValue('sk-test')

    const select = screen.getByRole('combobox')
    fireEvent.change(select, { target: { value: 'openai' } })

    expect(screen.getByPlaceholderText(/OpenAI/)).toHaveValue('')
  })

  it('智能粘贴：切换厂商时自动匹配其他厂商 Key 格式', async () => {
    // 模拟剪贴板中有 OpenAI 格式的 Key
    Object.defineProperty(navigator, 'clipboard', {
      value: {
        readText: vi.fn().mockResolvedValue(validOpenAIKey),
      },
      writable: true,
      configurable: true,
    })

    render(<APIKeyPanel status={mockStatus} onProviderCreated={vi.fn()} />)

    // 触发 focus 事件
    fireEvent.focus(window)

    await waitFor(() => {
      const input = screen.getByPlaceholderText(/OpenAI/) as HTMLInputElement
      expect(input.value).toBe(validOpenAIKey)
    }, { timeout: 500 })
  })
})
