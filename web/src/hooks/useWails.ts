import { useCallback } from 'react'
import * as WailsApp from '@wails/go/main/WailsApp'
import { main, entity, feedback } from '@wails/go/models'
import { toWailsProviderConfig, fromWailsProviderConfig } from '@/utils/providerAdapter'
import type { ProviderConfig } from '@/types/provider'

type SendMessageRequest = main.SendMessageRequest
type SendMessageResponse = main.SendMessageResponse
type ConversationSummary = main.ConversationSummary
type ModelInfo = main.ModelInfo
type EmergencyResult = main.EmergencyResult
type DisclaimerStatus = main.DisclaimerStatus
type UpdateInfoResponse = main.UpdateInfoResponse
type DownloadUpdateRequest = main.DownloadUpdateRequest
type UpdateSettingsResponse = main.UpdateSettingsResponse
type SystemInfo = feedback.SystemInfo

type AuthDetectResult = main.AuthDetectResult
type MessageResponse = main.MessageResponse
type KnowledgeDocumentDTO = main.KnowledgeDocumentDTO
type ImportKnowledgeResponse = main.ImportKnowledgeResponse

import { SaveAPIKey, HasAPIKey, GetVersion, GetVersionInfo, DetectAuthMethods } from '@wails/go/main/WailsApp'

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

  const getDeletedConversations = useCallback(async (): Promise<ConversationSummary[]> => {
    return await WailsApp.GetDeletedConversations()
  }, [])

  const setDataRetentionDays = useCallback(async (days: number): Promise<void> => {
    return await WailsApp.SetDataRetentionDays(days)
  }, [])

  const getConversationMessages = useCallback(async (convID: string): Promise<MessageResponse[]> => {
    return await WailsApp.GetConversationMessages(convID)
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

  const recordAnswerFeedback = useCallback(async (messageID: string, answerType: string, helpful: boolean): Promise<void> => {
    return await WailsApp.RecordAnswerFeedback(messageID, answerType, helpful)
  }, [])

  const checkUpdate = useCallback(async (): Promise<UpdateInfoResponse | null> => {
    return await WailsApp.CheckUpdate()
  }, [])

  const downloadUpdate = useCallback(async (req: DownloadUpdateRequest): Promise<string> => {
    return await WailsApp.DownloadUpdate(req)
  }, [])

  const applyUpdate = useCallback(async (path: string): Promise<void> => {
    return await WailsApp.ApplyUpdate(path)
  }, [])

  const getUpdateSettings = useCallback(async (): Promise<UpdateSettingsResponse> => {
    return await WailsApp.GetUpdateSettings()
  }, [])

  const setUpdateSettings = useCallback(async (req: UpdateSettingsResponse): Promise<void> => {
    return await WailsApp.SetUpdateSettings(req)
  }, [])

  const skipUpdateVersion = useCallback(async (v: string): Promise<void> => {
    return await WailsApp.SkipUpdateVersion(v)
  }, [])

  const openDownloadURL = useCallback((url: string): void => {
    WailsApp.OpenDownloadURL(url)
  }, [])

  const saveAPIKey = useCallback(async (provider: string, apiKey: string): Promise<void> => {
    return await SaveAPIKey(provider, apiKey)
  }, [])

  const hasAPIKey = useCallback(async (provider: string): Promise<boolean> => {
    return await HasAPIKey(provider)
  }, [])

  const getVersion = useCallback(async (): Promise<string> => {
    return await GetVersion()
  }, [])

  const getVersionInfo = useCallback(async (): Promise<main.VersionInfoResponse> => {
    return await GetVersionInfo()
  }, [])

  const detectAuthMethods = useCallback(async (): Promise<AuthDetectResult> => {
    return await DetectAuthMethods()
  }, [])

  const getVersionNotes = useCallback(async (): Promise<entity.VersionNote[]> => {
    return await WailsApp.GetVersionNotes()
  }, [])

  const collectSystemInfo = useCallback(async (): Promise<SystemInfo> => {
    return await WailsApp.CollectSystemInfo()
  }, [])

  const openGitHubIssue = useCallback(async (userDescription: string, errorLog: string): Promise<void> => {
    return await WailsApp.OpenGitHubIssue(userDescription, errorLog)
  }, [])

  const createProvider = useCallback(async (config: ProviderConfig): Promise<void> => {
    return await WailsApp.CreateProvider(toWailsProviderConfig(config))
  }, [])

  const updateProvider = useCallback(async (config: ProviderConfig): Promise<void> => {
    return await WailsApp.UpdateProvider(toWailsProviderConfig(config))
  }, [])

  const deleteProvider = useCallback(async (id: string): Promise<void> => {
    return await WailsApp.DeleteProvider(id)
  }, [])

  const listProviders = useCallback(async (): Promise<ProviderConfig[]> => {
    const list = await WailsApp.ListProviders()
    return list.map(fromWailsProviderConfig)
  }, [])

  const deleteConversation = useCallback(async (convID: string): Promise<void> => {
    return await WailsApp.DeleteConversation(convID)
  }, [])

  const restoreConversation = useCallback(async (convID: string): Promise<void> => {
    return await WailsApp.RestoreConversation(convID)
  }, [])

  const hardDeleteConversation = useCallback(async (convID: string): Promise<void> => {
    return await WailsApp.HardDeleteConversation(convID)
  }, [])

  const getMemories = useCallback(async (limit: number, offset: number): Promise<main.MemoryItem[]> => {
    return await WailsApp.GetMemories(limit, offset)
  }, [])

  const getMemoryByID = useCallback(async (factID: string): Promise<main.MemoryItem> => {
    return await WailsApp.GetMemoryByID(factID)
  }, [])

  const deleteMemory = useCallback(async (factID: string): Promise<void> => {
    return await WailsApp.DeleteMemory(factID)
  }, [])

  const searchMemories = useCallback(async (query: string): Promise<main.MemoryItem[]> => {
    return await WailsApp.SearchMemories(query)
  }, [])

  const getPendingReviews = useCallback(async (limit: number, offset: number): Promise<main.MemoryItem[]> => {
    return await WailsApp.GetPendingReviews(limit, offset)
  }, [])

  const approveFact = useCallback(async (factID: string): Promise<void> => {
    return await WailsApp.ApproveFact(factID)
  }, [])

  const rejectFact = useCallback(async (factID: string): Promise<void> => {
    return await WailsApp.RejectFact(factID)
  }, [])

  const getMemoryStats = useCallback(async (): Promise<main.MemoryStats> => {
    return await WailsApp.GetMemoryStats()
  }, [])

  const getMemoriesBySession = useCallback(async (sessionID: string): Promise<main.MemoryItem[]> => {
    return await WailsApp.GetMemoriesBySession(sessionID)
  }, [])

  const setMemoryInjectionEnabled = useCallback(async (enabled: boolean): Promise<void> => {
    return await WailsApp.SetMemoryInjectionEnabled(enabled)
  }, [])

  const setSessionMemoryInjection = useCallback(async (sessionID: string, enabled: boolean): Promise<void> => {
    return await WailsApp.SetSessionMemoryInjection(sessionID, enabled)
  }, [])

  const getEmbeddingStatus = useCallback(async (): Promise<main.EmbeddingStatusResponse> => {
    return await WailsApp.GetEmbeddingStatus()
  }, [])

  const getEmbeddingModelDirPath = useCallback(async (): Promise<string> => {
    return await WailsApp.GetEmbeddingModelDirPath()
  }, [])

  const openEmbeddingModelDir = useCallback(async (): Promise<void> => {
    return await WailsApp.OpenEmbeddingModelDir()
  }, [])

  const selectKnowledgeFile = useCallback(async (): Promise<string> => {
    return await WailsApp.SelectKnowledgeFile()
  }, [])

  const importKnowledgeFile = useCallback(async (filePath: string): Promise<ImportKnowledgeResponse> => {
    return await WailsApp.ImportKnowledgeFile(filePath)
  }, [])

  const listKnowledgeDocuments = useCallback(async (): Promise<KnowledgeDocumentDTO[]> => {
    return await WailsApp.ListKnowledgeDocuments()
  }, [])

  const deleteKnowledgeDocument = useCallback(async (id: string): Promise<void> => {
    return await WailsApp.DeleteKnowledgeDocument(id)
  }, [])

  const getKnowledgeImportJob = useCallback(async (jobID: string): Promise<ImportKnowledgeResponse> => {
    return await WailsApp.GetKnowledgeImportJob(jobID)
  }, [])

  return {
    sendMessage,
    sendMessageStream,
    stopGeneration,
    getConversations,
    getDeletedConversations,
    setDataRetentionDays,
    getConversationMessages,
    createConversation,
    getModels,
    checkEmergency,
    generateTitle,
    getDisclaimerStatus,
    acceptDisclaimer,
    declineDisclaimer,
    reportComplianceFeedback,
    recordAnswerFeedback,
    checkUpdate,
    downloadUpdate,
    applyUpdate,
    getUpdateSettings,
    setUpdateSettings,
    skipUpdateVersion,
    openDownloadURL,
    saveAPIKey,
    hasAPIKey,
    getVersion,
    getVersionInfo,
    detectAuthMethods,
    getVersionNotes,
    collectSystemInfo,
    openGitHubIssue,
    createProvider,
    updateProvider,
    deleteProvider,
    listProviders,
    deleteConversation,
    restoreConversation,
    hardDeleteConversation,
    getMemories,
    getMemoryByID,
    deleteMemory,
    searchMemories,
    getPendingReviews,
    approveFact,
    rejectFact,
    getMemoryStats,
    getMemoriesBySession,
    setMemoryInjectionEnabled,
    setSessionMemoryInjection,
    getEmbeddingStatus,
    getEmbeddingModelDirPath,
    openEmbeddingModelDir,
    selectKnowledgeFile,
    importKnowledgeFile,
    listKnowledgeDocuments,
    deleteKnowledgeDocument,
    getKnowledgeImportJob,
  }
}
