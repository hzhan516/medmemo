import { useCallback, useState } from 'react'
import { useChatStore } from '@/stores/chatStore'
import { useWails } from './useWails'

/**
 * 会话管理 Hook，封装消息发送、流式输出与状态更新。
 */
export function useConversation() {
  const wails = useWails()
  const {
    messages,
    isStreaming,
    currentConversationId,
    addMessage,
    updateLastMessage,
    setStreaming,
    setConversationId,
  } = useChatStore()

  const [error, setError] = useState<string | null>(null)

  const sendMessage = useCallback(
    async (content: string) => {
      if (!content.trim() || isStreaming) return

      setError(null)

      // 确保当前会话存在
      let convId = currentConversationId
      if (!convId) {
        try {
          convId = await wails.createConversation()
          setConversationId(convId)
        } catch (e) {
          setError('创建会话失败')
          return
        }
      }

      // 添加用户消息
      const userMsg = {
        id: `msg_${Date.now()}_user`,
        role: 'user' as const,
        content: content.trim(),
        timestamp: Date.now(),
      }
      addMessage(userMsg)

      // 添加空的 AI 消息占位
      const aiMsgId = `msg_${Date.now()}_ai`
      addMessage({
        id: aiMsgId,
        role: 'assistant',
        content: '',
        timestamp: Date.now(),
        isStreaming: true,
      })
      setStreaming(true)

      try {
        // 紧急症状检测
        const emergency = await wails.checkEmergency(content.trim())
        if (emergency.level === 'A' || emergency.level === 'B') {
          addMessage({
            id: `msg_${Date.now()}_system`,
            role: 'system',
            content: `【紧急提醒】${emergency.message} 建议操作：${emergency.action}`,
            timestamp: Date.now(),
          })
        }

        // 构造对话请求
        const history = messages.map((m) => ({
          role: m.role,
          content: m.content,
        }))

        const resp = await wails.sendMessage({
          conversation_id: convId,
          messages: [...history, { role: 'user', content: content.trim() }],
          model: 'kimi-lite',
        })

        updateLastMessage(resp.reply, false)
      } catch (e) {
        updateLastMessage('抱歉，消息发送失败，请稍后重试。', false)
        setError(String(e))
      } finally {
        setStreaming(false)
      }
    },
    [currentConversationId, isStreaming, messages, addMessage, updateLastMessage, setStreaming, setConversationId, wails]
  )

  const startNewConversation = useCallback(async () => {
    try {
      const id = await wails.createConversation()
      setConversationId(id)
      useChatStore.setState({ messages: [] })
      setError(null)
    } catch (e) {
      setError('创建新会话失败')
    }
  }, [wails, setConversationId])

  return {
    messages,
    isStreaming,
    currentConversationId,
    sendMessage,
    startNewConversation,
    error,
  }
}
