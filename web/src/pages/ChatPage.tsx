import { useState, useCallback, useEffect } from 'react'
import { ChatContainer } from '@/components/chat/ChatContainer'
import { ChatInput } from '@/components/chat/ChatInput'
import { Sidebar } from '@/components/layout/Sidebar'
import { Header } from '@/components/layout/Header'
import { ComplianceBar } from '@/components/ComplianceBar'
import { EmergencyAlertModal } from '@/components/EmergencyAlertModal'
import { EmergencyWarningBanner } from '@/components/EmergencyWarningBanner'
import { useConversation } from '@/hooks/useConversation'
import { useChatStore } from '@/stores/chatStore'
import { Undo2 } from 'lucide-react'

/**
 * 主对话页面，包含侧边栏、顶部栏、聊天区域和输入区。
 */
export function ChatPage() {
  const {
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
    reportComplianceFeedback,
    error,
  } = useConversation()

  const lastDeleted = useChatStore((s) => s.lastDeleted)
  const undoDelete = useChatStore((s) => s.undoDelete)
  const selectConversation = useChatStore((s) => s.selectConversation)
  const [showUndo, setShowUndo] = useState(false)

  // 监听全局快捷键
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
        e.preventDefault()
        startNewConversation()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [startNewConversation])

  // 显示撤销提示
  useEffect(() => {
    if (lastDeleted) {
      setShowUndo(true)
      const timer = setTimeout(() => setShowUndo(false), 5000)
      return () => clearTimeout(timer)
    }
    setShowUndo(false)
  }, [lastDeleted])

  const handleSelectConversation = useCallback(
    (id: string) => {
      selectConversation(id)
    },
    [selectConversation]
  )

  const handleNewConversation = useCallback(async () => {
    await startNewConversation()
  }, [startNewConversation])

  const handleUndo = useCallback(() => {
    undoDelete()
    setShowUndo(false)
  }, [undoDelete])

  // B 级警告是否展示：有 alert 且 level 为 B
  const showBWarning = emergencyAlert?.level === 'B'
  // 输入区是否被紧急症状阻断
  const inputBlockedByEmergency = showBWarning

  return (
    <div className="flex h-full">
      <Sidebar
        activeId={currentConversationId ?? undefined}
        onSelect={handleSelectConversation}
        onNewConversation={handleNewConversation}
      />

      <div className="flex-1 flex flex-col min-w-0 relative">
        <Header />

        <ComplianceBar conversationId={currentConversationId} />

        <ChatContainer
          messages={messages}
          isStreaming={isStreaming}
          onStartNewConversation={handleNewConversation}
          onRetry={retryMessage}
          onReportCompliance={reportComplianceFeedback}
        />

        {/* B 级紧急症状警告横幅 */}
        {showBWarning && (
          <EmergencyWarningBanner
            message={emergencyAlert.message}
            onAcknowledge={handleAcknowledgeWarning}
            onNotEmergency={handleEmergencyNotEmergency}
          />
        )}

        {error && (
          <div className="shrink-0 px-4 py-2 bg-destructive/10 text-destructive text-xs text-center">
            {error}
          </div>
        )}

        {/* 撤销删除提示 */}
        {showUndo && lastDeleted && (
          <div className="shrink-0 px-4 py-2 bg-accent/50 text-accent-foreground text-xs flex items-center justify-center gap-2">
            <span>会话 "{lastDeleted.title}" 已移至回收站</span>
            <button
              onClick={handleUndo}
              className="flex items-center gap-1 font-medium hover:underline"
            >
              <Undo2 size={12} />
              撤销
            </button>
          </div>
        )}

        <ChatInput
          onSend={sendMessage}
          onStop={stopGeneration}
          onNewConversation={handleNewConversation}
          isLoading={isStreaming}
          blockedByEmergency={inputBlockedByEmergency}
          lastUserMessage={messages.filter((m) => m.role === 'user').slice(-1)[0]?.content}
        />

        {/* A 级紧急症状全屏弹窗（z-index 最高） */}
        {emergencyAlert?.level === 'A' && (
          <EmergencyAlertModal
            open={true}
            message={emergencyAlert.message}
            action={emergencyAlert.action}
            onContinue={handleEmergencyContinue}
            onNotEmergency={handleEmergencyNotEmergency}
          />
        )}
      </div>
    </div>
  )
}
