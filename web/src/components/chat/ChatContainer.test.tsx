import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { ChatContainer } from './ChatContainer'
import type { ChatMessage } from '@/stores/chatStore'
import type { ConfidenceResult } from '@/components/confidence/types'

const mockConfidence: ConfidenceResult = {
  overallScore: 55,
  level: 'C',
  breakdown: {
    knowledge_source: 60,
    reasoning: 50,
    context: 60,
    history: 50,
    uncertainty: 55,
  },
  explanation: '信息不够完整',
  suggestion: '请补充更多信息',
  missingInfo: ['疼痛持续时间'],
}

const assistantMessage: ChatMessage = {
  id: 'msg-1',
  role: 'assistant',
  content: '请补充更多信息',
  timestamp: Date.now(),
  confidence: mockConfidence,
}

/**
 * ChatContainer 追问建议透传测试。
 */
describe('ChatContainer follow-up forwarding', () => {
  it('点击消息置信度追问建议会向上转发 onFollowupClick', async () => {
    const onFollowupClick = vi.fn()
    const user = userEvent.setup()

    render(
      <ChatContainer
        messages={[assistantMessage]}
        isStreaming={false}
        onFollowupClick={onFollowupClick}
      />
    )

    // 展开置信度条以显示追问建议
    const confidenceBar = screen.getByLabelText(/置信度/)
    await user.click(confidenceBar)

    const button = screen.getByRole('button', { name: /疼痛持续时间/ })
    await user.click(button)

    expect(onFollowupClick).toHaveBeenCalledTimes(1)
    expect(onFollowupClick).toHaveBeenCalledWith('症状持续时间（开始时间、持续多久）：')
  })
})
