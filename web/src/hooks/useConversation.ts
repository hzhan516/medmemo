import { useCallback, useEffect, useRef, useState } from 'react'
import { useChatStore, type Conversation } from '@/stores/chatStore'
import { useSettingsStore } from '@/stores/settingsStore'
import { useProviderStore } from '@/stores/providerStore'
import { useWails } from './useWails'
import { EventsOn } from '@wails/runtime/runtime'

/**
 * 粗略估算文本 token 数。
 * 中文/东亚字符按 1 token 估算，英文按 0.3 token 估算，作为前端展示用。
 */
function estimateTokens(text: string): number {
  let count = 0
  for (const char of text) {
    // CJK 统一表意文字范围
    if (/[\u4e00-\u9fff\u3000-\u303f\uff00-\uffef]/.test(char)) {
      count += 1
    } else if (/\s/.test(char)) {
      // 空白字符不计
    } else {
      count += 0.3
    }
  }
  return Math.max(1, Math.ceil(count))
}

/**
 * 会话管理 Hook，封装消息发送、流式输出与状态更新。
 */
export function useConversation() {
  const wails = useWails()
  const {
    messages,
    isStreaming,
    streamingIds,
    currentConversationId,
    emergencyAlert,
    emergencyWarningAcknowledged,
    addMessage,
    appendToLastMessageForConversation,
    setLastMessageError,
    setLastMessageErrorForConversation,
    setLastMessageWarningsForConversation,
    setLastMessageReplacedTermsForConversation,
    setLastMessageTokenUsageForConversation,
    setStreamingForConversation,
    setConversationId,
    addConversation,
    selectConversation,
    updateConversation,
    setEmergencyAlert,
    acknowledgeEmergencyWarning,
    setMessages,
    setConversations,
    setDeletedConversations,
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

  // 当前正在加载历史消息的会话 ID，用于竞态保护
  const pendingLoadRef = useRef<string | null>(null)
  // 标记是否已执行过启动加载
  const hydratedRef = useRef(false)

  // 注册 Wails 流式事件监听
  useEffect(() => {
    const removeStreamChunk = EventsOn('chat:stream_chunk', (chunk: {
      type: 'start' | 'content' | 'done' | 'error'
      payload: string
      metadata?: {
        conversation_id?: string
        model?: string
        provider_id?: string
        latency_ms?: number
        token_count?: number
        prompt_tokens?: number
        completion_tokens?: number
      }
    }) => {
      const convId = chunk.metadata?.conversation_id
      if (!convId) {
        console.warn('[stream_chunk] 缺少 conversation_id，忽略')
        return
      }

      switch (chunk.type) {
        case 'start':
          // start chunk 仅携带 metadata，无需 UI 操作
          break
        case 'content':
          appendToLastMessageForConversation(convId, chunk.payload)
          break
        case 'done':
          setStreamingForConversation(convId, false)
          // 更新对应会话的预览和时间
          const state = useChatStore.getState()
          const convMsgs = state.messagesMap[convId] || []
          const lastAssistant = [...convMsgs].reverse().find((m) => m.role === 'assistant')
          if (lastAssistant) {
            updateConversation(convId, {
              preview: lastAssistant.content.slice(0, 60),
              updatedAt: Date.now(),
            })
          }
          // 写入 token 用量统计
          if (chunk.metadata?.prompt_tokens !== undefined && chunk.metadata?.completion_tokens !== undefined) {
            const promptTokens = chunk.metadata.prompt_tokens
            const completionTokens = chunk.metadata.completion_tokens
            const totalTokens = promptTokens + completionTokens
            setLastMessageTokenUsageForConversation(convId, promptTokens, completionTokens, totalTokens)
            // 同时给对应会话的最后一条用户消息标记输入 token 数
            const allMsgs = state.messagesMap[convId] || []
            const lastUserMsgIdx = [...allMsgs].reverse().findIndex((m) => m.role === 'user')
            if (lastUserMsgIdx >= 0) {
              const actualIdx = allMsgs.length - 1 - lastUserMsgIdx
              useChatStore.setState((s) => {
                const convMsgs = [...(s.messagesMap[convId] || [])]
                convMsgs[actualIdx] = { ...convMsgs[actualIdx], totalTokens: promptTokens }
                const newMap = { ...s.messagesMap, [convId]: convMsgs }
                if (s.currentConversationId === convId) {
                  return { messages: convMsgs, messagesMap: newMap }
                }
                return { messagesMap: newMap }
              })
            }
          }
          break
        case 'error':
          setStreamingForConversation(convId, false)
          setLastMessageErrorForConversation(convId, chunk.payload)
          break
      }
    })

    const removeCompliance = EventsOn('chat:stream:compliance', (payload: {
      conversation_id?: string
      level: string
      warning: string
      notice: string
      replacedTerms?: string[]
      matchedRule?: string
    }) => {
      const convId = payload.conversation_id
      if (!convId) {
        console.warn('[stream:compliance] 缺少 conversation_id，忽略')
        return
      }
      const warnings: string[] = [payload.level]
      if (payload.warning) warnings.push(`WARNING:${payload.warning}`)
      if (payload.notice) warnings.push(`NOTICE:${payload.notice}`)
      if (payload.matchedRule) warnings.push(`RULE:${payload.matchedRule}`)
      setLastMessageWarningsForConversation(convId, warnings)
      if (payload.replacedTerms && payload.replacedTerms.length > 0) {
        setLastMessageReplacedTermsForConversation(convId, payload.replacedTerms)
      }
    })

    const removeTitle = EventsOn('chat:title:generated', (payload: { conv_id: string; title: string }) => {
      // 标题生成按 conv_id 处理，始终更新对应会话（不受会话切换影响）
      updateConversation(payload.conv_id, { title: payload.title })
    })

    return () => {
      removeStreamChunk()
      removeCompliance()
      removeTitle()
    }
  }, [appendToLastMessageForConversation, setLastMessageErrorForConversation, setLastMessageWarningsForConversation, setLastMessageReplacedTermsForConversation, setLastMessageTokenUsageForConversation, setStreamingForConversation, updateConversation])

  // 启动时加载历史会话列表
  useEffect(() => {
    if (hydratedRef.current) return
    hydratedRef.current = true

    const loadConversations = async () => {
      try {
        const convs = await wails.getConversations()
        const active: Conversation[] = []
        const deleted: Conversation[] = []
        convs.forEach((c) => {
          const conv: Conversation = {
            id: c.id,
            title: c.title,
            updatedAt: parseInt(c.updated_at, 10),
            unread: 0,
          }
          if (c.deleted_at) {
            conv.deletedAt = parseInt(c.deleted_at, 10)
            deleted.push(conv)
          } else {
            active.push(conv)
          }
        })
        setConversations(active)
        setDeletedConversations(deleted)
        console.log('[hydrate] 加载历史会话:', active.length, '条，回收站:', deleted.length, '条')
      } catch (e) {
        console.error('[hydrate] 加载历史会话失败:', e)
      }
    }

    loadConversations()
  }, [wails, setConversations])

  const sendMessage = useCallback(
    async (content: string) => {
      if (!content.trim()) return

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

      // 检查当前会话是否已在流式中（按会话隔离）
      if (streamingIds.has(convId)) {
        console.log('[sendMessage] 会话', convId, '正在流式中，忽略重复发送')
        return
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

      // 添加用户消息（同时估算 token 数用于前端展示）
      const trimmedContent = content.trim()
      const estimatedTokens = estimateTokens(trimmedContent)
      const userMsg = {
        id: `msg_${Date.now()}_user`,
        role: 'user' as const,
        content: trimmedContent,
        timestamp: Date.now(),
        totalTokens: estimatedTokens,
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
      setStreamingForConversation(convId, true)

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
        setStreamingForConversation(convId, false)
        setError(String(e))
      }
    },
    [
      currentConversationId,
      streamingIds,
      messages,
      emergencyWarningAcknowledged,
      addMessage,
      appendToLastMessageForConversation,
      setLastMessageError,
      setLastMessageErrorForConversation,
      setLastMessageTokenUsageForConversation,
      setStreamingForConversation,
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
          useChatStore.setState((state) => {
            const convId = state.currentConversationId
            const newMsgs = state.messages.slice(0, msgIndex)
            if (!convId) return { messages: newMsgs }
            return {
              messages: newMsgs,
              messagesMap: { ...state.messagesMap, [convId]: newMsgs },
            }
          })
        }
        await sendMessage(lastUserMsg.content)
      }
    },
    [messages, sendMessage]
  )

  // 加载指定会话的历史消息
  const loadConversationMessages = useCallback(
    async (convId: string) => {
      pendingLoadRef.current = convId
      try {
        const rawMsgs = await wails.getConversationMessages(convId)
        console.log('[loadConversationMessages]', convId, '加载到', rawMsgs.length, '条消息')
        // 竞态保护：若用户已切换到其他会话，丢弃旧结果
        if (useChatStore.getState().currentConversationId !== convId) {
          console.log('[loadConversationMessages] 会话已切换，丢弃结果:', convId)
          return
        }
        const loadedMessages = rawMsgs.map((m) => ({
          id: m.id,
          role: m.role as 'user' | 'assistant' | 'system',
          content: m.content,
          timestamp: parseInt(m.timestamp, 10),
        }))
        setMessages(loadedMessages)
      } catch (e) {
        console.error('加载历史消息失败:', e)
        setError('加载历史消息失败')
      } finally {
        if (pendingLoadRef.current === convId) {
          pendingLoadRef.current = null
        }
      }
    },
    [wails, setMessages]
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
      // 新建会话时清除紧急症状状态
      setEmergencyAlert(null)
    } catch (e) {
      setError('创建新会话失败')
    }
  }, [wails, addConversation, selectConversation, setEmergencyAlert])

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
    loadConversationMessages,
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
