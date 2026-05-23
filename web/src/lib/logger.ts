/**
 * 日志封装工具。
 * 开发环境输出到控制台，生产环境静默（避免泄露内部信息）。
 */

const isDev = import.meta.env.DEV

export const logger = {
  error: (...args: unknown[]): void => {
    if (isDev) console.error(...args)
  },
  warn: (...args: unknown[]): void => {
    if (isDev) console.warn(...args)
  },
  log: (...args: unknown[]): void => {
    if (isDev) console.log(...args)
  },
}
