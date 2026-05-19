import { useCallback } from 'react'
import * as WailsApp from '@wails/go/main/WailsApp'
import type {
  SendMessageRequest,
  SendMessageResponse,
  ConversationSummary,
  ModelInfo,
  EmergencyResult,
  DisclaimerStatus,
} from '@wails/go/main/WailsApp'

/**
 * Wails 后端绑定方法封装 Hook。
 * 统一处理错误并提供类型安全的调用接口。
 */
export function useWails() {
  const sendMessage = useCallback(
    async (req: SendMessageRequest): Promise<SendMessageResponse> => {
      return await WailsApp.SendMessage(req)
    },
    []
  )

  const sendMessageStream = useCallback(
    async (req: SendMessageRequest): Promise<void> => {
      return await WailsApp.SendMessageStream(req)
    },
    []
  )

  const stopGeneration = useCallback(async (): Promise<void> => {
    return await WailsApp.StopGeneration()
  }, [])

  const getConversations = useCallback(async (): Promise<ConversationSummary[]> => {
    return await WailsApp.GetConversations()
  }, [])

  const createConversation = useCallback(async (): Promise<string> => {
    return await WailsApp.CreateConversation()
  }, [])

  const getModels = useCallback(async (): Promise<ModelInfo[]> => {
    return await WailsApp.GetModels()
  }, [])

  const checkEmergency = useCallback(async (text: string): Promise<EmergencyResult> => {
    return await WailsApp.CheckEmergency(text)
  }, [])

  const generateTitle = useCallback(async (convId: string, userMessage: string): Promise<void> => {
    return await WailsApp.GenerateTitle(convId, userMessage)
  }, [])

  const getDisclaimerStatus = useCallback(async (): Promise<DisclaimerStatus> => {
    return await WailsApp.GetDisclaimerStatus()
  }, [])

  const acceptDisclaimer = useCallback(async (version: string): Promise<void> => {
    return await WailsApp.AcceptDisclaimer(version)
  }, [])

  const declineDisclaimer = useCallback(async (): Promise<void> => {
    return await WailsApp.DeclineDisclaimer()
  }, [])

  const reportComplianceFeedback = useCallback(async (ruleID: string, originalText: string): Promise<void> => {
    return await WailsApp.ReportComplianceFeedback(ruleID, originalText)
  }, [])

  return {
    sendMessage,
    sendMessageStream,
    stopGeneration,
    getConversations,
    createConversation,
    getModels,
    checkEmergency,
    generateTitle,
    getDisclaimerStatus,
    acceptDisclaimer,
    declineDisclaimer,
    reportComplianceFeedback,
  }
}
