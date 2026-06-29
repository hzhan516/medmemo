import { useEffect, useState, useCallback, lazy, Suspense } from 'react'
import { EventsOn } from '@wails/runtime/runtime'
import { logger } from '@/lib/logger'
import { HashRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AppLayout } from '@/components/layout/AppLayout'
import { ChatPage } from '@/pages/ChatPage'
import { AboutPage } from '@/pages/AboutPage'
import { MemoryPage } from '@/pages/MemoryPage'
import { KnowledgePage } from '@/pages/KnowledgePage'
import { DisclaimerModal } from '@/components/DisclaimerModal'
import { UpdateModal } from '@/components/UpdateModal'
import { UpdateBanner } from '@/components/UpdateBanner'
import { VersionNotesModal } from '@/components/VersionNotesModal'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import { FeedbackModal } from '@/components/FeedbackModal'
import { useFeedback } from '@/hooks/useFeedback'

// 动态导入：设置页面（含大量表单依赖）和安装向导（仅首次启动使用）
const SettingsPage = lazy(() => import('@/pages/SettingsPage').then((m) => ({ default: m.SettingsPage })))
const OnboardingWizard = lazy(() =>
  import('@/components/onboarding/OnboardingWizard').then((m) => ({ default: m.OnboardingWizard }))
)
import { useTheme } from '@/hooks/useTheme'
import { useWails } from '@/hooks/useWails'
import { useUpdate } from '@/hooks/useUpdate'
import { useVersionNotes } from '@/hooks/useVersionNotes'
import { useOnboardingStore } from '@/stores/onboardingStore'
import { useProviderStore } from '@/stores/providerStore'
import { useChatStore } from '@/stores/chatStore'


/**
 * 根组件：全局主题初始化、HashRouter 路由配置与免责声明检测。
 * 桌面端使用 HashRouter 避免无 server 场景下的 404 问题。
 */
