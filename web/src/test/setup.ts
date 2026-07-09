import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach, beforeEach, vi } from 'vitest'
import { registerGlobalWailsMock, resetWailsMock } from './mocks/wails'
import { useChatStore } from '@/stores/chatStore'
import { useProviderStore } from '@/stores/providerStore'
import { useSettingsStore } from '@/stores/settingsStore'

// Mock @wails/runtime 模块，使组件导入指向我们的 mock
vi.mock('@wails/runtime', () => import('@/test/mocks/wails'))
vi.mock('@wails/runtime/runtime', () => import('@/test/mocks/wails'))

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
  useSettingsStore.setState({
    activeProviderId: null,
    providerHealthStatus: {},
    lastSelectedProviderId: null,
  })
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

// 模拟 ResizeObserver（Sidebar / react-virtuoso 使用）
// 需在 observe 时立即触发回调，否则虚拟列表无法计算可见区域
global.ResizeObserver = class {
  callback: ResizeObserverCallback

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback
  }

  observe(target: Element) {
    const rect = target.getBoundingClientRect()
    const size: ResizeObserverSize = { inlineSize: rect.width, blockSize: rect.height }
    this.callback(
      [
        {
          target,
          contentRect: rect,
          borderBoxSize: [size],
          contentBoxSize: [size],
          devicePixelContentBoxSize: [size],
        } as ResizeObserverEntry,
      ],
      this
    )
  }

  unobserve = vi.fn()
  disconnect = vi.fn()
} as unknown as typeof ResizeObserver

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

// 模拟 getBoundingClientRect（jsdom 默认返回 0，影响 react-virtuoso 等虚拟列表库）
Object.defineProperty(HTMLElement.prototype, 'getBoundingClientRect', {
  configurable: true,
  value: function (this: HTMLElement) {
    const height = parseFloat(this.style.height) || 768
    const width = parseFloat(this.style.width) || 1024
    return {
      width,
      height,
      top: 0,
      left: 0,
      bottom: height,
      right: width,
      x: 0,
      y: 0,
      toJSON: () => {},
    }
  },
})

// 模拟 offsetHeight / clientHeight / offsetWidth / clientWidth
// react-virtuoso 依赖这些值计算可见区域
function getComputedDim(element: HTMLElement, dim: 'height' | 'width'): number {
  const styleVal = parseFloat(element.style[dim])
  if (!isNaN(styleVal) && styleVal > 0) return styleVal
  const parent = element.parentElement
  if (parent) {
    const parentVal = getComputedDim(parent, dim)
    if (!isNaN(parentVal) && parentVal > 0) return parentVal
  }
  return dim === 'height' ? 768 : 1024
}

Object.defineProperty(HTMLElement.prototype, 'offsetHeight', {
  configurable: true,
  get() { return getComputedDim(this, 'height') },
})
Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
  configurable: true,
  get() { return getComputedDim(this, 'height') },
})
Object.defineProperty(HTMLElement.prototype, 'offsetWidth', {
  configurable: true,
  get() { return getComputedDim(this, 'width') },
})
Object.defineProperty(HTMLElement.prototype, 'clientWidth', {
  configurable: true,
  get() { return getComputedDim(this, 'width') },
})
