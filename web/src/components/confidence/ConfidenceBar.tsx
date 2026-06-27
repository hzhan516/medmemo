import { useState } from 'react'
import { ChevronDown, ChevronUp } from 'lucide-react'
import { type ConfidenceResult, type ConfidenceBarMode, type ConfidenceLevel, levelConfig, levelSuggestion } from './types'
import { ConfidencePanel } from './ConfidencePanel'
import { ConfidenceFollowup } from './ConfidenceFollowup'

// 合法等级集合，用于防御非法 level 值
const validLevels: Set<ConfidenceLevel> = new Set(['A', 'B', 'C', 'D', 'E'])

function safeLevel(level: ConfidenceLevel | undefined | ''): ConfidenceLevel {
  return level && validLevels.has(level) ? level : 'E'
}

interface ConfidenceBarProps {
  result: ConfidenceResult
  mode?: ConfidenceBarMode
  onModeChange?: (mode: ConfidenceBarMode) => void
  onFollowupClick?: (text: string) => void
}

/**
 * 置信度条组件（TASK-065）。
 *
 * 展示位置：每条 AI 回答底部。
 * 三种模式：
 * - expanded：展开显示完整文本（首次使用默认）
 * - compact：仅等级+分数（熟悉后自动折叠）
 * - hidden：完全隐藏（设置中可关闭）
 *
 * 五级颜色：绿(A)/蓝(B)/黄(C)/橙(D)/红(E)。
 */
export function ConfidenceBar({
  result,
  mode = 'compact',
  onModeChange,
  onFollowupClick,
}: ConfidenceBarProps) {
  const [panelOpen, setPanelOpen] = useState(false)
  const level = safeLevel(result.level)
  const config = levelConfig[level]

  if (mode === 'hidden') {
    return null
  }

  const isLowConfidence = level === 'C' || level === 'D' || level === 'E'

  const handleBarClick = () => {
    setPanelOpen((prev) => !prev)
  }

  const handleSwitchMode = (newMode: ConfidenceBarMode) => {
    onModeChange?.(newMode)
  }

  return (
    <div className="mt-2 select-none">
      {/* 置信度条 */}
      <button
        onClick={handleBarClick}
        className="w-full text-left group"
        aria-expanded={panelOpen}
        aria-label={`置信度: ${config.label}(${result.overallScore}%)`}
      >
        <div
          className="flex items-center gap-2 px-3 py-1.5 rounded-md transition-colors duration-200 hover:bg-black/5 dark:hover:bg-white/5"
          style={{ borderLeft: `3px solid ${config.color}` }}
        >
          {/* 颜色指示条 */}
          <div
            className="w-2 h-2 rounded-full shrink-0"
            style={{ backgroundColor: config.color }}
          />

          {mode === 'expanded' ? (
            <span className="text-xs text-gray-600 dark:text-gray-400 truncate">
              ── 置信度: {config.icon} {config.label}({result.overallScore}%) · {result.explanation} · {levelSuggestion[level]} ──
            </span>
          ) : (
            <span className="text-xs text-gray-600 dark:text-gray-400">
              置信度: {config.icon} {config.label}({result.overallScore}%)
            </span>
          )}

          <span className="ml-auto text-gray-400 dark:text-gray-600 transition-transform duration-200 group-hover:text-gray-600 dark:group-hover:text-gray-400">
            {panelOpen ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
          </span>
        </div>
      </button>

      {/* 展开面板 */}
      <div
        className={`overflow-hidden transition-all duration-200 ease-in-out ${
          panelOpen ? 'max-h-[300px] opacity-100 mt-1' : 'max-h-0 opacity-0'
        }`}
      >
        <ConfidencePanel
          result={{ ...result, level }}
          onSwitchMode={handleSwitchMode}
          currentMode={mode}
        />
      </div>

      {/* 低置信度追问建议 */}
      {isLowConfidence && (
        <ConfidenceFollowup
          result={result}
          onQuestionClick={onFollowupClick}
        />
      )}
    </div>
  )
}
