import { describe, it, expect } from 'vitest'
import { render, screen } from '@/test/render'
import { KnowledgePage } from './KnowledgePage'

describe('KnowledgePage', () => {
  it('renders the knowledge base management page', () => {
    render(<KnowledgePage />)
    expect(screen.getByText('知识库管理')).toBeInTheDocument()
    expect(screen.getByText('导入文件')).toBeInTheDocument()
  })
})
