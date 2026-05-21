import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@/test/render'
import { fireEvent } from '@testing-library/react'
import { AuthSelector } from './AuthSelector'
import { resetWailsMock } from '@/test/mocks/wails'
import type { AuthDetectResult, AuthPanel } from '@/types/provider'

function createMockResult(overrides?: Partial<AuthDetectResult>): AuthDetectResult {
  return {
    results: [
      { method: 'cli_token', available: false, connected: false, tier: 1, detail: '未检测' },
      { method: 'oauth_device', available: true, connected: false, tier: 2, detail: '可用' },
      { method: 'api_key', available: true, connected: false, tier: 3, detail: '可用' },
      { method: 'local', available: false, connected: false, tier: 4, detail: '未检测' },
    ],
    recommended: 'oauth_device',
    all_unavailable: false,
    ...overrides,
  }
}

describe('AuthSelector', () => {
  beforeEach(() => {
    resetWailsMock()
  })

  const defaultProps = {
    result: null as AuthDetectResult | null,
    detecting: false,
    error: null as string | null,
    expandedPanel: null as AuthPanel | null,
    ollamaPulling: false,
    ollamaPullProgress: '',
    ollamaServerStarting: false,
    onDetect: vi.fn(),
    onSelectMethod: vi.fn(),
    onProviderCreated: vi.fn(),
  }

  it('未检测时显示开始检测按钮', () => {
    render(<AuthSelector {...defaultProps} />)
    expect(screen.getByText('开始检测')).toBeInTheDocument()
  })

  it('检测中显示加载状态', () => {
    render(<AuthSelector {...defaultProps} detecting />)
    expect(screen.getByText('正在检测认证环境...')).toBeInTheDocument()
  })

  it('检测错误时显示错误信息', () => {
    render(<AuthSelector {...defaultProps} error="检测超时" />)
    expect(screen.getByText('检测超时')).toBeInTheDocument()
    expect(screen.getByText('重新检测')).toBeInTheDocument()
  })

  it('检测结果展示四种认证方式卡片', () => {
    render(<AuthSelector {...defaultProps} result={createMockResult()} />)
    expect(screen.getByTestId('auth-card-cli_token')).toBeInTheDocument()
    expect(screen.getByTestId('auth-card-oauth_device')).toBeInTheDocument()
    expect(screen.getByTestId('auth-card-api_key')).toBeInTheDocument()
    expect(screen.getByTestId('auth-card-local')).toBeInTheDocument()
  })

  it('推荐项显示推荐标签', () => {
    render(<AuthSelector {...defaultProps} result={createMockResult()} />)
    expect(screen.getByText('推荐')).toBeInTheDocument()
  })

  it('点击卡片展开配置面板', () => {
    const onSelectMethod = vi.fn()
    render(<AuthSelector {...defaultProps} result={createMockResult()} onSelectMethod={onSelectMethod} />)

    fireEvent.click(screen.getByTestId('auth-card-api_key'))
    expect(onSelectMethod).toHaveBeenCalledWith('api_key')
  })

  it('全部不可用时显示提示', () => {
    render(
      <AuthSelector
        {...defaultProps}
        result={createMockResult({ all_unavailable: true, recommended: 'local' })}
      />
    )
    expect(screen.getByText(/未检测到可用的认证方式/)).toBeInTheDocument()
  })

  it('已连接的状态显示绿色对勾', () => {
    const result = createMockResult({
      results: [
        { method: 'api_key', available: true, connected: true, tier: 3, detail: '已配置' },
      ],
      recommended: 'api_key',
    })
    render(<AuthSelector {...defaultProps} result={result} />)
    expect(screen.getByText('已连接')).toBeInTheDocument()
  })
})
