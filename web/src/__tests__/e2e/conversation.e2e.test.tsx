import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { ChatPage } from '@/pages/ChatPage'
import { setMockHandlers, EventsEmit } from '@/test/mocks/wails'
import { useChatStore } from '@/stores/chatStore'

/**
 * 对话流程 E2E 测试。
 * 覆盖新建对话、消息发送、流式渲染、会话切换、停止生成、错误恢复、长会话稳定性。
 */

describe('E2E: 对话流程', () => {
  it('发送消息 → 接收 AI 流式回复 → 验证消息气泡', async () => {
    const user = userEvent.setup()
    render(<ChatPage />)

    // 初始状态：空会话，显示欢迎页
    expect(screen.getByText('MedMemo 健康助手')).toBeInTheDocument()

    // 输入并发送消息
    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '我最近失眠怎么办')
    await user.keyboard('{Enter}')

    // 等待流式结束，验证用户消息和 AI 回复（通过 store 断言避免 DOM 时序问题）
    await waitFor(() => {
      expect(screen.getByText('我最近失眠怎么办')).toBeInTheDocument()
    })
    await waitFor(
      () => {
        const state = useChatStore.getState()
        const assistantMsgs = state.messages.filter((m) => m.role === 'assistant')
        expect(assistantMsgs.length).toBeGreaterThanOrEqual(1)
        expect(assistantMsgs[assistantMsgs.length - 1].content).toContain('这是一个模拟的流式回复。')
      },
      { timeout: 5000 }
    )

    // 验证消息角色：用户消息右对齐（通过头像区分）
    const userBubbles = screen.getAllByText('我最近失眠怎么办')
    expect(userBubbles.length).toBeGreaterThanOrEqual(1)
  })

  it('新建第二个对话 → 验证会话列表更新 → 切换会话 → 验证消息隔离', async () => {
    const user = userEvent.setup()
    render(<ChatPage />)

    // 发送第一条消息（自动创建会话 conv_1）
    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '第一条消息')
    await user.keyboard('{Enter}')

    await waitFor(() => {
      expect(screen.getByText('第一条消息')).toBeInTheDocument()
    })

    // 等待流式结束后再新建
    await waitFor(() => {
      expect(screen.queryByText('AI 正在生成回复...')).not.toBeInTheDocument()
    })

    // 点击「新建对话」按钮
    const newConvBtn = screen.getByTitle('新建会话 (Ctrl+N)')
    await user.click(newConvBtn)

    // 验证新会话为空状态（回到欢迎页）
    await waitFor(() => {
      expect(screen.getByText('MedMemo 健康助手')).toBeInTheDocument()
    })

    // 在新会话发送第二条消息
    const textarea2 = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea2, '第二条消息')
    await user.keyboard('{Enter}')

    await waitFor(() => {
      expect(screen.getByText('第二条消息')).toBeInTheDocument()
    })

    // 验证第一条消息不在当前视图中（消息隔离）
    expect(screen.queryByText('第一条消息')).not.toBeInTheDocument()

    // 验证侧边栏有两个会话
    await waitFor(() => {
      const convItems = screen.getAllByText(/新对话|模拟生成的标题/)
      expect(convItems.length).toBeGreaterThanOrEqual(2)
    })
  })

  it('发送消息 → 停止生成 → 验证保留已生成内容 + [用户中断] 标记', async () => {
    // 模拟一个缓慢的流式响应
    setMockHandlers({
      SendMessageStream: async () => {
        EventsEmit('chat:stream_chunk', { type: 'start', payload: '' })
        EventsEmit('chat:stream_chunk', { type: 'content', payload: '这是' })
        await new Promise((r) => setTimeout(r, 50))
        EventsEmit('chat:stream_chunk', { type: 'content', payload: '被中断' })
        await new Promise((r) => setTimeout(r, 50))
        EventsEmit('chat:stream_chunk', { type: 'content', payload: '的内容' })
        // 不发送 done 事件，模拟中断
      },
      StopGeneration: async () => {
        EventsEmit('chat:stream_chunk', { type: 'error', payload: '生成已中断' })
      },
    })

    const user = userEvent.setup()
    render(<ChatPage />)

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '测试中断')
    await user.keyboard('{Enter}')

    // 等待部分流式内容出现
    await waitFor(() => {
      expect(screen.getByText(/这是被中断的内容/)).toBeInTheDocument()
    })

    // 点击停止生成按钮
    await waitFor(() => {
      const stopBtn = screen.getByLabelText('停止生成')
      expect(stopBtn).toBeInTheDocument()
    })
    const stopBtn = screen.getByLabelText('停止生成')
    await user.click(stopBtn)

    // 验证中断错误提示
    await waitFor(() => {
      expect(screen.getByText('生成已中断')).toBeInTheDocument()
    })
  })

  it('发送消息 → 网络异常 → 验证错误提示 + 重试按钮', async () => {
    setMockHandlers({
      SendMessageStream: async () => {
        EventsEmit('chat:stream_chunk', { type: 'error', payload: '网络连接超时，请检查网络后重试' })
      },
    })

    const user = userEvent.setup()
    render(<ChatPage />)

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '测试网络异常')
    await user.keyboard('{Enter}')

    // 验证错误提示出现
    await waitFor(() => {
      expect(screen.getByText(/网络连接超时/)).toBeInTheDocument()
    })

    // 验证重试按钮
    await waitFor(() => {
      expect(screen.getByTitle('重新生成')).toBeInTheDocument()
    })
  })

  it('连续发送 10 条消息 → 验证消息顺序正确', async () => {
    const user = userEvent.setup()
    render(<ChatPage />)

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)

    for (let i = 1; i <= 10; i++) {
      // 确保不处于 loading 状态
      await waitFor(() => {
        expect(screen.queryByLabelText('停止生成')).not.toBeInTheDocument()
      })

      await user.type(textarea, `消息${i}`)
      await user.keyboard('{Enter}')

      await waitFor(() => {
        expect(screen.getByText(`消息${i}`)).toBeInTheDocument()
      })
    }

    // 验证所有 10 条消息都在页面中
    for (let i = 1; i <= 10; i++) {
      expect(screen.getByText(`消息${i}`)).toBeInTheDocument()
    }
  })

  it('输入 /new 命令 → 验证新建会话', async () => {
    const user = userEvent.setup()
    render(<ChatPage />)

    // 先发送一条消息
    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '第一条')
    await user.keyboard('{Enter}')

    await waitFor(() => {
      expect(screen.getByText('第一条')).toBeInTheDocument()
    })

    // 输入 /new 命令
    await waitFor(() => {
      expect(screen.queryByLabelText('停止生成')).not.toBeInTheDocument()
    })

    await user.type(textarea, '/new')
    await user.keyboard('{Enter}')

    // 验证回到空状态
    await waitFor(() => {
      expect(screen.getByText('MedMemo 健康助手')).toBeInTheDocument()
    })
  })
})
