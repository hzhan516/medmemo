import { useCallback } from 'react'
import { AlertTriangle, Phone, MessageCircle, ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface EmergencyAlertModalProps {
  open: boolean
  message: string
  action: string
  onContinue: () => void
  onNotEmergency: () => void
}

/**
 * A级紧急症状全屏弹窗组件。
 *
 * 检测到可能危及生命的紧急症状时展示，不可通过点击背景或 ESC 关闭。
 * 提供「拨打 120」「继续咨询」「非紧急情况」三个操作选项。
 */
export function EmergencyAlertModal({
  open,
  message,
  action,
  onContinue,
  onNotEmergency,
}: EmergencyAlertModalProps) {
  const handleCall120 = useCallback(() => {
    // 桌面端尝试唤起默认电话应用，失败则复制号码到剪贴板
    const telUrl = 'tel:120'
    const iframe = document.createElement('iframe')
    iframe.style.display = 'none'
    document.body.appendChild(iframe)
    try {
      iframe.contentWindow?.location.assign(telUrl)
    } catch {
      navigator.clipboard.writeText('120').catch(() => {})
    }
    setTimeout(() => document.body.removeChild(iframe), 1000)
  }, [])

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center bg-red-600/90 backdrop-blur-sm"
      onClick={(e) => e.stopPropagation()}
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="emergency-title"
    >
      <div className="w-full max-w-lg mx-4 flex flex-col rounded-2xl bg-background border-2 border-red-500 shadow-2xl overflow-hidden">
        {/* 头部 */}
        <div className="flex items-center gap-3 px-6 py-5 border-b border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-950/50">
          <div className="w-12 h-12 rounded-full bg-red-100 dark:bg-red-900 flex items-center justify-center shrink-0">
            <AlertTriangle className="w-7 h-7 text-red-600 dark:text-red-400" />
          </div>
          <div className="flex-1">
            <h2
              id="emergency-title"
              className="text-xl font-bold text-red-700 dark:text-red-300"
            >
              ⚠️ 检测到紧急症状
            </h2>
            <p className="text-sm text-red-600/80 dark:text-red-400/80 mt-0.5">
              {action}
            </p>
          </div>
        </div>

        {/* 内容 */}
        <div className="px-6 py-5">
          <div className="p-4 rounded-xl bg-red-50 dark:bg-red-950/40 border border-red-200 dark:border-red-800">
            <p className="text-base text-foreground leading-relaxed">{message}</p>
            <p className="text-sm text-red-600 dark:text-red-400 mt-2 font-medium">
              如遇紧急症状，请优先拨打急救电话或前往最近的急诊科室，切勿依赖本工具进行判断或延误救治。
            </p>
          </div>
        </div>

        {/* 操作按钮 */}
        <div className="flex flex-col gap-3 px-6 pb-6">
          <Button
            variant="default"
            size="lg"
            onClick={handleCall120}
            className="w-full gap-2 bg-red-600 hover:bg-red-700 text-white"
          >
            <Phone size={18} />
            拨打 120
          </Button>

          <div className="flex gap-3">
            <Button
              variant="outline"
              size="default"
              onClick={onContinue}
              className="flex-1 gap-1.5"
            >
              <MessageCircle size={16} />
              继续咨询
            </Button>
            <Button
              variant="ghost"
              size="default"
              onClick={onNotEmergency}
              className="flex-1 gap-1.5 text-muted-foreground hover:text-foreground"
            >
              <ShieldAlert size={16} />
              非紧急情况
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
