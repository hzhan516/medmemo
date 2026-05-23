import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { ChatPage } from '@/pages/ChatPage'
import { setMockHandlers, EventsEmit, resetWailsMock } from '@/test/mocks/wails'
import { useChatStore } from '@/stores/chatStore'

/**
 * E2E: 流式生成中切换会话，验证消息保留与恢复。
 */

describe('E2E: 流式生成中切换会话', () => {
  beforeEach(() => {
    resetWailsMock()
  })

  it('streaming 中切换会话再回来 → 用户消息和 thinking indicator 仍然可见', async () => {
    let resolveStream: (() => void) | null = null
    const streamPromise = new Promise<void>((r) => {
      resolveStream = r
    })

    // 模拟一个挂起的流式响应（未完成），且后端查询返回空（模拟用户消息尚未被后端保存）
    setMockHandlers({
      SendMessageStream: async (req: { conversation_id: string }) => {
        const convId = req.conversation_id
        EventsEmit('chat:stream_chunk', { type: 'start', payload: '', metadata: { conversation_id: convId } })
        EventsEmit('chat:stream_chunk', { type: 'content', payload: '这是', metadata: { conversation_id: convId } })
        // 挂起，模拟 AI 仍在生成中
        await streamPromise
        EventsEmit('chat:stream_chunk', { type: 'done', payload: '', metadata: { conversation_id: convId } })
      },
      // 关键：模拟后端尚未保存消息，验证前端依赖本地缓存而非后端数据
      GetConversationMessages: async () => [],
    })

    const user = userEvent.setup()
    render(<ChatPage />)

    // 发送消息
    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '切换会话测试')
    await user.keyboard('{Enter}')

    // 等待用户消息出现
    await waitFor(() => {
      expect(screen.queryAllByText('切换会话测试').length).toBeGreaterThanOrEqual(1)
    })

    // 等待 thinking indicator 出现（停止生成按钮可见）
    await waitFor(() => {
      expect(screen.getByLabelText('停止生成')).toBeInTheDocument()
    })

    // 验证 store 中当前会话正在 streaming
    expect(useChatStore.getState().isStreaming).toBe(true)

    // 新建会话 B
    const newConvBtn = screen.getByTitle('新建会话 (Ctrl+N)')
    await user.click(newConvBtn)

    // 验证会话 B 是空状态（欢迎页）
    await waitFor(() => {
      expect(screen.getByText('MedMemo 健康助手')).toBeInTheDocument()
    })

    // 验证会话 B 不在 streaming
    expect(useChatStore.getState().isStreaming).toBe(false)

    // 点击会话 A 回到会话 A（侧边栏和消息区都有相同文本，取第一个侧边栏项）
    const convAItems = screen.getAllByText('切换会话测试')
    await user.click(convAItems[0])

    // 验证用户消息仍然可见（关键断言：本地缓存未被后端空数据覆盖）
    await waitFor(() => {
      expect(screen.queryAllByText('切换会话测试').length).toBeGreaterThanOrEqual(1)
    })

    // 验证 thinking indicator 仍然显示（停止生成按钮可见）
    expect(screen.getByLabelText('停止生成')).toBeInTheDocument()

    // 验证 store 中恢复了 streaming 状态
    expect(useChatStore.getState().isStreaming).toBe(true)

    // 释放流式响应，避免测试挂起
    if (resolveStream) {
      resolveStream()
    }

    // 等待流式结束
    await waitFor(() => {
      expect(useChatStore.getState().isStreaming).toBe(false)
    })
  })

  it('逐 chunk 透传的流式回复 → 前端 token 逐字追加', async () => {
    setMockHandlers({
      SendMessageStream: async (req: { conversation_id: string }) => {
        const convId = req.conversation_id
        EventsEmit('chat:stream_chunk', { type: 'start', payload: '', metadata: { conversation_id: convId } })
        const chunks = ['流', '式', '测', '试', '成', '功']
        for (const chunk of chunks) {
          await new Promise((r) => setTimeout(r, 5))
          EventsEmit('chat:stream_chunk', { type: 'content', payload: chunk, metadata: { conversation_id: convId } })
        }
        EventsEmit('chat:stream_chunk', { type: 'done', payload: '', metadata: { conversation_id: convId, latency_ms: 50 } })
      },
    })

    const user = userEvent.setup()
    render(<ChatPage />)

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '测试逐字输出')
    await user.keyboard('{Enter}')

    // 等待流式结束
    await waitFor(() => {
      const state = useChatStore.getState()
      const lastAssistant = state.messages.filter((m) => m.role === 'assistant').pop()
      expect(lastAssistant?.content).toBe('流式测试成功')
    })

    // 验证最终内容在 DOM 中
    expect(screen.queryAllByText('流式测试成功').length).toBeGreaterThanOrEqual(1)
  })

  it('chat:stream:replace 事件 → 前端替换最后一条 assistant 消息', async () => {
    setMockHandlers({
      SendMessageStream: async (req: { conversation_id: string }) => {
        const convId = req.conversation_id
        EventsEmit('chat:stream_chunk', { type: 'start', payload: '', metadata: { conversation_id: convId } })
        // 先推送原始内容（模拟脱敏前的占位符文本）
        EventsEmit('chat:stream_chunk', { type: 'content', payload: '原始占位符内容', metadata: { conversation_id: convId } })
        // 然后发送 replace 事件（模拟脱敏还原后的真实内容）
        EventsEmit('chat:stream:replace', {
          conversation_id: convId,
          content: '还原后的真实内容',
        })
        EventsEmit('chat:stream_chunk', { type: 'done', payload: '', metadata: { conversation_id: convId } })
      },
    })

    const user = userEvent.setup()
    render(<ChatPage />)

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '测试 replace')
    await user.keyboard('{Enter}')

    // 等待 replace 生效后的最终内容
    await waitFor(() => {
      const state = useChatStore.getState()
      const lastAssistant = state.messages.filter((m) => m.role === 'assistant').pop()
      expect(lastAssistant?.content).toBe('还原后的真实内容')
    })

    // 验证替换后的内容在 DOM 中，原始内容不在
    expect(screen.queryAllByText('还原后的真实内容').length).toBeGreaterThanOrEqual(1)
    expect(screen.queryByText('原始占位符内容')).not.toBeInTheDocument()
  })
})
