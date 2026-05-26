/**
 * 回答置信度机制类型定义（前端）。
 * 与后端 entity.ConfidenceResult 对应。
 */

export type ConfidenceLevel = 'A' | 'B' | 'C' | 'D' | 'E'

export interface ConfidenceBreakdown {
  knowledge_source: number
  reasoning: number
  context: number
  history: number
  uncertainty: number
}

export interface ConfidenceResult {
  overallScore: number
  level: ConfidenceLevel
  breakdown: ConfidenceBreakdown
  explanation: string
  suggestion: string
  missingInfo: string[]
}

export type ConfidenceBarMode = 'expanded' | 'compact' | 'hidden'

export const levelConfig: Record<ConfidenceLevel, {
  color: string
  label: string
  icon: string
}> = {
  A: { color: '#27ae60', label: '高度确信', icon: '✅' },
  B: { color: '#3498db', label: '较为确信', icon: '👍' },
  C: { color: '#f39c12', label: '中等确信', icon: '⚠️' },
  D: { color: '#e67e22', label: '低确信', icon: '❗' },
  E: { color: '#e74c3c', label: '不确定', icon: '🚫' },
}

export const levelSuggestion: Record<ConfidenceLevel, string> = {
  A: '可作为参考',
  B: '建议与医生讨论',
  C: '仅供参考，强烈建议咨询医生',
  D: '建议尽快就医',
  E: '必须就医',
}

export const dimensionLabels: Record<keyof ConfidenceBreakdown, string> = {
  knowledge_source: '知识来源可靠性',
  reasoning: '推理链完整性',
  context: '上下文完整度',
  history: '历史准确性',
  uncertainty: '不确定性表达',
}
