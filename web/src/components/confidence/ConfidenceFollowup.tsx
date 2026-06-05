import { useState } from 'react'
import { Lightbulb, AlertTriangle, Ban, X } from 'lucide-react'
import { type ConfidenceResult, type ConfidenceLevel } from './types'

interface ConfidenceFollowupProps {
  result: ConfidenceResult
  onQuestionClick?: (text: string) => void
}

/**
 * 低置信度追问建议组件（TASK-067）。
 *
 * 触发条件：
 * - C 级 (50-69%)：💡 提示块，列出缺失信息帮助更准确判断
 * - D 级 (30-49%)：❗ 警告块，建议尽快就医
 * - E 级 (0-29%)：🚫 危险块，必须就医
 *
 * 缺失信息项为可点击按钮，点击后自动填入输入框。
 */
export function ConfidenceFollowup({ result, onQuestionClick }: ConfidenceFollowupProps) {
  const [dismissed, setDismissed] = useState(false)

  if (dismissed || result.missingInfo.length === 0) {
    return null
  }

  const { level, missingInfo } = result

  const handleQuestionClick = (item: string) => {
    // 将缺失信息转换为追问问题格式
    const questionMap: Record<string, string> = {
      '疼痛持续时间': '症状持续多久了？',
      '既往病史': '您有什么既往病史吗？',
      '过敏史': '您有过敏史吗？',
      '用药情况': '您目前在服用什么药物？',
      '是否发热': '您有发热吗？体温多少？',
    }
    const text = questionMap[item] || `请补充一下${item}的信息`
    onQuestionClick?.(text)
  }

  const handleDismiss = () => {
    setDismissed(true)
  }

  return (
    <div className={`mt-2 rounded-lg p-3 text-sm ${getContainerClass(level)}`}>
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2">
          {getIcon(level)}
          <span className="font-medium">{getTitle(level)}</span>
        </div>
        <button
          onClick={handleDismiss}
          className="shrink-0 p-0.5 rounded hover:bg-black/5 dark:hover:bg-white/10 transition-colors"
          aria-label="关闭追问建议"
        >
          <X size={14} />
        </button>
      </div>

      <p className="mt-1.5 text-xs opacity-90">{getDescription(level)}</p>

      <div className="mt-2 flex flex-wrap gap-2">
        {missingInfo.map((item) => (
          <button
            key={item}
            onClick={() => handleQuestionClick(item)}
            className={`text-xs px-2.5 py-1 rounded-full border transition-colors ${getButtonClass(level)}`}
          >
            {item}
          </button>
        ))}
      </div>
    </div>
  )
}

function getContainerClass(level: ConfidenceLevel): string {
  switch (level) {
    case 'C':
      return 'bg-yellow-50 border border-yellow-200 text-yellow-800 dark:bg-yellow-900/20 dark:border-yellow-800/40 dark:text-yellow-300'
    case 'D':
      return 'bg-orange-50 border border-orange-200 text-orange-800 dark:bg-orange-900/20 dark:border-orange-800/40 dark:text-orange-300'
    case 'E':
      return 'bg-red-50 border border-red-200 text-red-800 dark:bg-red-900/20 dark:border-red-800/40 dark:text-red-300'
    default:
      return ''
  }
}

function getIcon(level: ConfidenceLevel) {
  switch (level) {
    case 'C':
      return <Lightbulb size={16} className="text-yellow-600 dark:text-yellow-400" />
    case 'D':
      return <AlertTriangle size={16} className="text-orange-600 dark:text-orange-400" />
    case 'E':
      return <Ban size={16} className="text-red-600 dark:text-red-400" />
    default:
      return null
  }
}

function getTitle(level: ConfidenceLevel): string {
  switch (level) {
    case 'C':
      return '以下信息可以帮助我更准确判断'
    case 'D':
      return '信息不足，建议尽快就医'
    case 'E':
      return 'AI 无法提供有效建议'
    default:
      return ''
  }
}

function getDescription(level: ConfidenceLevel): string {
  switch (level) {
    case 'C':
      return '补充这些信息后，我可以给出更准确的建议。'
    case 'D':
      return '不要自行判断，请尽快就医寻求专业帮助。'
    case 'E':
      return '请立即就医，AI 无法根据现有信息提供有效建议。'
    default:
      return ''
  }
}

function getButtonClass(level: ConfidenceLevel): string {
  switch (level) {
    case 'C':
      return 'border-yellow-300 bg-yellow-100 hover:bg-yellow-200 dark:border-yellow-700 dark:bg-yellow-900/30 dark:hover:bg-yellow-900/50'
    case 'D':
      return 'border-orange-300 bg-orange-100 hover:bg-orange-200 dark:border-orange-700 dark:bg-orange-900/30 dark:hover:bg-orange-900/50'
    case 'E':
      return 'border-red-300 bg-red-100 hover:bg-red-200 dark:border-red-700 dark:bg-red-900/30 dark:hover:bg-red-900/50'
    default:
      return ''
  }
}
