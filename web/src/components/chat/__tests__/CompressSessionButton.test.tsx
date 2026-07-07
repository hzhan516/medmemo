import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, act, waitFor } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { CompressSessionButton } from '@/components/chat/CompressSessionButton'
import { useChatStore } from '@/stores/chatStore'
import { useProviderStore } from '@/stores/providerStore'
import { useSettingsStore } from '@/stores/settingsStore'
import { resetWailsMock, setMockHandlers } from '@/test/mocks/wails'

const CONV_ID = 'conv-compress'
const PROVIDER_ID = 'provider-kimi'
const MODEL_ID = 'kimi-lite'

describe('CompressSessionButton', () => {
  beforeEach(() => {
    resetWailsMock()
    setMockHandlers({})
    useChatStore.setState({
      currentConversationId: CONV_ID,
      contextUsageMap: {},
    })
    useProviderStore.setState({ providers: [] })
    useSettingsStore.setState({ activeProviderId: null, activeModelId: null })
  })

  it('renders default label', () => {
    render(
      <CompressSessionButton
        conversationId={CONV_ID}
        providerId={PROVIDER_ID}
        modelId={MODEL_ID}
      />
    )
    expect(screen.getByRole('button', { name: '压缩当前会话' })).toBeInTheDocument()
  })

  it('is disabled, busy and shows loading text while compressing', () => {
    useChatStore.getState().setCompressing(CONV_ID, true)
    render(
      <CompressSessionButton
        conversationId={CONV_ID}
        providerId={PROVIDER_ID}
        modelId={MODEL_ID}
      />
    )

    const button = screen.getByRole('button', { name: '压缩中…' })
    expect(button).toBeDisabled()
    expect(button).toHaveAttribute('aria-busy', 'true')
  })

  it('calls compressSession with conversation, provider and model ids on click', async () => {
    const compressSpy = vi.fn().mockResolvedValue(undefined)
    setMockHandlers({ CompressSession: compressSpy })

    render(
      <CompressSessionButton
        conversationId={CONV_ID}
        providerId={PROVIDER_ID}
        modelId={MODEL_ID}
      />
    )

    await userEvent.click(screen.getByRole('button', { name: '压缩当前会话' }))

    expect(compressSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        conversationId: CONV_ID,
        providerId: PROVIDER_ID,
        modelId: MODEL_ID,
      })
    )
  })

  it('shows lastError in red text when compressSession rejects and clears it on success', async () => {
    const compressSpy = vi
      .fn()
      .mockRejectedValueOnce(new Error('压缩服务不可用'))
      .mockResolvedValueOnce(undefined)
    setMockHandlers({ CompressSession: compressSpy })

    render(
      <CompressSessionButton
        conversationId={CONV_ID}
        providerId={PROVIDER_ID}
        modelId={MODEL_ID}
      />
    )

    await userEvent.click(screen.getByRole('button', { name: '压缩当前会话' }))
    expect(await screen.findByText('压缩服务不可用')).toHaveClass('text-red-500')

    await userEvent.click(screen.getByRole('button', { name: '压缩当前会话' }))
    await waitFor(() => {
      expect(screen.queryByText('压缩服务不可用')).not.toBeInTheDocument()
    })
  })
})
