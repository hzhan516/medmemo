import { useCallback, useState } from 'react'
import { useWails } from './useWails'

export interface FeedbackState {
  isOpen: boolean
  errorInfo?: string
}

/**
 * 问题反馈功能 Hook。
 * 封装系统信息收集与 GitHub Issue 打开逻辑。
 */
export function useFeedback() {
  const { collectSystemInfo, openGitHubIssue } = useWails()
  const [state, setState] = useState<FeedbackState>({ isOpen: false })

  const openFeedback = useCallback((errorInfo?: string) => {
    setState({ isOpen: true, errorInfo })
  }, [])

  const closeFeedback = useCallback(() => {
    setState({ isOpen: false, errorInfo: undefined })
  }, [])

  const submitFeedback = useCallback(
    async (userDescription: string): Promise<void> => {
      const errorLog = state.errorInfo ?? ''
      await openGitHubIssue(userDescription, errorLog)
      setState({ isOpen: false, errorInfo: undefined })
    },
    [openGitHubIssue, state.errorInfo]
  )

  return {
    isOpen: state.isOpen,
    errorInfo: state.errorInfo,
    openFeedback,
    closeFeedback,
    submitFeedback,
    collectSystemInfo,
  }
}
