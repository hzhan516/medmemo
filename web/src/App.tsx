import { useEffect, useState, useCallback, lazy, Suspense } from 'react'
import { logger } from '@/lib/logger'
import { HashRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AppLayout } from '@/components/layout/AppLayout'
import { ChatPage } from '@/pages/ChatPage'
import { AboutPage } from '@/pages/AboutPage'
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

  const { getDisclaimerStatus, acceptDisclaimer, declineDisclaimer, listProviders, createProvider, getConversations, getConversationMessages } = useWails()

  const setProviders = useProviderStore((s) => s.setProviders)
  const initialized = useProviderStore((s) => s.initialized)
  const setConversations = useChatStore((s) => s.setConversations)
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
    downloadPath,
    error: updateError,
    doDownload,
    doApply,
    doSkip,
    dismissUpdate,
    openDownloadPage,
  } = useUpdate()

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
        const backendConversations = await getConversations()
        if (cancelled) return
        const mapped = backendConversations.map((conv) => ({
          id: conv.id,
          title: conv.title,
          updatedAt: Number(conv.updated_at),
          unread: 0,
        }))
        setConversations(mapped)
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
  }, [getConversations, getConversationMessages, setConversations, selectConversation, setMessages])

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

      <ErrorBoundary onError={(error, _errorInfo) => openFeedback(`${error.name}: ${error.message}\n${_errorInfo.componentStack ?? ''}`)}>
        <HashRouter>
          <Routes>
            <Route element={<AppLayout />}>
              <Route path="/chat" element={<ChatPage />} />
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
