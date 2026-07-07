import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, act } from '@/test/render'
import { ContextUsageBar } from '@/components/chat/ContextUsageBar'
import { useChatStore } from '@/stores/chatStore'
import { resetWailsMock, setMockHandlers } from '@/test/mocks/wails'

const CONV_ID = 'conv-context-usage'

describe('ContextUsageBar', () => {
  beforeEach(() => {
    resetWailsMock()
    setMockHandlers({})
    useChatStore.setState({
      currentConversationId: CONV_ID,
      contextUsageMap: {},
    })
  })

  it('renders animate-pulse skeleton when usage is missing', () => {
    const { container } = render(<ContextUsageBar conversationId={CONV_ID} />)
    // 骨架屏使用 animate-pulse 且隐藏于无障碍树
    expect(container.querySelector('.animate-pulse')).toBeInTheDocument()
  })

  it('renders progressbar with clamped width and correct aria values', () => {
    useChatStore.getState().setContextUsage(CONV_ID, {
      usedTokens: 750,
      maxTokens: 1000,
      ratio: 0.75,
      approximate: false,
      authoritative: true,
    })
    render(<ContextUsageBar conversationId={CONV_ID} />)

    const bar = screen.getByRole('progressbar')
    expect(bar).toHaveAttribute('aria-valuenow', '75')

    const fill = bar.querySelector('[style*="width"]')
    expect(fill).toHaveStyle({ width: '75%' })
  })

  it('shows approximate marker when approximate is true', () => {
    useChatStore.getState().setContextUsage(CONV_ID, {
      usedTokens: 300,
      maxTokens: 1000,
      ratio: 0.3,
      approximate: true,
      authoritative: false,
    })
    render(<ContextUsageBar conversationId={CONV_ID} />)

    expect(screen.getByText('≈')).toBeInTheDocument()
  })

  it('does not show approximate marker when approximate is false', () => {
    useChatStore.getState().setContextUsage(CONV_ID, {
      usedTokens: 300,
      maxTokens: 1000,
      ratio: 0.3,
      approximate: false,
      authoritative: true,
    })
    render(<ContextUsageBar conversationId={CONV_ID} />)

    expect(screen.queryByText('≈')).not.toBeInTheDocument()
  })

  it('re-renders when store context usage changes', () => {
    useChatStore.getState().setContextUsage(CONV_ID, {
      usedTokens: 300,
      maxTokens: 1000,
      ratio: 0.3,
      approximate: false,
      authoritative: true,
    })
    render(<ContextUsageBar conversationId={CONV_ID} />)

    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '30')

    act(() => {
      useChatStore.getState().setContextUsage(CONV_ID, {
        usedTokens: 800,
        maxTokens: 1000,
        ratio: 0.8,
        approximate: false,
        authoritative: true,
      })
    })

    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '80')
  })

  it('switches color class names based on ratio', () => {
    const cases: Array<{ ratio: number; cls: string }> = [
      { ratio: 0.5, cls: 'bg-emerald-500' },
      { ratio: 0.75, cls: 'bg-amber-500' },
      { ratio: 0.9, cls: 'bg-red-500' },
    ]

    for (const { ratio, cls } of cases) {
      const { unmount } = render(
        <ContextUsageBar conversationId={CONV_ID} />
      )
      act(() => {
        useChatStore.getState().setContextUsage(CONV_ID, {
          usedTokens: Math.round(ratio * 1000),
          maxTokens: 1000,
          ratio,
          approximate: false,
          authoritative: true,
        })
      })

      const fill = screen.getByRole('progressbar').querySelector('[style*="width"]')
      expect(fill).toHaveClass(cls)
      unmount()
    }
  })
})
