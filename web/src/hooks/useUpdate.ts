import { useState, useCallback, useEffect } from 'react'
import { EventsOn } from '@wails/runtime/runtime'
import { useWails } from './useWails'

export interface UpdateInfo {
  version: string
  display_version: string
  name: string
  body: string
  published_at: string
  mandatory: boolean
  channel: string
  prerelease: boolean
  prerelease_label: string
  build_number: string
}

export interface DownloadProgress {
  downloaded: number
  total: number
}

/**
 * 自动更新 Hook。
 * 封装更新检测、下载、安装流程，监听后端推送的 update:available / update:progress 事件。
 */
export function useUpdate() {
  const { checkUpdate, downloadUpdate, applyUpdate, skipUpdateVersion, openDownloadURL } = useWails()

  const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null)
  const [downloadProgress, setDownloadProgress] = useState<DownloadProgress | null>(null)
  const [isDownloading, setIsDownloading] = useState(false)
  const [downloadPath, setDownloadPath] = useState<string>('')
  const [error, setError] = useState<string>('')

  // 监听后端推送的更新可用事件
  useEffect(() => {
    const unbindAvailable = EventsOn('update:available', (payload: UpdateInfo) => {
      setUpdateInfo(payload)
    })
    const unbindProgress = EventsOn('update:progress', (payload: DownloadProgress) => {
      setDownloadProgress(payload)
    })

    return () => {
      unbindAvailable()
      unbindProgress()
    }
  }, [])

  /**
   * 手动检测更新。
   */
  const doCheckUpdate = useCallback(async () => {
    setError('')
    try {
      const info = await checkUpdate()
      if (info) {
        setUpdateInfo(info)
      }
      return info
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg)
      return null
    }
  }, [checkUpdate])

  /**
   * 下载更新。
   */
  const doDownload = useCallback(async () => {
    if (!updateInfo) return
    setError('')
    setIsDownloading(true)
    setDownloadProgress(null)
    try {
      const path = await downloadUpdate({ version: updateInfo.version })
      setDownloadPath(path)
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg)
    } finally {
      setIsDownloading(false)
    }
  }, [updateInfo, downloadUpdate])

  /**
   * 应用更新（安装）。
   */
  const doApply = useCallback(async () => {
    if (!downloadPath) return
    setError('')
    try {
      await applyUpdate(downloadPath)
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg)
    }
  }, [downloadPath, applyUpdate])

  /**
   * 跳过当前版本。
   */
  const doSkip = useCallback(async () => {
    if (!updateInfo) return
    try {
      await skipUpdateVersion(updateInfo.version)
      setUpdateInfo(null)
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg)
    }
  }, [updateInfo, skipUpdateVersion])

  /**
   * 关闭更新提示（本次会话不再展示）。
   */
  const dismissUpdate = useCallback(() => {
    setUpdateInfo(null)
  }, [])

  /**
   * 通过浏览器打开下载页面（macOS / 手动安装场景）。
   */
  const openDownloadPage = useCallback(async (url: string) => {
    try {
      await openDownloadURL(url)
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg)
    }
  }, [openDownloadURL])

  return {
    updateInfo,
    downloadProgress,
    isDownloading,
    downloadPath,
    error,
    doCheckUpdate,
    doDownload,
    doApply,
    doSkip,
    dismissUpdate,
    openDownloadPage,
  }
}
