import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@/test/render'
import userEvent from '@testing-library/user-event'
import { UpdateModal } from './UpdateModal'
import type { UpdateInfo } from '@/hooks/useUpdate'

const baseInfo: UpdateInfo = {
  version: 'v1.1.9',
  display_version: 'v1.1.9',
  name: 'v1.1.9 更新',
  body: '修复跨平台自动更新',
  published_at: '2026-06-26T00:00:00Z',
  mandatory: false,
  channel: 'stable',
  prerelease: false,
  prerelease_label: '',
  build_number: '',
}

function renderModal(props: Partial<Parameters<typeof UpdateModal>[0]> = {}) {
  return render(
    <UpdateModal
      info={baseInfo}
      isDownloading={false}
      isRestarting={false}
      downloadProgress={null}
      downloadPath=""
      error=""
      onDownload={vi.fn()}
      onApply={vi.fn()}
      onSkip={vi.fn()}
      onDismiss={vi.fn()}
      onOpenDownloadPage={vi.fn()}
      {...props}
    />
  )
}

describe('UpdateModal', () => {
  it('下载完成后显示“立即重启完成更新”', () => {
    renderModal({ downloadPath: '/tmp/MedMemo.AppImage' })
    expect(screen.getByText('下载完成，立即重启完成更新')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /立即重启完成更新/ })).toBeInTheDocument()
  })

  it('重启中按钮禁用并显示“重启中...”', async () => {
    const onApply = vi.fn()
    renderModal({
      downloadPath: '/tmp/MedMemo.AppImage',
      isRestarting: true,
      onApply,
    })

    const button = screen.getByRole('button', { name: '重启中...' })
    expect(button).toBeDisabled()

    await userEvent.click(button)
    expect(onApply).not.toHaveBeenCalled()
  })

  it('macOS 授权失败时显示手动安装提示', () => {
    renderModal({
      downloadPath: '/tmp/MedMemo.dmg',
      error: 'manual install required: open /Users/alice/Downloads/MedMemo.dmg',
    })

    expect(screen.getByText(/macOS 自动替换需要管理员授权/)).toBeInTheDocument()
    expect(screen.getByText(/手动将 MedMemo.app 拖拽到 Applications/)).toBeInTheDocument()
  })
})
