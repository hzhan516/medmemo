import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { KnowledgePage } from './KnowledgePage'
import { setMockHandlers } from '@/test/mocks/wails'

describe('KnowledgePage', () => {
  beforeEach(() => {
    setMockHandlers({})
    vi.useFakeTimers({ shouldAdvanceTime: true })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders the knowledge base management page', () => {
    render(<KnowledgePage />)
    expect(screen.getByText('知识库管理')).toBeInTheDocument()
    expect(screen.getByText('导入文件')).toBeInTheDocument()
  })

  it('stops polling when import job reaches indexed_vector_unavailable', async () => {
    const jobCalls: string[] = []
    let callCount = 0
    setMockHandlers({
      SelectKnowledgeFile: vi.fn(() => Promise.resolve('/tmp/mock.md')),
      ImportKnowledgeFile: vi.fn(() =>
        Promise.resolve({ job_id: 'job_1', status: 'pending', total: 2, processed: 0 })
      ),
      ListKnowledgeDocuments: vi.fn(() => Promise.resolve([])),
      GetKnowledgeImportJob: vi.fn((jobID: string) => {
        callCount++
        jobCalls.push(jobID)
        // 前两次返回 pending，第三次返回 vector_unavailable 终止状态
        const status = callCount <= 2 ? 'pending' : 'indexed_vector_unavailable'
        return Promise.resolve({ job_id: jobID, status, total: 2, processed: 2 })
      }),
    })

    render(<KnowledgePage />)
    await userEvent.click(screen.getByText('导入文件'))

    await waitFor(() => {
      expect(callCount).toBeGreaterThanOrEqual(1)
    })

    // 推进到第三次轮询，状态变为终止
    await vi.advanceTimersByTimeAsync(2500)

    await waitFor(() => {
      expect(screen.getByText(/indexed_vector_unavailable/)).toBeInTheDocument()
    })

    const terminalCallCount = callCount

    // 继续推进较长时间，确认不再继续轮询
    await vi.advanceTimersByTimeAsync(5000)

    expect(callCount).toBe(terminalCallCount)
  })
})
