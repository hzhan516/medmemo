import { useState, useEffect, useCallback } from 'react'
import { logger } from '@/lib/logger'
import { useWails } from './useWails'
import { useSettingsStore } from '@/stores/settingsStore'
import { entity } from '@wails/go/models'

export type VersionNote = entity.VersionNote

/**
 * 版本提示检测 Hook。
 * 启动时获取当前版本和版本提示数据，判断是否需要展示弹窗。
 * 开发环境（version === "dev"）跳过提示，避免干扰调试。
 */
export function useVersionNotes() {
  const { getVersion, getVersionNotes } = useWails()
  const lastSeen = useSettingsStore((s) => s.lastSeenVersionNotes)
  const setLastSeen = useSettingsStore((s) => s.setLastSeenVersionNotes)

  const [notes, setNotes] = useState<VersionNote[]>([])
  const [currentVersion, setCurrentVersion] = useState('')
  const [shouldShow, setShouldShow] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false

    async function init() {
      try {
        const [version, versionNotes] = await Promise.all([
          getVersion(),
          getVersionNotes(),
        ])
        if (cancelled) return

        setCurrentVersion(version)
        setNotes(versionNotes)

        // 开发环境跳过提示
        if (version === 'dev') {
          setShouldShow(false)
          return
        }

        // 首次打开（lastSeen 为空）或版本更新后
        if (!lastSeen || lastSeen !== version) {
          setShouldShow(true)
        }
      } catch (err) {
        logger.error('Failed to load version notes:', err)
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    init()
    return () => {
      cancelled = true
    }
  }, [getVersion, getVersionNotes, lastSeen])

  const dismiss = useCallback(() => {
    setLastSeen(currentVersion)
    setShouldShow(false)
  }, [currentVersion, setLastSeen])

  return { notes, currentVersion, shouldShow, loading, dismiss }
}
