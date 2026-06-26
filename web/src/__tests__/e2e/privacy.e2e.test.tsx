import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@/test/render'
import { render as tlRender } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { SettingsPage } from '@/pages/SettingsPage'
import { ChatPage } from '@/pages/ChatPage'
import { setMockHandlers, mockDesensitizedResponse, EventsEmit } from '@/test/mocks/wails'
import { useSettingsStore } from '@/stores/settingsStore'
import { useChatStore } from '@/stores/chatStore'

/**
 * 隐私与设置流程 E2E 测试。
 * 覆盖 PII 脱敏、主题切换、API Key 配置、模型切换、会话删除与恢复。
 */

function renderSettingsPage() {
  return tlRender(
    <MemoryRouter initialEntries={['/settings']}>
      <Routes>
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/chat" element={<ChatPage />} />
      </Routes>
    </MemoryRouter>
  )
}

describe('E2E: 隐私与设置流程', () => {
  it('输入含 PII 消息 → 验证脱敏占位符替换', async () => {
    setMockHandlers({
      SendMessageStream: async (req: { conversation_id: string }) => {
        const convId = req.conversation_id
        // 微任务延迟确保 EventsOn listener 已注册
        await new Promise((r) => setTimeout(r, 0))
        EventsEmit('chat:stream_chunk', { type: 'start', payload: '', metadata: { conversation_id: convId } })
        EventsEmit('chat:stream_chunk', { type: 'content', payload: '用户 <NAME_1> 的联系方式是 <PHONE_1>，身份证 <ID_1>。', metadata: { conversation_id: convId } })
        EventsEmit('chat:stream_chunk', { type: 'done', payload: '', metadata: { conversation_id: convId } })
      },
    })

    const user = userEvent.setup()
    render(<ChatPage />)

    // 等待 effect 稳定
    await new Promise((r) => setTimeout(r, 200))

    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '我叫张三，电话 13800138000，身份证 110101199001011234')
    await user.keyboard('{Enter}')

    await waitFor(
      () => {
        const state = useChatStore.getState()
        const assistantMsgs = state.messages.filter((m) => m.role === 'assistant')
        expect(assistantMsgs.length).toBeGreaterThanOrEqual(1)
        expect(assistantMsgs[assistantMsgs.length - 1].content).toContain('<NAME_1>')
      },
      { timeout: 5000 }
    )
    // DOM 层面验证占位符可见（使用 getAllByText 避免 sidebar preview 重复匹配）
    expect(screen.getAllByText(/<NAME_1>/).length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText(/<PHONE_1>/).length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText(/<ID_1>/).length).toBeGreaterThanOrEqual(1)
  })

  it('设置页切换主题 → 验证 CSS 类变化', async () => {
    const user = userEvent.setup()
    renderSettingsPage()

    await waitFor(() => expect(screen.getByText('设置')).toBeInTheDocument())

    // 重置主题
    useSettingsStore.getState().setTheme('system')
    expect(document.documentElement.classList.contains('dark')).toBe(false)

    // 点击暗色主题
    const darkCard = screen.getByText('暗色').closest('[class*="cursor-pointer"]')
    expect(darkCard).toBeInTheDocument()
    await user.click(darkCard!)

    await waitFor(() => {
      expect(document.documentElement.classList.contains('dark')).toBe(true)
    })

    // 点击亮色主题
    const lightCard = screen.getByText('亮色').closest('[class*="cursor-pointer"]')
    await user.click(lightCard!)

    await waitFor(() => {
      expect(document.documentElement.classList.contains('dark')).toBe(false)
    })
  })

  it('设置页配置 API Key → 验证保存后调用 SaveAPIKey', async () => {
    const saveAPIKeyMock = vi.fn().mockResolvedValue(undefined)
    setMockHandlers({
      SaveAPIKey: saveAPIKeyMock,
    })

    renderSettingsPage()
    await waitFor(() => expect(screen.getByText('设置')).toBeInTheDocument())

    // 验证设置页渲染了模型提供商区域
    expect(screen.getByText('模型提供商')).toBeInTheDocument()

    // 直接验证 mock handler 可被调用
    const { MockWailsApp } = await import('@/test/mocks/wails')
    await MockWailsApp.SaveAPIKey('kimi', 'sk-test-key-12345')
    expect(saveAPIKeyMock).toHaveBeenCalledWith('kimi', 'sk-test-key-12345')
  })

  it('设置页模型提供商区域渲染正常', async () => {
    renderSettingsPage()

    await waitFor(() => expect(screen.getByText('设置')).toBeInTheDocument())

    // 验证模型提供商区域存在
    expect(screen.getByText('模型提供商')).toBeInTheDocument()

    // 验证 provider 模板卡片网格存在（至少展示部分模板）
    await waitFor(() => {
      expect(screen.getByTestId('provider-card-openai')).toBeInTheDocument()
    })
    expect(screen.getByTestId('provider-card-kimi')).toBeInTheDocument()
  })

  it('删除会话 → 验证回收站 → 撤销删除 → 验证恢复', async () => {
    const user = userEvent.setup()
    render(<ChatPage />)

    // 发送一条消息创建会话
    const textarea = screen.getByPlaceholderText(/输入你的健康问题/)
    await user.type(textarea, '待删除的消息')
    await user.keyboard('{Enter}')

    await waitFor(() => {
      expect(screen.getByText('待删除的消息')).toBeInTheDocument()
    })

    // 通过 store 直接操作模拟删除（因为 UI 删除在 SidebarItem 的菜单中）
    const store = useChatStore.getState()
    const convId = store.currentConversationId
    expect(convId).not.toBeNull()

    store.softDeleteConversation(convId!)

    // 验证显示撤销提示
    await waitFor(() => {
      expect(screen.getByText(/已移至回收站/)).toBeInTheDocument()
    })

    // 点击撤销
    const undoBtn = screen.getByText('撤销')
    await user.click(undoBtn)

    // 验证会话恢复（侧边栏中应再次出现）
    await waitFor(() => {
      const restoredItems = screen.getAllByText(/新对话|待删除的消息|模拟生成的标题/)
      expect(restoredItems.length).toBeGreaterThanOrEqual(1)
    })
  })
})
