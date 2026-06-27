import { describe, it, expect, vi } from 'vitest'
import { render, screen, act } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { ConfidenceBar } from './ConfidenceBar'
import type { ConfidenceResult } from './types'

const mockResult: ConfidenceResult = {
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
  missingInfo: ['疼痛持续时间', '既往病史'],
}

/**
 * ConfidenceBar 追问建议交互测试。
 */
describe('ConfidenceBar follow-up', () => {
  it('点击追问建议会触发 onFollowupClick 回调', async () => {
    const onFollowupClick = vi.fn()
    const user = userEvent.setup()

    render(
      <ConfidenceBar
        result={mockResult}
        mode="compact"
        onFollowupClick={onFollowupClick}
      />
    )

    const button = screen.getByRole('button', { name: /疼痛持续时间/ })
    await user.click(button)

    expect(onFollowupClick).toHaveBeenCalledTimes(1)
    expect(onFollowupClick).toHaveBeenCalledWith('症状持续多久了？')
  })

  it('未知缺失项回退为通用追问文案', async () => {
    const onFollowupClick = vi.fn()
    const user = userEvent.setup()

    render(
      <ConfidenceBar
        result={{ ...mockResult, missingInfo: ['未知项'] }}
        mode="compact"
        onFollowupClick={onFollowupClick}
      />
    )

    const button = screen.getByRole('button', { name: /未知项/ })
    await user.click(button)

    expect(onFollowupClick).toHaveBeenCalledWith('请补充一下未知项的信息')
  })

  it('展开面板的最大高度限制为 300px', async () => {
    const user = userEvent.setup()

    render(<ConfidenceBar result={mockResult} mode="compact" />)

    const bar = screen.getByLabelText(/置信度/)
    await user.click(bar)

    const panel = screen.getByText(/信息不够完整/).closest('.overflow-hidden')
    expect(panel).toHaveClass('max-h-[300px]')
  })
})
