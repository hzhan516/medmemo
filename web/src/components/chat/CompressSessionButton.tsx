import { useChatStore } from '@/stores/chatStore'
import { compressSession } from '@/services/contextUsageService'
import { Button } from '@/components/ui/button'

interface Props {
  conversationId: string
  providerId: string
  modelId: string
}

/**
 * 压缩当前会话按钮。
 *
 * - 调用上下文压缩服务，减少会话 token 占用。
 * - 压缩过程中禁用按钮并显示加载状态。
 * - 压缩完成后刷新上下文用量显示。
 */
export function CompressSessionButton({ conversationId, providerId, modelId }: Props) {
  const isCompressing = useChatStore(
    (s) => s.contextUsageMap[conversationId]?.isCompressing ?? false
  )

  async function handleClick() {
    try {
      await compressSession({ conversationId, providerId, modelId })
    } catch (e) {
      // 当前项目未提供全局 toast，先通过控制台输出错误；后续可替换为统一通知机制
      console.error('压缩会话失败:', e)
    }
  }

  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      onClick={handleClick}
      disabled={isCompressing}
      aria-busy={isCompressing}
    >
      {isCompressing ? '压缩中…' : '压缩当前会话'}
    </Button>
  )
}
