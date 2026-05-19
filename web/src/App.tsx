import { useEffect, useState, useCallback } from 'react'
import { HashRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AppLayout } from '@/components/layout/AppLayout'
import { ChatPage } from '@/pages/ChatPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { AboutPage } from '@/pages/AboutPage'
import { DisclaimerModal } from '@/components/DisclaimerModal'
import { UpdateModal } from '@/components/UpdateModal'
import { UpdateBanner } from '@/components/UpdateBanner'
import { OnboardingWizard } from '@/components/onboarding/OnboardingWizard'
import { useTheme } from '@/hooks/useTheme'
import { useWails } from '@/hooks/useWails'
import { useUpdate } from '@/hooks/useUpdate'
import { useOnboardingStore } from '@/stores/onboardingStore'

/**
 * 根组件：全局主题初始化、HashRouter 路由配置与免责声明检测。
 * 桌面端使用 HashRouter 避免无 server 场景下的 404 问题。
 */
function App() {
  useTheme()

  const { getDisclaimerStatus, acceptDisclaimer, declineDisclaimer } = useWails()
  const [disclaimerRequired, setDisclaimerRequired] = useState<boolean | null>(null)
  const [disclaimerText, setDisclaimerText] = useState('')
  const [disclaimerVersion, setDisclaimerVersion] = useState('')

  const onboardingCompleted = useOnboardingStore((s) => s.completed)
  const onboardingSkipped = useOnboardingStore((s) => s.skipped)
  const [showOnboarding, setShowOnboarding] = useState(false)

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
        console.error('Failed to get disclaimer status:', err)
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
        console.error('Failed to accept disclaimer:', err)
      }
    },
    [acceptDisclaimer]
  )

  const handleDecline = useCallback(async () => {
    try {
      await declineDisclaimer()
    } catch (err) {
      console.error('Failed to decline disclaimer:', err)
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

      <HashRouter>
        <Routes>
          <Route element={<AppLayout />}>
            <Route path="/chat" element={<ChatPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/about" element={<AboutPage />} />
            <Route path="/" element={<Navigate to="/chat" replace />} />
            <Route path="*" element={<Navigate to="/chat" replace />} />
          </Route>
        </Routes>
      </HashRouter>

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

      {showOnboarding && <OnboardingWizard onClose={() => setShowOnboarding(false)} />}
    </>
  )
}

export default App