function App() {
  useTheme()

  const { getDisclaimerStatus, acceptDisclaimer, declineDisclaimer, listProviders, createProvider, getConversations, getDeletedConversations, getConversationMessages } = useWails()

  const setProviders = useProviderStore((s) => s.setProviders)
  const initialized = useProviderStore((s) => s.initialized)
  const setConversations = useChatStore((s) => s.setConversations)
  const setDeletedConversations = useChatStore((s) => s.setDeletedConversations)
  const selectConversation = useChatStore((s) => s.selectConversation)
  const setMessages = useChatStore((s) => s.setMessages)
  const [disclaimerRequired, setDisclaimerRequired] = useState<boolean | null>(null)
  const [disclaimerText, setDisclaimerText] = useState('')
  const [disclaimerVersion, setDisclaimerVersion] = useState('')

  const onboardingCompleted = useOnboardingStore((s) => s.completed)
  const onboardingSkipped = useOnboardingStore((s) => s.skipped)
  const [showOnboarding, setShowOnboarding] = useState(false)

  const { notes, currentVersion, shouldShow: showVersionNotes, dismiss: dismissVersionNotes } = useVersionNotes()

  const { isOpen: feedbackOpen, errorInfo: feedbackErrorInfo, openFeedback, closeFeedback, submitFeedback } = useFeedback()

  const {
    updateInfo,
    downloadProgress,
    isDownloading,
    isRestarting,
    downloadPath,
    error: updateError,
    doDownload,
    doApply,
    doSkip,
    dismissUpdate,
    openDownloadPage,
  } = useUpdate()

  // v1.1.4: embedding 迁移状态监听
  const [migrationStatus, setMigrationStatus] = useState<{
    active: boolean
    processed: number
    total: number
  } | null>(null)

  useEffect(() => {
    const unsubStart = EventsOn('embedding:migration:start', (data: { total: number }) => {
      setMigrationStatus({ active: true, processed: 0, total: data.total })
    })
    const unsubProgress = EventsOn('embedding:migration:progress', (data: { processed: number; total: number }) => {
      setMigrationStatus({ active: true, processed: data.processed, total: data.total })
    })
    const unsubDone = EventsOn('embedding:migration:done', () => {
      setMigrationStatus(null)
    })
    return () => {
      unsubStart()
      unsubProgress()
      unsubDone()
    }
  }, [])

  // 应用启动时加载 Provider 列表（从后端 SQLite）
  useEffect(() => {
    if (initialized) return
    let cancelled = false
    ;(async () => {
      try {
        const backendProviders = await listProviders()
        if (cancelled) return

        // 若后端为空且 localStorage 有旧数据，执行迁移
        if (backendProviders.length === 0) {
          const raw = localStorage.getItem('medmemo-providers')
          if (raw) {
            try {
              const parsed = JSON.parse(raw)
              const oldProviders: unknown[] = parsed?.state?.providers || []
              if (oldProviders.length > 0) {
                const migrated = oldProviders.filter((p): p is Record<string, unknown> => {
                  if (typeof p !== 'object' || p === null) return false
                  return 'id' in p && typeof (p as Record<string, unknown>).id === 'string'
                })
                for (const p of migrated) {
                  try {
                    await createProvider({
                      id: String(p.id),
                      name: String(p.name || ''),
                      apiHost: String(p.apiHost || ''),
                      apiKey: String(p.apiKey || ''),
                      modelId: String(p.modelId || ''),
                      models: Array.isArray(p.models) ? p.models : [],
                      temperature: Number(p.temperature ?? 0.7),
                      timeoutMs: Number(p.timeoutMs ?? 30000),
                      maxRetries: Number(p.maxRetries ?? 3),
                      group: String(p.group || ''),
                      enabled: Boolean(p.enabled),
                      sortOrder: Number(p.sortOrder ?? 0),
                      createdAt: Number(p.createdAt ?? Date.now()),
                      updatedAt: Number(p.updatedAt ?? Date.now()),
                      auth_method: String(p.authMethod || p.auth_method || 'api_key'),
                      auth_params: p.authParams || p.auth_params || {},
                    } as unknown as Parameters<typeof createProvider>[0])
                  } catch (migrateErr) {
                    logger.error('Failed to migrate provider:', p.id, migrateErr)
                  }
                }
                // 迁移完成后重新加载
                const reloaded = await listProviders()
                if (!cancelled) setProviders(reloaded)
                localStorage.removeItem('medmemo-providers')
                logger.warn(`已迁移 ${migrated.length} 个 Provider 到后端存储`)
                return
              }
            } catch (e) {
              logger.error('Failed to parse old provider storage:', e)
            }
          }
        }

        setProviders(backendProviders)
      } catch (err) {
        logger.error('Failed to load providers:', err)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [initialized, listProviders, createProvider, setProviders])

  // 应用启动时加载对话列表（从后端 SQLite）
  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const [backendConversations, backendDeleted] = await Promise.all([
          getConversations(),
          getDeletedConversations().catch((err) => {
            logger.error('Failed to load deleted conversations:', err)
            return [] as Awaited<ReturnType<typeof getDeletedConversations>>
          }),
        ])
        if (cancelled) return
        const mapped = backendConversations.map((conv) => ({
          id: conv.id,
          title: conv.title,
          updatedAt: Number(conv.updated_at),
          unread: 0,
        }))
        setConversations(mapped)

        const mappedDeleted = backendDeleted.map((conv) => ({
          id: conv.id,
          title: conv.title,
          updatedAt: Number(conv.updated_at),
          deletedAt: conv.deleted_at ? Number(conv.deleted_at) : undefined,
          unread: 0,
        }))
        setDeletedConversations(mappedDeleted)

        // 自动选中最近更新的对话并加载消息
        if (mapped.length > 0) {
          const latest = mapped[0]
          selectConversation(latest.id)
          try {
            const msgResponse = await getConversationMessages(latest.id)
            if (cancelled) return
            const mappedMessages = msgResponse.map((msg) => ({
              id: msg.id,
              role: msg.role as 'user' | 'assistant' | 'system',
              content: msg.content,
              timestamp: Number(msg.timestamp),
              promptTokens: msg.prompt_tokens,
              completionTokens: msg.completion_tokens,
              totalTokens: msg.total_tokens,
              confidence: msg.confidence
                ? {
                    overallScore: (msg.confidence as Record<string, unknown>).overall_score as number,
                    level: (msg.confidence as Record<string, unknown>).level as 'A' | 'B' | 'C' | 'D' | 'E',
                    breakdown: {
                      knowledge_source: ((msg.confidence as Record<string, unknown>).breakdown as Record<string, number>)?.knowledge_source ?? 0,
                      reasoning: ((msg.confidence as Record<string, unknown>).breakdown as Record<string, number>)?.reasoning ?? 0,
                      context: ((msg.confidence as Record<string, unknown>).breakdown as Record<string, number>)?.context ?? 0,
                      history: ((msg.confidence as Record<string, unknown>).breakdown as Record<string, number>)?.history ?? 0,
                      uncertainty: ((msg.confidence as Record<string, unknown>).breakdown as Record<string, number>)?.uncertainty ?? 0,
                    },
                    explanation: (msg.confidence as Record<string, unknown>).explanation as string,
                    suggestion: (msg.confidence as Record<string, unknown>).suggestion as string,
                    missingInfo: ((msg.confidence as Record<string, unknown>).missing_info as string[]) ?? [],
                  }
                : undefined,
            }))
            setMessages(mappedMessages)
          } catch (msgErr) {
            logger.error('Failed to load conversation messages:', msgErr)
          }
        }
      } catch (err) {
        logger.error('Failed to load conversations:', err)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [getConversations, getDeletedConversations, getConversationMessages, setConversations, setDeletedConversations, selectConversation, setMessages])

  // 应用启动时检测免责声明状态
  useEffect(() => {
    let cancelled = false
    getDisclaimerStatus()
      .then((status) => {
        if (cancelled) return
        setDisclaimerRequired(status.required)
        setDisclaimerText(status.text)
        setDisclaimerVersion(status.version)
      })
      .catch((err) => {
        logger.error('Failed to get disclaimer status:', err)
        // 若后端查询失败，保守策略：强制展示免责声明
        if (!cancelled) {
          setDisclaimerRequired(true)
        }
      })
    return () => {
      cancelled = true
    }
  }, [getDisclaimerStatus])

  const handleAccept = useCallback(
    async (version: string) => {
      try {
        await acceptDisclaimer(version)
        setDisclaimerRequired(false)
      } catch (err) {
        logger.error('Failed to accept disclaimer:', err)
      }
    },
    [acceptDisclaimer]
  )

  const handleDecline = useCallback(async () => {
    try {
      await declineDisclaimer()
    } catch (err) {
      logger.error('Failed to decline disclaimer:', err)
    }
  }, [declineDisclaimer])

  // 免责声明完成后，检测是否需要展示安装向导
  useEffect(() => {
    if (disclaimerRequired === false && !onboardingCompleted && !onboardingSkipped) {
      setShowOnboarding(true)
    }
  }, [disclaimerRequired, onboardingCompleted, onboardingSkipped])

  // 等待免责声明状态检测完成，避免闪烁
  if (disclaimerRequired === null) {
    return (
      <div className="h-screen w-screen flex items-center justify-center bg-background text-foreground">
        <div className="flex flex-col items-center gap-3">
          <div className="w-8 h-8 rounded-full border-2 border-primary border-t-transparent animate-spin" />
          <span className="text-sm text-muted-foreground">正在初始化…</span>
        </div>
      </div>
    )
  }

  return (
    <>
      <UpdateBanner info={updateInfo} onShowDetails={() => {}} onDismiss={dismissUpdate} />

      {/* v1.1.4: embedding 迁移进度 UI */}
      {migrationStatus?.active && (migrationStatus.total <= 100 || (migrationStatus.total > 100 && migrationStatus.processed < 100)) && (
        <div className="fixed inset-0 z-[9999] bg-background/95 flex items-center justify-center">
          <div className="flex flex-col items-center gap-4">
            <div className="w-10 h-10 rounded-full border-2 border-primary border-t-transparent animate-spin" />
            <span className="text-sm text-muted-foreground">
              正在更新记忆索引 ({migrationStatus.processed}/{migrationStatus.total})…
            </span>
            <span className="text-xs text-muted-foreground/60">
              仅在版本更新后首次启动时执行
            </span>
          </div>
        </div>
      )}

      {migrationStatus?.active && migrationStatus.total > 100 && migrationStatus.processed >= 100 && (
        <div className="fixed bottom-0 left-0 right-0 z-50 bg-muted/90 backdrop-blur px-4 py-2 flex items-center gap-3 text-sm">
          <div className="w-4 h-4 rounded-full border-2 border-primary border-t-transparent animate-spin" />
          <span className="text-muted-foreground">
            正在后台更新记忆索引 ({migrationStatus.processed}/{migrationStatus.total})…
          </span>
          <div className="flex-1 h-1.5 bg-muted-foreground/20 rounded-full overflow-hidden">
            <div
              className="h-full bg-primary rounded-full transition-all duration-300"
              style={{ width: `${(migrationStatus.processed / migrationStatus.total) * 100}%` }}
            />
          </div>
        </div>
      )}

      <ErrorBoundary onError={(error, _errorInfo) => openFeedback(`${error.name}: ${error.message}\n${_errorInfo.componentStack ?? ''}`)}>
        <HashRouter>
          <Routes>
            <Route element={<AppLayout />}>
              <Route path="/chat" element={<ChatPage />} />
              <Route path="/memories" element={<MemoryPage />} />
              <Route path="/knowledge" element={<KnowledgePage />} />
              <Route
                path="/settings"
                element={
                  <Suspense
                    fallback={
                      <div className="h-full flex items-center justify-center text-sm text-muted-foreground">
                        加载中…
                      </div>
                    }
                  >
                    <SettingsPage />
                  </Suspense>
                }
              />
              <Route path="/about" element={<AboutPage onOpenFeedback={openFeedback} />} />
              <Route path="/" element={<Navigate to="/chat" replace />} />
              <Route path="*" element={<Navigate to="/chat" replace />} />
            </Route>
          </Routes>
        </HashRouter>
      </ErrorBoundary>

      {disclaimerRequired && (
        <DisclaimerModal
          text={disclaimerText}
          version={disclaimerVersion}
          onAccept={handleAccept}
          onDecline={handleDecline}
        />
      )}

      <UpdateModal
        info={updateInfo}
        isDownloading={isDownloading}
        isRestarting={isRestarting}
        downloadProgress={downloadProgress}
        downloadPath={downloadPath}
        error={updateError}
        onDownload={doDownload}
        onApply={doApply}
        onSkip={doSkip}
        onDismiss={dismissUpdate}
        onOpenDownloadPage={openDownloadPage}
      />

      <Suspense fallback={null}>
        {showOnboarding && <OnboardingWizard onClose={() => setShowOnboarding(false)} />}
      </Suspense>

      {showVersionNotes && (
        <VersionNotesModal
          notes={notes}
          currentVersion={currentVersion}
          onDismiss={dismissVersionNotes}
        />
      )}

      <FeedbackModal
        isOpen={feedbackOpen}
        errorInfo={feedbackErrorInfo}
        onClose={closeFeedback}
        onSubmit={submitFeedback}
      />
    </>
  )
}

export default App
