import { useState, useCallback } from 'react'
import { ChatContainer } from '@/components/chat/ChatContainer'
import { ChatInput } from '@/components/chat/ChatInput'
import { Sidebar } from '@/components/layout/Sidebar'
import { Header } from '@/components/layout/Header'
import { useConversation } from '@/hooks/useConversation'

/**
 * 主对话页面，包含侧边栏、顶部栏、聊天区域和输入区。
 */
export function ChatPage() {
  const {
    messages,
    isStreaming,
    currentConversationId,
    sendMessage,
    stopGeneration,
    retryMessage,
    startNewConversation,
    error,
  } = useConversation()

  // 会话列表（占位，后续对接后端 API）
  const [conversations, setConversations] = useState<
    Array<{ id: string; title: string; preview?: string; timestamp?: string; unread?: number }>
  >([
    {
      id: 'demo-1',
      title: '头痛和发热咨询',
      preview: '最近三天一直头痛，伴有低烧...',
      timestamp: '10:23',
      unread: 2,
    },
    {
      id: 'demo-2',
      title: '家族糖尿病史',
      preview: '我父亲和祖父都有糖尿病...',
      timestamp: '昨天',
      unread: 0,
    },
  ])

  const handleNewConversation = useCallback(async () => {
    await startNewConversation()
    // 前端添加一个新会话项
    const newConv = {
      id: `conv_${Date.now()}`,
      title: '新对话',
    }
    setConversations((prev) => [newConv, ...prev])
  }, [startNewConversation])

  const handleRename = useCallback((id: string, title: string) => {
    setConversations((prev) =>
      prev.map((c) => (c.id === id ? { ...c, title } : c))
    )
  }, [])

  const handleDelete = useCallback((id: string) => {
    setConversations((prev) => prev.filter((c) => c.id !== id))
  }, [])

  return (
    <div className="flex h-full">
      <Sidebar
        conversations={conversations}
        activeId={currentConversationId ?? undefined}
        onSelect={(id) => console.log('select conversation', id)}
        onNewConversation={handleNewConversation}
        onRename={handleRename}
        onDelete={handleDelete}
      />

      <div className="flex-1 flex flex-col min-w-0">
        <Header />

        <ChatContainer
          messages={messages}
          isStreaming={isStreaming}
          onStartNewConversation={handleNewConversation}
          onRetry={retryMessage}
        />

        {error && (
          <div className="shrink-0 px-4 py-2 bg-destructive/10 text-destructive text-xs text-center">
            {error}
          </div>
        )}

        <ChatInput
          onSend={sendMessage}
          onStop={stopGeneration}
          isLoading={isStreaming}
        />
      </div>
    </div>
  )
}
