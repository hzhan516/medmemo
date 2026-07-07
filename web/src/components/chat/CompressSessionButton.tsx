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
  const lastError = useChatStore(
    (s) => s.contextUsageMap[conversationId]?.lastError
  )

  async function handleClick() {
    try {
      await compressSession({ conversationId, providerId, modelId })
      useChatStore.getState().setContextUsage(conversationId, { lastError: undefined })
    } catch (e) {
      useChatStore.getState().setContextUsage(conversationId, {
        lastError: e instanceof Error ? e.message : '压缩失败，请稍后重试',
      })
    }
  }

  return (
    <div className="flex flex-col items-end gap-1">
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
      {lastError && (
        <span className="text-red-500 text-xs max-w-[200px] text-right">{lastError}</span>
      )}
    </div>
  )
}
