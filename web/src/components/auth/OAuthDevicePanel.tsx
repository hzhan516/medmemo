import { useState, useCallback, useEffect, useRef } from 'react'
import { Smartphone, CheckCircle2, XCircle, Loader2, Copy, ExternalLink } from 'lucide-react'
import { StartOAuthDeviceFlow, CancelOAuthDeviceFlow, GetOAuthDeviceFlowStatus } from '@wails/go/main/WailsApp'
import { BrowserOpenURL } from '@wails/runtime'
import type { AuthMethodDetectStatus } from '@/types/provider'

interface OAuthDevicePanelProps {
  status: AuthMethodDetectStatus | undefined
}

/**
 * OAuth Device Flow 配置面板。
 * 启动 Device Flow、展示用户码、轮询状态。
 */
export function OAuthDevicePanel({ status }: OAuthDevicePanelProps) {
  const [deviceCode, setDeviceCode] = useState<string | null>(null)
  const [userCode, setUserCode] = useState('')
  const [verificationURI, setVerificationURI] = useState('')
  const [polling, setPolling] = useState(false)
  const [flowStatus, setFlowStatus] = useState<string>('')
  const [error, setError] = useState<string | null>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const clearPolling = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current)
      intervalRef.current = null
    }
  }, [])

  useEffect(() => {
    return () => clearPolling()
  }, [clearPolling])

  const handleStart = useCallback(async () => {
    setError(null)
    setFlowStatus('')
    clearPolling()
    try {
      const result = await StartOAuthDeviceFlow('kimi')
      setDeviceCode(result.device_code)
      setUserCode(result.user_code)
      setVerificationURI(result.verification_uri)
      setPolling(true)

      // 开始轮询
      const intervalMs = (result.interval || 5) * 1000
      intervalRef.current = setInterval(async () => {
        try {
          const s = await GetOAuthDeviceFlowStatus(result.device_code)
          if (!s) {
            setFlowStatus('error')
            setPolling(false)
            clearPolling()
            return
          }
          setFlowStatus(s.status)
          if (s.status === 'success' || s.status === 'error' || s.status === 'cancelled') {
            setPolling(false)
            clearPolling()
            if (s.error) setError(s.error)
          }
        } catch {
          // 轮询异常继续
        }
      }, intervalMs)
    } catch (err) {
      setError(err instanceof Error ? err.message : '启动 Device Flow 失败')
    }
  }, [clearPolling])

  const handleCancel = useCallback(async () => {
    if (deviceCode) {
      await CancelOAuthDeviceFlow(deviceCode)
    }
    clearPolling()
    setDeviceCode(null)
    setUserCode('')
    setVerificationURI('')
    setPolling(false)
    setFlowStatus('')
  }, [deviceCode, clearPolling])

  const handleOpenURL = useCallback(() => {
    if (verificationURI) {
      BrowserOpenURL(verificationURI)
    }
  }, [verificationURI])

  const handleCopyCode = useCallback(async () => {
    if (userCode) {
      await navigator.clipboard.writeText(userCode)
    }
  }, [userCode])

  if (!status) {
    return (
      <div className="p-4 rounded-lg bg-muted/50 text-sm text-muted-foreground">
        尚未检测 OAuth 状态。
      </div>
    )
  }

  // Device Flow 进行中
  if (deviceCode) {
    return (
      <div className="space-y-4">
        <div className="p-4 rounded-lg border space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">设备授权码</span>
            <button
              onClick={handleCopyCode}
              className="flex items-center gap-1 text-xs text-primary hover:underline"
            >
              <Copy className="w-3 h-3" />
              复制
            </button>
          </div>
          <div className="text-center py-3 bg-muted rounded-lg">
            <code className="text-2xl font-bold tracking-widest">{userCode}</code>
          </div>
          <p className="text-xs text-muted-foreground">
            请在浏览器中访问下方链接并输入上方授权码：
          </p>
          <button
            onClick={handleOpenURL}
            className="w-full flex items-center justify-center gap-2 py-2 rounded-lg border text-sm hover:bg-accent transition-colors"
          >
            <ExternalLink className="w-4 h-4" />
            {verificationURI}
          </button>
        </div>

        <div className="flex items-center gap-2 text-xs">
          {polling ? (
            <>
              <Loader2 className="w-3 h-3 animate-spin text-primary" />
              <span className="text-muted-foreground">等待授权…</span>
            </>
          ) : flowStatus === 'success' ? (
            <>
              <CheckCircle2 className="w-3 h-3 text-green-500" />
              <span className="text-green-600">授权成功</span>
            </>
          ) : (
            <>
              <XCircle className="w-3 h-3 text-destructive" />
              <span className="text-destructive">{error || '授权失败'}</span>
            </>
          )}
        </div>

        <button
          onClick={handleCancel}
          className="w-full py-2 rounded-lg border text-sm hover:bg-accent transition-colors"
        >
          取消授权
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-3 p-3 rounded-lg border">
        <Smartphone className="w-5 h-5 text-muted-foreground shrink-0 mt-0.5" />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">OAuth Device Flow</span>
            {status.connected ? (
              <CheckCircle2 className="w-4 h-4 text-green-500" />
            ) : (
              <XCircle className="w-4 h-4 text-muted-foreground" />
            )}
          </div>
          <p className="text-xs text-muted-foreground mt-1">{status.detail}</p>
        </div>
      </div>

      {!status.connected && (
        <button
          onClick={handleStart}
          className="w-full py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors flex items-center justify-center gap-2"
        >
          <Smartphone className="w-4 h-4" />
          启动 Device Flow 授权
        </button>
      )}

      {status.connected && (
        <div className="p-3 rounded-lg bg-green-500/5 border border-green-500/20 text-xs text-green-700">
          已通过 OAuth Device Flow 授权，无需重复操作。
        </div>
      )}

      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  )
}
