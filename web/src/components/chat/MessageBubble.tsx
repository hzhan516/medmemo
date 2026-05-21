import { User, Bot, AlertCircle, Copy, RotateCcw, ShieldAlert, Info, ThumbsDown, CheckCircle } from 'lucide-react'
import type { ChatMessage } from '@/stores/chatStore'
import { MarkdownRenderer } from '@/components/markdown/MarkdownRenderer'

interface MessageBubbleProps {
  message: ChatMessage
  onRetry?: (messageId: string) => void
  onReportCompliance?: (messageId: string, ruleID: string) => void
}

/**
 * 消息气泡组件。
 * 用户消息：蓝色渐变，右侧对齐，圆角 16px。
 * AI 消息：白色/暗色背景，左侧对齐。
 * 系统提示：浅色背景，居中，13px 小字。
 * 合规标记：L2_WARNING 橙色警告框，L3_NOTICE 蓝色提示条。
 */
export function MessageBubble({ message, onRetry, onReportCompliance }: MessageBubbleProps) {
  const { id, role, content, isStreaming, interrupted, error, warnings, replacedTerms, complianceFeedback } = message

  // 解析合规级别
  const hasL1Blocked = warnings?.some((w) => w === 'L1_BLOCKED')
  const hasL2Warning = warnings?.some((w) => w === 'L2_WARNING')
  const hasL3Notice = warnings?.some((w) => w === 'L3_NOTICE')
  const l2WarningText = warnings?.find((w) => w.startsWith('WARNING:'))?.replace('WARNING:', '')
  const l3NoticeText = warnings?.find((w) => w.startsWith('NOTICE:'))?.replace('NOTICE:', '')

  const hasComplianceIssue = hasL1Blocked || hasL2Warning || hasL3Notice || (replacedTerms && replacedTerms.length > 0)
  const isFeedbackSubmitted = complianceFeedback === 'submitted'
  const firstReplacedRule = replacedTerms && replacedTerms.length > 0 ? replacedTerms[0] : ''

  const handleReportCompliance = () => {
    if (isFeedbackSubmitted || !onReportCompliance) return
    // 优先使用命中的规则 ID，fallback 到 replacedTerms
    const ruleID = warnings?.find((w) => w.startsWith('RULE:'))?.replace('RULE:', '') || firstReplacedRule || 'unknown'
    onReportCompliance(id, ruleID)
  }

  if (role === 'system') {
    return (
      <div className="flex justify-center my-3">
        <div className="flex items-center gap-1.5 px-4 py-2 rounded-full bg-system-gray text-xs text-muted-foreground">
          <AlertCircle size={12} />
          <span>{content}</span>
        </div>
      </div>
    )
  }

  const isUser = role === 'user'

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content)
    } catch {
      // 忽略复制失败
    }
  }

  return (
    <div
      className={`flex gap-3 my-4 ${isUser ? 'flex-row-reverse' : 'flex-row'}`}
    >
      {/* 头像 */}
      <div
        className={`
          shrink-0 w-8 h-8 rounded-full flex items-center justify-center
          ${isUser
            ? 'bg-gradient-to-br from-user-blue to-user-blue-dark text-white'
            : 'bg-accent text-accent-foreground'
          }
        `}
      >
        {isUser ? <User size={16} /> : <Bot size={16} />}
      </div>

      {/* 气泡 */}
      <div
        className={`
          max-w-[80%] px-4 py-3 text-sm leading-relaxed
          ${isUser
            ? 'bg-gradient-to-br from-user-blue to-user-blue-dark text-white rounded-2xl rounded-tr-sm'
            : `bg-ai-bg dark:bg-ai-bg-dark text-ai-text dark:text-gray-200 border rounded-2xl rounded-tl-sm shadow-sm ${error ? 'border-destructive' : 'border-border'}`
          }
        `}
      >
        {isUser ? (
          <div className="whitespace-pre-wrap break-words">
            {content}
            {isStreaming && (
              <span className="inline-block w-1.5 h-4 ml-0.5 bg-current opacity-50 animate-pulse" />
            )}
          </div>
        ) : (
          <div className="break-words">
            {/* 错误状态 */}
            {error && (
              <div className="flex items-start gap-2 mb-2 p-2 rounded-lg bg-destructive/10 text-destructive text-xs">
                <AlertCircle size={14} className="shrink-0 mt-0.5" />
                <div className="flex-1 min-w-0">
                  <p className="font-medium">生成失败</p>
                  <p className="opacity-80">{error}</p>
                </div>
              </div>
            )}

            {/* L1 替换提示 */}
            {(hasL1Blocked || (replacedTerms && replacedTerms.length > 0)) && (
              <div className="flex items-start gap-2 mb-2 p-2 rounded-lg bg-gray-50 dark:bg-gray-800/40 border border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400 text-xs">
                <Info size={14} className="shrink-0 mt-0.5" />
                <div className="flex-1 min-w-0">
                  <p className="font-medium">内容已调整为合规表述</p>
                  <p className="opacity-80">原文中的部分用语已被替换为更安全的表达方式。</p>
                </div>
              </div>
            )}

            {/* L2 警告框 */}
            {hasL2Warning && (
              <div className="flex items-start gap-2 mb-2 p-2 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 text-amber-700 dark:text-amber-400 text-xs">
                <ShieldAlert size={14} className="shrink-0 mt-0.5" />
                <div className="flex-1 min-w-0">
                  <p className="font-medium">内容风险提示</p>
                  <p className="opacity-80">{l2WarningText || "以上内容仅为信息参考，不能替代专业医疗诊断。"}</p>
                </div>
              </div>
            )}

            <MarkdownRenderer content={content} />

            {/* 中断标记 */}
            {interrupted && (
              <span className="text-xs text-muted-foreground ml-1">
                [用户中断]
              </span>
            )}

            {/* 流式光标 */}
            {isStreaming && content.length > 0 && (
              <span className="inline-block w-1.5 h-4 ml-0.5 bg-current opacity-50 animate-pulse" />
            )}

            {/* L3 提示条 */}
            {hasL3Notice && (
              <div className="flex items-start gap-2 mt-2 pt-2 border-t border-blue-200 dark:border-blue-800 text-blue-600 dark:text-blue-400 text-xs">
                <Info size={12} className="shrink-0 mt-0.5" />
                <span>{l3NoticeText || "以上内容仅为健康科普信息，不能替代专业医疗诊断。"}</span>
              </div>
            )}

            {/* 合规误判申诉按钮 */}
            {hasComplianceIssue && !isStreaming && (
              <div className="flex items-center justify-end gap-2 mt-2 pt-2 border-t border-border/50">
                {isFeedbackSubmitted ? (
                  <span className="flex items-center gap-1 text-xs text-muted-foreground">
                    <CheckCircle size={12} />
                    已提交反馈
                  </span>
                ) : (
                  <button
                    onClick={handleReportCompliance}
                    className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
                    title="如果认为以上内容被误拦截，可提交反馈"
                  >
                    <ThumbsDown size={12} />
                    {hasL1Blocked ? '原始回复有误' : '此内容无风险'}
                  </button>
                )}
              </div>
            )}

            {/* Token 用量统计 */}
            {!isStreaming && !error && (message.promptTokens !== undefined || message.completionTokens !== undefined) && (
              <div className="flex items-center justify-end mt-2 pt-2 border-t border-border/50">
                <span className="text-xs text-muted-foreground">
                  {message.promptTokens !== undefined && `Input ${message.promptTokens}`}
                  {message.promptTokens !== undefined && message.completionTokens !== undefined && ' · '}
                  {message.completionTokens !== undefined && `Output ${message.completionTokens}`}
                </span>
              </div>
            )}

            {/* 错误时的操作按钮 */}
            {error && (
              <div className="flex items-center gap-2 mt-2 pt-2 border-t border-destructive/20">
                <button
                  onClick={handleCopy}
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
                  title="复制已生成的内容"
                >
                  <Copy size={12} />
                  复制内容
                </button>
                {onRetry && (
                  <button
                    onClick={() => onRetry(id)}
                    className="flex items-center gap-1 text-xs text-primary hover:opacity-80 transition-opacity"
                    title="重新生成"
                  >
                    <RotateCcw size={12} />
                    重试
                  </button>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
