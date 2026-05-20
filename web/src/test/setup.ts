import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach, beforeEach, vi } from 'vitest'
import { registerGlobalWailsMock, resetWailsMock } from './mocks/wails'
import { useChatStore } from '@/stores/chatStore'
import { useProviderStore } from '@/stores/providerStore'
import { useSettingsStore } from '@/stores/settingsStore'

// Mock @wails/runtime 模块，使组件导入指向我们的 mock
vi.mock('@wails/runtime', () => import('@/test/mocks/wails'))

// 注册全局 Wails Mock（window.go.main.WailsApp）
registerGlobalWailsMock()

// 每个测试前清理状态
beforeEach(() => {
  resetWailsMock()
  localStorage.clear()
  cleanup()
  document.documentElement.classList.remove('dark')
  // 重置 Zustand 全局状态
  useChatStore.setState({
    messages: [],
    isStreaming: false,
    currentConversationId: null,
    conversations: [],
    deletedConversations: [],
    searchQuery: '',
    lastDeleted: null,
    showTrash: false,
    dismissedBarSessions: [],
    emergencyAlert: null,
    emergencyWarningAcknowledged: false,
  })
  useProviderStore.setState({ providers: [] })
  useSettingsStore.setState({ activeProviderId: null })
})

// 每个测试后清理
afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

// 模拟 matchMedia（useTheme 中使用）
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

// 模拟 ResizeObserver（Sidebar 等组件使用）
global.ResizeObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}))

// 模拟 IntersectionObserver
global.IntersectionObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}))

// 模拟 window.innerWidth（Sidebar 响应式使用）
Object.defineProperty(window, 'innerWidth', {
  writable: true,
  configurable: true,
  value: 1280,
})

// 模拟 scrollIntoView（jsdom 不支持）
Element.prototype.scrollIntoView = vi.fn()
