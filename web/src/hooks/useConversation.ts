import { useCallback, useEffect, useState } from 'react'
import { useChatStore } from '@/stores/chatStore'
import { useWails } from './useWails'
import { EventsOn } from '@wails/runtime/runtime'

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
    appendToLastMessage,
    setLastMessageError,
    abortLastMessage,
    setStreaming,
    setConversationId,
    addConversation,
    selectConversation,
    updateConversation,
  } = useChatStore()

  const [error, setError] = useState<string | null>(null)

  // 注册 Wails 流式事件监听
  useEffect(() => {
    const removeToken = EventsOn('chat:stream:token', (chunk: string) => {
      appendToLastMessage(chunk)
    })
    const removeEnd = EventsOn('chat:stream:end', () => {
      setStreaming(false)
      // 流式结束后更新当前会话的预览和时间
      if (currentConversationId) {
        const lastMsg = useChatStore.getState().messages
        const lastAssistant = [...lastMsg].reverse().find((m) => m.role === 'assistant')
        if (lastAssistant) {
          updateConversation(currentConversationId, {
            preview: lastAssistant.content.slice(0, 60),
            updatedAt: Date.now(),
          })
        }
      }
    })
    const removeError = EventsOn('chat:stream:error', (err: string) => {
      setLastMessageError(err)
      setStreaming(false)
    })
    const removeInterrupted = EventsOn('chat:stream:interrupted', () => {
      abortLastMessage()
      setStreaming(false)
    })

    return () => {
      removeToken()
      removeEnd()
      removeError()
      removeInterrupted()
    }
  }, [appendToLastMessage, setLastMessageError, abortLastMessage, setStreaming, currentConversationId, updateConversation])

  const sendMessage = useCallback(
    async (content: string) => {
      if (!content.trim() || isStreaming) return

      setError(null)

      // 确保当前会话存在
      let convId = currentConversationId
      if (!convId) {
        try {
          convId = await wails.createConversation()
          const newConv = {
            id: convId,
            title: '新对话',
            updatedAt: Date.now(),
            unread: 0,
          }
          addConversation(newConv)
          selectConversation(convId)
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

      // 更新会话 preview
      updateConversation(convId, {
        preview: content.trim().slice(0, 60),
        updatedAt: Date.now(),
      })

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

        // 构造对话请求并启动流式生成
        const history = messages.map((m) => ({
          role: m.role,
          content: m.content,
        }))

        await wails.sendMessageStream({
          conversation_id: convId,
          messages: [...history, { role: 'user', content: content.trim() }],
          model: 'kimi-lite',
        })
      } catch (e) {
        setLastMessageError(String(e))
        setStreaming(false)
        setError(String(e))
      }
    },
    [
      currentConversationId,
      isStreaming,
      messages,
      addMessage,
      appendToLastMessage,
      setLastMessageError,
      setStreaming,
      setConversationId,
      addConversation,
      selectConversation,
      updateConversation,
      wails,
    ]
  )

  const stopGeneration = useCallback(async () => {
    try {
      await wails.stopGeneration()
    } catch (e) {
      console.error('停止生成失败:', e)
    }
  }, [wails])

  const retryMessage = useCallback(
    async (messageId: string) => {
      // 找到最后一条用户消息并重发
      const userMessages = messages.filter((m) => m.role === 'user')
      const lastUserMsg = userMessages[userMessages.length - 1]
      if (lastUserMsg) {
        // 移除后续的 assistant 消息（错误/中断的那条）
        const msgIndex = messages.findIndex((m) => m.id === messageId)
        if (msgIndex >= 0) {
          useChatStore.setState((state) => ({
            messages: state.messages.slice(0, msgIndex),
          }))
        }
        await sendMessage(lastUserMsg.content)
      }
    },
    [messages, sendMessage]
  )

  const startNewConversation = useCallback(async () => {
    try {
      const id = await wails.createConversation()
      const newConv = {
        id,
        title: '新对话',
        updatedAt: Date.now(),
        unread: 0,
      }
      addConversation(newConv)
      selectConversation(id)
      setError(null)
    } catch (e) {
      setError('创建新会话失败')
    }
  }, [wails, addConversation, selectConversation])

  return {
    messages,
    isStreaming,
    currentConversationId,
    sendMessage,
    stopGeneration,
    retryMessage,
    startNewConversation,
    error,
  }
}
