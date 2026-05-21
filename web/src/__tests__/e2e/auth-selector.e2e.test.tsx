import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { AuthSelector } from '@/components/auth/AuthSelector'
import { useAuth } from '@/hooks/useAuth'
import { setMockHandlers, resetWailsMock } from '@/test/mocks/wails'

/**
 * 认证方式智能选择 E2E 测试。
 * 覆盖检测流程、推荐展示、面板展开、Provider 创建。
 */

describe('E2E: AuthSelector 认证方式智能选择', () => {
  beforeEach(() => {
    resetWailsMock()
  })

  it('完整流程：检测 → 推荐 OAuth → 展开 API Key 面板 → 保存', async () => {
    const user = userEvent.setup()
    const onProviderCreated = vi.fn()

    setMockHandlers({
      DetectAuthMethods: vi.fn(() =>
        Promise.resolve({
          results: [
            { method: 'cli_token', available: false, connected: false, tier: 1, detail: '未检测到 CLI' },
            { method: 'oauth_device', available: true, connected: false, tier: 2, detail: '支持 OAuth' },
            { method: 'api_key', available: true, connected: false, tier: 3, detail: '可输入 API Key' },
            { method: 'local', available: false, connected: false, tier: 4, detail: '未检测到 Ollama' },
          ],
          recommended: 'oauth_device',
          all_unavailable: false,
        })
      ),
      SaveAPIKey: vi.fn(() => Promise.resolve()),
    })

    function TestComponent() {
      const auth = useAuth()
      return (
        <AuthSelector
          result={auth.result}
          detecting={auth.detecting}
          error={auth.error}
          expandedPanel={auth.expandedPanel}
          ollamaPulling={auth.ollamaPulling}
          ollamaPullProgress={auth.ollamaPullProgress}
          ollamaServerStarting={auth.ollamaServerStarting}
          onDetect={auth.detect}
          onSelectMethod={auth.selectMethod}
          onProviderCreated={onProviderCreated}
        />
      )
    }

    render(<TestComponent />)

    // 步骤 1：初始状态显示开始检测按钮
    expect(screen.getByText('开始检测')).toBeInTheDocument()

    // 步骤 2：点击开始检测
    await user.click(screen.getByText('开始检测'))

    // 步骤 3：等待检测完成，展示推荐横幅
    await waitFor(() => {
      expect(screen.getByText(/为您推荐/)).toBeInTheDocument()
    })

    // 步骤 4：四种认证方式卡片均存在
    expect(screen.getByTestId('auth-card-cli_token')).toBeInTheDocument()
    expect(screen.getByTestId('auth-card-oauth_device')).toBeInTheDocument()
    expect(screen.getByTestId('auth-card-api_key')).toBeInTheDocument()
    expect(screen.getByTestId('auth-card-local')).toBeInTheDocument()

    // 步骤 5：点击 API Key 卡片展开面板
    await user.click(screen.getByTestId('auth-card-api_key'))

    await waitFor(() => {
      expect(screen.getByPlaceholderText(/API Key/)).toBeInTheDocument()
    })
  })

  it('CLI Token 已连接时显示一键使用按钮', async () => {
    const user = userEvent.setup()
    const onProviderCreated = vi.fn()

    setMockHandlers({
      DetectAuthMethods: vi.fn(() =>
        Promise.resolve({
          results: [
            { method: 'cli_token', available: true, connected: true, tier: 1, provider_type: 'kimi', detail: '已检测到 Kimi CLI' },
          ],
          recommended: 'cli_token',
          all_unavailable: false,
        })
      ),
      BuildCLIProvider: vi.fn(() =>
        Promise.resolve({
          id: 'kimi_cli_123',
          templateId: 'kimi',
          name: 'Kimi (CLI)',
          apiHost: 'https://api.moonshot.cn',
          apiKey: '',
          modelId: 'moonshot-v1-8k',
          temperature: 0.7,
          timeoutMs: 30000,
          maxRetries: 3,
          group: 'CLI',
          enabled: true,
          sortOrder: 0,
          createdAt: Date.now(),
          updatedAt: Date.now(),
          authMethod: 'cli_token',
          authParams: {},
        })
      ),
    })

    function TestComponent() {
      const auth = useAuth()
      return (
        <AuthSelector
          result={auth.result}
          detecting={auth.detecting}
          error={auth.error}
          expandedPanel={auth.expandedPanel}
          ollamaPulling={auth.ollamaPulling}
          ollamaPullProgress={auth.ollamaPullProgress}
          ollamaServerStarting={auth.ollamaServerStarting}
          onDetect={auth.detect}
          onSelectMethod={auth.selectMethod}
          onProviderCreated={onProviderCreated}
        />
      )
    }

    render(<TestComponent />)
    await user.click(screen.getByText('开始检测'))

    await waitFor(() => {
      expect(screen.getByText('一键使用 CLI Token')).toBeInTheDocument()
    })

    await user.click(screen.getByText('一键使用 CLI Token'))

    await waitFor(() => {
      expect(onProviderCreated).toHaveBeenCalled()
    })
  })
})
