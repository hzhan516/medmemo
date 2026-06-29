import { describe, it, expect, vi } from 'vitest'
import { render, screen, act } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { MessageBubble } from './MessageBubble'
import type { ChatMessage } from '@/stores/chatStore'
import { setMockHandlers } from '@/test/mocks/wails'

const assistantMessage: ChatMessage = {
  id: 'msg-1',
  role: 'assistant',
  content: '建议您尽快就诊呼吸内科。',
  timestamp: Date.now(),
}

describe('MessageBubble answer feedback', () => {
  it('点击"有帮助"会调用 RecordAnswerFeedback', async () => {
    const recordFeedback = vi.fn().mockResolvedValue(undefined)
    setMockHandlers({ RecordAnswerFeedback: recordFeedback })

    const user = userEvent.setup()
    render(<MessageBubble message={assistantMessage} />)

    const helpfulButton = screen.getByRole('button', { name: /有帮助/ })
    await user.click(helpfulButton)

    await act(async () => {
      await Promise.resolve()
    })

    expect(recordFeedback).toHaveBeenCalledTimes(1)
    expect(recordFeedback).toHaveBeenCalledWith('msg-1', 'recommendation', true)
  })

  it('点击"不准确"会调用 RecordAnswerFeedback', async () => {
    const recordFeedback = vi.fn().mockResolvedValue(undefined)
    setMockHandlers({ RecordAnswerFeedback: recordFeedback })

    const user = userEvent.setup()
    render(<MessageBubble message={assistantMessage} />)

    const inaccurateButton = screen.getByRole('button', { name: /不准确/ })
    await user.click(inaccurateButton)

    await act(async () => {
      await Promise.resolve()
    })

    expect(recordFeedback).toHaveBeenCalledTimes(1)
    expect(recordFeedback).toHaveBeenCalledWith('msg-1', 'recommendation', false)
  })
})
