/**
 * 合并 Tailwind CSS 类名，过滤掉 falsy 值。
 * 注意：本实现不做类名冲突合并，调用方需自行保证无冲突。
 */
export function cn(...inputs: (string | undefined | null | false)[]): string {
  return inputs.filter(Boolean).join(' ')
}
