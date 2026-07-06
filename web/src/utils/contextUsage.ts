export const WARNING_THRESHOLD = 0.75
export const AUTO_COMPRESSION_THRESHOLD = 0.90

export type ColorState = 'normal' | 'warning' | 'critical'

export function colorState(ratio: number): ColorState {
  if (ratio >= AUTO_COMPRESSION_THRESHOLD) return 'critical'
  if (ratio >= WARNING_THRESHOLD) return 'warning'
  return 'normal'
}

export function abbrevK(n: number): string {
  const k = n / 1000
  return `${k >= 100 ? Math.round(k) : Math.round(k * 10) / 10}k`
}

export function formatContextLabel(used: number, max: number): string {
  return `${abbrevK(used)} / ${abbrevK(max)} 已用上下文`
}
