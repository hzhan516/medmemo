import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { ChatPage } from '@/pages/ChatPage'
import { setMockHandlers, EventsEmit } from '@/test/mocks/wails'
import { useChatStore } from '@/stores/chatStore'

/**
 * 合规流程 E2E 测试。
 * 覆盖 A 级紧急症状弹窗、B 级警告横幅、L1 阻断拦截、L2 警告提示。
 */

describe('E2E: 合规流程', () => {
  it('输入 A 级紧急症状 → 验证全屏红色弹窗 → 点击继续咨询', async () => {
    const user = userEvent.setup()
    render(<ChatPage />)

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '我胸痛伴呼吸困难')
    await user.keyboard('{Enter}')

    // A 级弹窗应立即出现（不发送到 LLM）
    await waitFor(() => {
      expect(screen.getByRole('alertdialog')).toBeInTheDocument()
    })

    // 验证弹窗内容
    expect(screen.getByText('⚠️ 检测到紧急症状')).toBeInTheDocument()
    expect(screen.getByText('检测到 A 级紧急症状，建议立即就医或拨打 120。')).toBeInTheDocument()

    // 验证用户消息未添加到聊天列表
    expect(screen.queryByText('我胸痛伴呼吸困难')).not.toBeInTheDocument()

    // 点击「继续咨询」
    const continueBtn = screen.getByText('继续咨询')
    await user.click(continueBtn)

    // 弹窗消失
    await waitFor(() => {
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    })
  })

  it('输入 B 级紧急症状 → 验证红色警告横幅 → 点击「我已了解」', async () => {
    const user = userEvent.setup()
    render(<ChatPage />)

    // 等待 effect 稳定
    await new Promise((r) => setTimeout(r, 200))

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '我持续高热 3 天了')
    await user.keyboard('{Enter}')

    // B 级警告横幅出现
    await waitFor(() => {
      expect(screen.getByText('我已了解')).toBeInTheDocument()
    })

    // 验证用户消息已添加（B 级不阻断消息添加，只阻断 LLM 调用）
    expect(screen.getAllByText('我持续高热 3 天了').length).toBeGreaterThanOrEqual(1)

    // 点击「我已了解」
    const acknowledgeBtn = screen.getByText('我已了解')
    await user.click(acknowledgeBtn)

    // 验证 store 中紧急症状状态被清除
    await waitFor(() => {
      const state = useChatStore.getState()
      expect(state.emergencyWarningAcknowledged).toBe(true)
    })
  })

  it('输入高风险内容 → 验证 L1 阻断级拦截 → 替换为标准提示语', async () => {
    setMockHandlers({
      SendMessageStream: async () => {
        // 模拟后端返回 L1 阻断级内容后，通过 compliance event 推送
        EventsEmit('chat:stream:token', '我无法提供医疗诊断或治疗建议。')
        EventsEmit('chat:stream:end', null)
        // 流式结束后推送合规事件
        EventsEmit('chat:stream:compliance', {
          level: 'L1_BLOCK',
          warning: '',
          notice: '',
          replacedTerms: ['DIAGNOSIS_TERM'],
          matchedRule: 'L1_DIAGNOSIS_BLOCK',
        })
      },
    })

    const user = userEvent.setup()
    render(<ChatPage />)

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '你确诊我得了什么病')
    await user.keyboard('{Enter}')

    // 验证 L1 阻断提示
    await waitFor(() => {
      expect(screen.getByText('内容已调整为合规表述')).toBeInTheDocument()
    })

    // 验证替换后的内容
    expect(screen.getByText('我无法提供医疗诊断或治疗建议。')).toBeInTheDocument()
  })

  it('输入 L2 警告级内容 → 验证橙色高亮警告框 + 免责声明追加', async () => {
    setMockHandlers({
      SendMessageStream: async () => {
        EventsEmit('chat:stream:token', '这可能是某种健康问题的表现，建议咨询医生确认。')
        EventsEmit('chat:stream:end', null)
        EventsEmit('chat:stream:compliance', {
          level: 'L2_WARNING',
          warning: '此内容涉及健康风险，仅供参考',
          notice: '',
          replacedTerms: [],
          matchedRule: 'L2_IMPLIED_DIAGNOSIS',
        })
      },
    })

    const user = userEvent.setup()
    render(<ChatPage />)

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '我有点胸闷是不是心脏病')
    await user.keyboard('{Enter}')

    // 验证 L2 警告框（橙色）
    await waitFor(() => {
      expect(screen.getByText('内容风险提示')).toBeInTheDocument()
    })

    // 验证警告具体文案
    expect(screen.getByText(/此内容涉及健康风险/)).toBeInTheDocument()
  })
})
