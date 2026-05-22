import { useCallback, useEffect, useState } from 'react'
import { useChatStore } from '@/stores/chatStore'
import { useSettingsStore } from '@/stores/settingsStore'
import { useProviderStore } from '@/stores/providerStore'
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
    emergencyAlert,
    emergencyWarningAcknowledged,
    addMessage,
    appendToLastMessage,
    setLastMessageError,
    setLastMessageWarnings,
    setLastMessageReplacedTerms,
    setLastMessageTokenUsage,
    setStreaming,
    setConversationId,
    addConversation,
    selectConversation,
    updateConversation,
    setMessages,
    setEmergencyAlert,
    acknowledgeEmergencyWarning,
  } = useChatStore()

  // 当前活跃 Provider 及健康状态
  const activeProviderId = useSettingsStore((s) => s.activeProviderId)
  const activeModelId = useSettingsStore((s) => s.activeModelId)
  const healthStatus = useSettingsStore((s) => s.providerHealthStatus)
  const setActiveProviderId = useSettingsStore((s) => s.setActiveProviderId)
  const setActiveModelId = useSettingsStore((s) => s.setActiveModelId)
  const providers = useProviderStore((s) => s.providers)

  const activeProvider = providers.find((p) => p.id === activeProviderId)
  const activeStatus = activeProvider ? (healthStatus[activeProvider.id] ?? 'unknown') : 'unknown'

  // 获取第一个可用的 green Provider（用于回退）
  const fallbackProvider = providers.find(
    (p) => p.enabled && (healthStatus[p.id] ?? 'unknown') === 'green'
  )

  const [error, setError] = useState<string | null>(null)

  // 注册 Wails 流式事件监听
  useEffect(() => {
    const removeStreamChunk = EventsOn('chat:stream_chunk', (chunk: { type: 'start' | 'content' | 'done' | 'error'; payload: string; metadata?: { model?: string; provider_id?: string; latency_ms?: number; token_count?: number; prompt_tokens?: number; completion_tokens?: number } }) => {
      switch (chunk.type) {
        case 'start':
          // start chunk 仅携带 metadata，无需 UI 操作
          break
        case 'content':
          appendToLastMessage(chunk.payload)
          break
        case 'done': {
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
          // 写入 token 用量统计
          if (chunk.metadata?.prompt_tokens !== undefined && chunk.metadata?.completion_tokens !== undefined) {
            setLastMessageTokenUsage(
              chunk.metadata.prompt_tokens,
              chunk.metadata.completion_tokens,
              (chunk.metadata.prompt_tokens ?? 0) + (chunk.metadata.completion_tokens ?? 0),
            )
          }
          break
        }
        case 'error':
          setLastMessageError(chunk.payload)
          setStreaming(false)
          break
      }
    })
    const removeCompliance = EventsOn('chat:stream:compliance', (payload: { level: string; warning: string; notice: string; replacedTerms?: string[]; matchedRule?: string }) => {
      const warnings: string[] = [payload.level]
      if (payload.warning) warnings.push(`WARNING:${payload.warning}`)
      if (payload.notice) warnings.push(`NOTICE:${payload.notice}`)
      if (payload.matchedRule) warnings.push(`RULE:${payload.matchedRule}`)
      setLastMessageWarnings(warnings)
      if (payload.replacedTerms && payload.replacedTerms.length > 0) {
        setLastMessageReplacedTerms(payload.replacedTerms)
      }
    })
    const removeTitle = EventsOn('chat:title:generated', (payload: { conv_id: string; title: string }) => {
      updateConversation(payload.conv_id, { title: payload.title })
    })

    return () => {
      removeStreamChunk()
      removeCompliance()
      removeTitle()
    }
  }, [appendToLastMessage, setLastMessageError, setLastMessageWarnings, setStreaming, currentConversationId, updateConversation])

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

      // 紧急症状检测（独立于 AI 回复流程）
      const emergency = await wails.checkEmergency(content.trim())
      if (emergency.level === 'A') {
        setEmergencyAlert({
          level: 'A',
          message: emergency.message,
          action: emergency.action,
        })
        // A 级：不发送消息到 LLM，也不添加用户消息到列表
        // 用户需在全屏弹窗中选择操作
        return
      }
      if (emergency.level === 'B') {
        setEmergencyAlert({
          level: 'B',
          message: emergency.message,
          action: emergency.action,
        })
        // B 级：仍添加用户消息，但阻断后续 LLM 调用直到用户确认
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

      // B 级未确认时，不继续发送到 LLM
      if (emergency.level === 'B' && !emergencyWarningAcknowledged) {
        return
      }

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

      // 活跃 Provider 不可用时的回退逻辑
      let targetProvider = activeProvider
      let targetModelId = activeModelId
      if (!targetProvider || activeStatus === 'red') {
        if (fallbackProvider) {
          targetProvider = fallbackProvider
          targetModelId = fallbackProvider.models?.find((m) => m.enabled)?.id || fallbackProvider.modelId
          setActiveProviderId(fallbackProvider.id)
          setActiveModelId(targetModelId)
        }
        // 若仍无可用 Provider，继续使用默认模型（向后兼容）
      }

      try {
        // 构造对话请求并启动流式生成
        const history = messages.map((m) => ({
          role: m.role,
          content: m.content,
        }))

        await wails.sendMessageStream({
          conversation_id: convId,
          messages: [...history, { role: 'user', content: content.trim() }],
          model: targetModelId || targetProvider?.modelId || 'kimi-lite',
          provider_id: targetProvider?.id || '',
        } as Parameters<typeof wails.sendMessageStream>[0])

        // 首条用户消息后异步生成标题（不阻塞流式输出）
        const isFirstMessage = messages.filter((m) => m.role === 'user').length === 0
        if (isFirstMessage) {
          wails.generateTitle(convId, content.trim()).catch(() => {
            // 标题生成失败静默处理，不影响对话流程
          })
        }
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
      emergencyWarningAcknowledged,
      addMessage,
      appendToLastMessage,
      setLastMessageError,
      setLastMessageTokenUsage,
      setStreaming,
      setConversationId,
      addConversation,
      selectConversation,
      updateConversation,
      setEmergencyAlert,
      wails,
      activeModelId,
      setActiveModelId,
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
      setMessages([])
      setError(null)
      // 新建会话时清除紧急症状状态
      setEmergencyAlert(null)
    } catch (e) {
      setError('创建新会话失败')
    }
  }, [wails, addConversation, selectConversation, setMessages, setEmergencyAlert])

  const loadConversationMessages = useCallback(
    async (convID: string) => {
      try {
        const response = await wails.getConversationMessages(convID)
        const mappedMessages = response.map((msg) => ({
          id: msg.id,
          role: msg.role as 'user' | 'assistant' | 'system',
          content: msg.content,
          timestamp: Number(msg.timestamp),
        }))
        setMessages(mappedMessages)
      } catch (e) {
        console.error('加载会话消息失败:', e)
        setMessages([])
      }
    },
    [wails, setMessages]
  )

  // 紧急症状弹窗操作回调
  const handleEmergencyContinue = useCallback(() => {
    setEmergencyAlert(null)
  }, [setEmergencyAlert])

  const handleEmergencyNotEmergency = useCallback(() => {
    // 误判反馈：记录到控制台，关闭弹窗/横幅
    console.warn('[Emergency Feedback] User reported false positive:', emergencyAlert)
    setEmergencyAlert(null)
  }, [emergencyAlert, setEmergencyAlert])

  const handleAcknowledgeWarning = useCallback(() => {
    acknowledgeEmergencyWarning()
    // 确认后自动触发重发最后一条用户消息
    const userMessages = messages.filter((m) => m.role === 'user')
    const lastUserMsg = userMessages[userMessages.length - 1]
    if (lastUserMsg) {
      // 清除 B 级 alert 后重发
      setEmergencyAlert(null)
      sendMessage(lastUserMsg.content)
    } else {
      setEmergencyAlert(null)
    }
  }, [acknowledgeEmergencyWarning, messages, setEmergencyAlert, sendMessage])

  return {
    messages,
    isStreaming,
    currentConversationId,
    emergencyAlert,
    sendMessage,
    stopGeneration,
    retryMessage,
    startNewConversation,
    handleEmergencyContinue,
    handleEmergencyNotEmergency,
    handleAcknowledgeWarning,
    error,
    reportComplianceFeedback: async (messageId: string, ruleID: string) => {
      const msg = messages.find((m) => m.id === messageId)
      if (!msg) return
      try {
        await wails.reportComplianceFeedback(ruleID, msg.content)
        useChatStore.setState((state) => ({
          messages: state.messages.map((m) =>
            m.id === messageId ? { ...m, complianceFeedback: 'submitted' as const } : m
          ),
        }))
      } catch (e) {
        console.error('提交合规反馈失败:', e)
      }
    },
  }
}
