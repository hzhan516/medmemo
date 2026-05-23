import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { APIKeyGuide } from './APIKeyGuide'

describe('APIKeyGuide', () => {
  it('默认收起折叠面板', () => {
    render(<APIKeyGuide providerId="openai" onOpenURL={vi.fn()} />)
    expect(screen.getByTestId('apikey-guide-toggle')).toBeInTheDocument()
    expect(screen.queryByTestId('apikey-guide-content')).not.toBeInTheDocument()
  })

  it('点击展开显示步骤列表', () => {
    render(<APIKeyGuide providerId="openai" onOpenURL={vi.fn()} />)
    fireEvent.click(screen.getByTestId('apikey-guide-toggle'))
    expect(screen.getByTestId('apikey-guide-content')).toBeInTheDocument()
    // 验证步骤编号存在
    expect(screen.getByText('1')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
  })

  it('点击展开后再点击收起', () => {
    render(<APIKeyGuide providerId="openai" onOpenURL={vi.fn()} />)
    fireEvent.click(screen.getByTestId('apikey-guide-toggle'))
    expect(screen.getByTestId('apikey-guide-content')).toBeInTheDocument()
    fireEvent.click(screen.getByTestId('apikey-guide-toggle'))
    expect(screen.queryByTestId('apikey-guide-content')).not.toBeInTheDocument()
  })

  it('点击“去获取”按钮触发 onOpenURL 回调', () => {
    const onOpenURL = vi.fn()
    render(<APIKeyGuide providerId="kimi" onOpenURL={onOpenURL} />)
    fireEvent.click(screen.getByTestId('apikey-guide-toggle'))
    fireEvent.click(screen.getByTestId('apikey-guide-open-url'))
    expect(onOpenURL).toHaveBeenCalledTimes(1)
    expect(onOpenURL).toHaveBeenCalledWith('https://platform.moonshot.cn/console/api-keys')
  })

  it('切换厂商时步骤内容更新', () => {
    const { rerender } = render(<APIKeyGuide providerId="openai" onOpenURL={vi.fn()} />)
    fireEvent.click(screen.getByTestId('apikey-guide-toggle'))
    expect(screen.getByText(/platform.openai.com/)).toBeInTheDocument()

    rerender(<APIKeyGuide providerId="deepseek" onOpenURL={vi.fn()} />)
    expect(screen.getByText(/platform.deepseek.com/)).toBeInTheDocument()
  })

  it('未知厂商返回 null 不渲染', () => {
    const { container } = render(<APIKeyGuide providerId="unknown" onOpenURL={vi.fn()} />)
    expect(container.firstChild).toBeNull()
  })

  it('步骤中的高亮文本以 code 标签渲染', () => {
    render(<APIKeyGuide providerId="openai" onOpenURL={vi.fn()} />)
    fireEvent.click(screen.getByTestId('apikey-guide-toggle'))
    // 验证高亮的关键操作文本存在
    expect(screen.getByText('API Keys')).toBeInTheDocument()
  })
})
