import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { ChatInput } from './ChatInput'

/**
 * ChatInput 组件单元测试。
 * 覆盖快捷键行为：Enter 发送、Shift+Enter 换行、Escape 清空、ArrowUp 编辑上一条。
 */

describe('ChatInput', () => {
  it('空输入框时按 ArrowUp 可填充最后一条用户消息', async () => {
    const onSend = vi.fn()
    const user = userEvent.setup()

    render(
      <ChatInput
        onSend={onSend}
        lastUserMessage="我最近失眠怎么办"
      />
    )

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    expect(textarea).toHaveValue('')

    await user.click(textarea)
    await user.keyboard('{ArrowUp}')

    expect(textarea).toHaveValue('我最近失眠怎么办')
  })

  it('非空输入框时按 ArrowUp 不替换内容', async () => {
    const onSend = vi.fn()
    const user = userEvent.setup()

    render(
      <ChatInput
        onSend={onSend}
        lastUserMessage="我最近失眠怎么办"
      />
    )

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '部分输入')
    expect(textarea).toHaveValue('部分输入')

    await user.keyboard('{ArrowUp}')
    expect(textarea).toHaveValue('部分输入')
  })

  it('无 lastUserMessage 时按 ArrowUp 不变化', async () => {
    const onSend = vi.fn()
    const user = userEvent.setup()

    render(<ChatInput onSend={onSend} />)

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.click(textarea)
    await user.keyboard('{ArrowUp}')

    expect(textarea).toHaveValue('')
  })

  it('Enter 发送消息并清空输入框', async () => {
    const onSend = vi.fn()
    const user = userEvent.setup()

    render(<ChatInput onSend={onSend} />)

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '测试消息')
    await user.keyboard('{Enter}')

    expect(onSend).toHaveBeenCalledWith('测试消息')
    expect(textarea).toHaveValue('')
  })

  it('Shift+Enter 插入换行符而不发送', async () => {
    const onSend = vi.fn()
    const user = userEvent.setup()

    render(<ChatInput onSend={onSend} />)

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '第一行')
    await user.keyboard('{Shift>}{Enter}{/Shift}')

    expect(onSend).not.toHaveBeenCalled()
    expect(textarea).toHaveValue('第一行\n')
  })

  it('Escape 清空输入框', async () => {
    const onSend = vi.fn()
    const user = userEvent.setup()

    render(<ChatInput onSend={onSend} />)

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '待清空内容')
    await user.keyboard('{Escape}')

    expect(textarea).toHaveValue('')
  })
})
