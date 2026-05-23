import { AlertTriangle, CheckCircle, ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface EmergencyWarningBannerProps {
  message: string
  onAcknowledge: () => void
  onNotEmergency: () => void
}

/**
 * B级紧急症状警告横幅组件。
 *
 * 位于输入区域上方，检测到需尽快就医的症状时展示。
 * 用户需点击「我已了解」后方可继续发送消息。
 */
export function EmergencyWarningBanner({
  message,
  onAcknowledge,
  onNotEmergency,
}: EmergencyWarningBannerProps) {
  return (
    <div className="shrink-0 px-4 py-3 bg-red-50 dark:bg-red-950/40 border-l-4 border-red-500 border-y border-y-red-200 dark:border-y-red-900">
      <div className="max-w-3xl mx-auto flex items-start gap-3">
        <AlertTriangle
          size={18}
          className="shrink-0 mt-0.5 text-red-600 dark:text-red-400"
        />
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-red-800 dark:text-red-200">
            {message}
          </p>
          <p className="text-xs text-red-600/80 dark:text-red-400/80 mt-1">
            建议尽快前往医院就诊。本工具仅提供健康信息参考，不诊断、不治疗。
          </p>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <Button
            variant="outline"
            size="sm"
            onClick={onAcknowledge}
            className="gap-1.5 border-red-300 dark:border-red-700 text-red-700 dark:text-red-300 hover:bg-red-100 dark:hover:bg-red-900"
          >
            <CheckCircle size={14} />
            我已了解
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={onNotEmergency}
            className="gap-1.5 text-muted-foreground hover:text-foreground"
          >
            <ShieldAlert size={14} />
            非紧急
          </Button>
        </div>
      </div>
    </div>
  )
}
