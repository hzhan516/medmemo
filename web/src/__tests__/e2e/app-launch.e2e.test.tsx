import { describe, it, expect, vi } from 'vitest'
import { render as tlRender, screen, waitFor } from '@testing-library/react'
import App from '@/App'
import { setMockHandlers } from '@/test/mocks/wails'
import { useChatStore } from '@/stores/chatStore'

/**
 * App 启动加载 E2E 测试。
 * 验证应用重启后自动从后端加载历史对话列表并恢复消息。
 */

describe('E2E: 应用启动加载持久化对话', () => {
  it('启动时后端有历史对话 → 侧边栏渲染列表并自动选中最近对话', async () => {
    setMockHandlers({
      GetDisclaimerStatus: async () => ({
        required: false,
        text: '本产品提供的信息仅供参考，不构成医疗诊断或治疗建议。',
        version: '1.0.0',
      }),
      GetConversations: async () => [
        {
          id: 'conv_old_2',
          title: '头痛记录',
          updated_at: String(Date.now()),
        },
        {
          id: 'conv_old_1',
          title: '失眠咨询',
          updated_at: String(Date.now() - 3600000),
        },
      ],
      GetConversationMessages: async (convID: string) => {
        if (convID === 'conv_old_2') {
          return [
            {
              id: 'msg_1',
              role: 'user',
              content: '我最近经常头痛',
              timestamp: String(Date.now() - 60000),
            },
            {
              id: 'msg_2',
              role: 'assistant',
              content: '头痛可能由多种原因引起，建议咨询医生确认。',
              timestamp: String(Date.now()),
            },
          ]
        }
        return []
      },
    })

    // 使用原始 render，避免与 App 内部的 HashRouter 嵌套
    tlRender(<App />)

    // 验证侧边栏渲染了两个历史对话
    await waitFor(() => {
      expect(screen.getByText('失眠咨询')).toBeInTheDocument()
      expect(screen.getByText('头痛记录')).toBeInTheDocument()
    })

    // 先验证 store 状态已正确加载
    await waitFor(() => {
      const state = useChatStore.getState()
      expect(state.conversations.length).toBe(2)
      expect(state.currentConversationId).toBe('conv_old_2')
      expect(state.messages.length).toBe(2)
    })

    // 验证最近对话（conv_old_2）的消息被自动加载到聊天区域
    await waitFor(() => {
      expect(screen.getByText('我最近经常头痛')).toBeInTheDocument()
      expect(
        screen.getByText('头痛可能由多种原因引起，建议咨询医生确认。')
      ).toBeInTheDocument()
    }, { timeout: 3000 })
  })

  it('启动时后端无历史对话 → 显示空状态欢迎页', async () => {
    setMockHandlers({
      GetDisclaimerStatus: async () => ({
        required: false,
        text: '免责声明文本',
        version: '1.0.0',
      }),
      GetConversations: async () => [],
    })

    tlRender(<App />)

    await waitFor(() => {
      expect(screen.getByText('健康信息助手')).toBeInTheDocument()
    })
  })
})
