import { type ConfidenceResult, type ConfidenceBarMode, levelConfig, dimensionLabels } from './types'

interface ConfidencePanelProps {
  result: ConfidenceResult
  onSwitchMode: (mode: ConfidenceBarMode) => void
  currentMode: ConfidenceBarMode
}

/**
 * 置信度展开面板（TASK-066）。
 *
 * 展示内容：
 * - 综合评分（大字）
 * - 五维度进度条（各带分数）
 * - 知识来源详情
 * - "为什么不是100%"解释
 * - 缺失信息列表
 */
export function ConfidencePanel({ result, onSwitchMode, currentMode }: ConfidencePanelProps) {
  const config = levelConfig[result.level]

  const dimensions = Object.entries(result.breakdown) as [keyof typeof dimensionLabels, number][]

  return (
    <div className="bg-gray-50 dark:bg-gray-800/50 rounded-lg p-4 text-sm">
      {/* 综合评分 */}
      <div className="flex items-center gap-3 mb-4">
        <div
          className="w-12 h-12 rounded-full flex items-center justify-center text-white text-lg font-bold shrink-0"
          style={{ backgroundColor: config.color }}
        >
          {result.level}
        </div>
        <div>
          <div className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            {config.icon} {result.overallScore}% ({config.label})
          </div>
          <div className="text-xs text-gray-500 dark:text-gray-400">
            {result.explanation}
          </div>
        </div>
      </div>

      {/* 五维度进度条 */}
      <div className="space-y-3 mb-4">
        {dimensions.map(([key, score]) => (
          <DimensionBar
            key={key}
            label={dimensionLabels[key]}
            score={score}
            color={config.color}
          />
        ))}
      </div>

      {/* 缺失信息 */}
      {result.missingInfo.length > 0 && (
        <div className="mb-4 p-3 bg-amber-50 dark:bg-amber-900/20 rounded-md">
          <div className="text-xs font-medium text-amber-700 dark:text-amber-400 mb-1">
            💡 为什么不是 100%？
          </div>
          <div className="text-xs text-amber-600 dark:text-amber-300">
            缺少: {result.missingInfo.join('、')}
          </div>
        </div>
      )}

      {/* 模式切换 */}
      <div className="flex items-center gap-2 pt-3 border-t border-gray-200 dark:border-gray-700">
        <span className="text-xs text-gray-400 dark:text-gray-500">展示模式:</span>
        {(['expanded', 'compact', 'hidden'] as ConfidenceBarMode[]).map((m) => (
          <button
            key={m}
            onClick={() => onSwitchMode(m)}
            className={`text-xs px-2 py-0.5 rounded transition-colors ${
              currentMode === m
                ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                : 'text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700'
            }`}
          >
            {m === 'expanded' ? '展开' : m === 'compact' ? '紧凑' : '隐藏'}
          </button>
        ))}
      </div>
    </div>
  )
}

/**
 * 单维度进度条子组件。
 */
function DimensionBar({
  label,
  score,
  color,
}: {
  label: string
  score: number
  color: string
}) {
  const percentage = Math.min(100, Math.max(0, score))

  return (
    <div>
      <div className="flex justify-between text-xs mb-1">
        <span className="text-gray-600 dark:text-gray-400">{label}</span>
        <span className="text-gray-900 dark:text-gray-200 font-medium">{percentage.toFixed(0)}%</span>
      </div>
      <div className="h-1.5 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
        <div
          className="h-full rounded-full transition-all duration-300"
          style={{
            width: `${percentage}%`,
            backgroundColor: color,
            opacity: 0.8,
          }}
        />
      </div>
    </div>
  )
}
