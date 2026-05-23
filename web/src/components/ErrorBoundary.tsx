import React, { Component, type ReactNode } from 'react'
import { logger } from '@/lib/logger'

interface ErrorBoundaryProps {
  children: ReactNode
  onError: (error: Error, errorInfo: React.ErrorInfo) => void
}

interface ErrorBoundaryState {
  hasError: boolean
}

/**
 * React 错误边界组件。
 * 捕获子组件树中的未处理异常，通过回调通知父组件弹出反馈弹窗。
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    logger.error('[ErrorBoundary] Uncaught error:', error, errorInfo)
    this.props.onError(error, errorInfo)
  }

  render() {
    // 错误时返回 null，由父组件的 FeedbackModal 接管展示
    // 正常时渲染子树
    if (this.state.hasError) {
      return (
        <div className="h-screen w-screen flex items-center justify-center bg-background">
          <div className="text-center space-y-3">
            <div className="w-10 h-10 mx-auto rounded-full bg-destructive/10 flex items-center justify-center">
              <span className="text-destructive text-lg">!</span>
            </div>
            <p className="text-sm text-muted-foreground">应用遇到了问题，正在准备反馈…</p>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
