import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useConversation } from './useConversation'
import { setMockHandlers, resetWailsMock } from '@/test/mocks/wails'
import { useProviderStore } from '@/stores/providerStore'
import { useSettingsStore } from '@/stores/settingsStore'
import type { ProviderConfig } from '@/types/provider'

describe('useConversation', () => {
  beforeEach(() => {
    resetWailsMock()
    setMockHandlers({})

    const provider: ProviderConfig = {
      id: 'prov_1',
      templateId: 'kimi',
      name: 'Kimi Test',
      apiHost: 'https://api.example.com',
      apiKey: 'test-key',
      modelId: 'kimi-lite',
      models: [{ id: 'kimi-lite', name: 'Kimi Lite', enabled: true }],
      authMethod: 'api_key',
      authParams: {},
      enabled: true,
      group: 'test',
      sortOrder: 0,
      createdAt: Date.now(),
      updatedAt: Date.now(),
      needsApiKey: false,
    }

    useProviderStore.setState({ providers: [provider] })
    useSettingsStore.setState({
      activeProviderId: 'prov_1',
      activeModelId: 'kimi-lite',
      providerHealthStatus: { prov_1: 'green' },
    })
  })

  it('sendMessage passes ai_message_id matching the local assistant message', async () => {
    let capturedRequest: Record<string, unknown> | null = null
    setMockHandlers({
      CreateConversation: vi.fn(() => Promise.resolve('conv_test_1')),
      CheckEmergency: vi.fn(() => Promise.resolve({ level: 'none', message: '', action: '' })),
      SendMessageStream: vi.fn((req: Record<string, unknown>) => {
        capturedRequest = req
        return Promise.resolve()
      }),
    })

    const { result } = renderHook(() => useConversation())

    await act(async () => {
      await result.current.sendMessage('你好')
    })

    await waitFor(() => {
      expect(capturedRequest).not.toBeNull()
    })

    expect(capturedRequest).toHaveProperty('ai_message_id')
    expect(typeof capturedRequest!.ai_message_id).toBe('string')
    expect((capturedRequest!.ai_message_id as string).length).toBeGreaterThan(0)

    const messages = result.current.messages
    const assistantMsg = messages.find((m) => m.role === 'assistant')
    expect(assistantMsg).toBeDefined()
    expect(assistantMsg!.id).toBe(capturedRequest!.ai_message_id)
  })
})
