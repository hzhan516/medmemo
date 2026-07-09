import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

// hoisted mock setup
const {
  mockWindowMinimise,
  mockWindowToggleMaximise,
  mockWindowIsMaximised,
  mockQuit,
} = vi.hoisted(() => ({
  mockWindowMinimise: vi.fn(),
  mockWindowToggleMaximise: vi.fn(),
  mockWindowIsMaximised: vi.fn().mockResolvedValue(false),
  mockQuit: vi.fn(),
}))

vi.mock('@wails/runtime/runtime', () => ({
  WindowMinimise: mockWindowMinimise,
  WindowToggleMaximise: mockWindowToggleMaximise,
  WindowIsMaximised: mockWindowIsMaximised,
  Quit: mockQuit,
}))

import { WindowControls } from './WindowControls'

describe('WindowControls', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockWindowIsMaximised.mockResolvedValue(false)
  })

  it('renders three window control buttons', () => {
    render(<WindowControls />)

    expect(screen.getByRole('button', { name: /最小化/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /最大化/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /关闭/i })).toBeInTheDocument()
  })

  it('calls WindowMinimise on minimize click', async () => {
    const user = userEvent.setup()
    render(<WindowControls />)

    await user.click(screen.getByRole('button', { name: /最小化窗口/i }))
    expect(mockWindowMinimise).toHaveBeenCalledTimes(1)
  })

  it('calls WindowToggleMaximise on maximize click', async () => {
    const user = userEvent.setup()
    render(<WindowControls />)

    await user.click(screen.getByRole('button', { name: /最大化窗口/i }))
    expect(mockWindowToggleMaximise).toHaveBeenCalledTimes(1)
  })

  it('calls Quit on close click', async () => {
    const user = userEvent.setup()
    render(<WindowControls />)

    await user.click(screen.getByRole('button', { name: /关闭应用/i }))
    expect(mockQuit).toHaveBeenCalledTimes(1)
  })

  it('shows restore aria-label when maximised', async () => {
    mockWindowIsMaximised.mockResolvedValueOnce(true)

    render(<WindowControls />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /还原窗口/i })).toBeInTheDocument()
    })
  })

  it('shows maximize aria-label when not maximised', async () => {
    mockWindowIsMaximised.mockResolvedValueOnce(false)

    render(<WindowControls />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /最大化窗口/i })).toBeInTheDocument()
    })
  })

  it('all buttons have accessible names', () => {
    render(<WindowControls />)

    const buttons = screen.getAllByRole('button')
    expect(buttons).toHaveLength(3)
    for (const btn of buttons) {
      expect(btn).toHaveAccessibleName()
    }
  })
})
