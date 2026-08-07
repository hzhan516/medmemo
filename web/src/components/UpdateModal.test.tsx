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

  it('macOS 授权失败时显示手动安装提示', async () => {
    vi.resetModules()
    const originalPlatform = navigator.platform
    Object.defineProperty(navigator, 'platform', { value: 'MacIntel', configurable: true })

    try {
      const { UpdateModal: MacUpdateModal } = await import('./UpdateModal')
      render(
        <MacUpdateModal
          info={baseInfo}
          isDownloading={false}
          isRestarting={false}
          downloadProgress={null}
          downloadPath="/tmp/MedMemo.dmg"
          error="manual install required: open /Users/alice/Downloads/MedMemo.dmg"
          onDownload={vi.fn()}
          onApply={vi.fn()}
          onSkip={vi.fn()}
          onDismiss={vi.fn()}
          onOpenDownloadPage={vi.fn()}
        />
      )

      expect(screen.getByText(/macOS 自动替换需要管理员授权/)).toBeInTheDocument()
      expect(screen.getByText(/手动将 MedMemo.app 拖拽到 Applications/)).toBeInTheDocument()
    } finally {
      Object.defineProperty(navigator, 'platform', { value: originalPlatform, configurable: true })
    }
  })

  it('Linux 包管理安装失败时显示命令行手动安装提示', async () => {
    vi.resetModules()
    const originalPlatform = navigator.platform
    Object.defineProperty(navigator, 'platform', { value: 'Linux x86_64', configurable: true })

    try {
      const { UpdateModal: LinuxUpdateModal } = await import('./UpdateModal')
      render(
        <LinuxUpdateModal
          info={baseInfo}
          isDownloading={false}
          isRestarting={false}
          downloadProgress={null}
          downloadPath=""
          error='manual install required: sudo dpkg -i "/tmp/MedMemo.deb"'
          onDownload={vi.fn()}
          onApply={vi.fn()}
          onSkip={vi.fn()}
          onDismiss={vi.fn()}
          onOpenDownloadPage={vi.fn()}
        />
      )

      expect(screen.getByText(/Linux 包管理安装需要手动执行/)).toBeInTheDocument()
      expect(screen.getByText(/sudo dpkg -i/)).toBeInTheDocument()
      expect(screen.getByText(/"\/tmp\/MedMemo.deb"/)).toBeInTheDocument()
    } finally {
      Object.defineProperty(navigator, 'platform', { value: originalPlatform, configurable: true })
    }
  })

  it('macOS 下“前往下载”按钮打开 release tag 页面', async () => {
    vi.resetModules()
    const originalPlatform = navigator.platform
    Object.defineProperty(navigator, 'platform', { value: 'MacIntel', configurable: true })

    try {
      const { UpdateModal: MacUpdateModal } = await import('./UpdateModal')
      const onOpenDownloadPage = vi.fn()
      const onDismiss = vi.fn()

      render(
        <MacUpdateModal
          info={baseInfo}
          isDownloading={false}
          isRestarting={false}
          downloadProgress={null}
          downloadPath=""
          error=""
          onDownload={vi.fn()}
          onApply={vi.fn()}
          onSkip={vi.fn()}
          onDismiss={onDismiss}
          onOpenDownloadPage={onOpenDownloadPage}
        />
      )

      await userEvent.click(screen.getByRole('button', { name: '前往下载' }))
      expect(onOpenDownloadPage).toHaveBeenCalledWith(
        `https://github.com/hzhan516/medmemo/releases/tag/${baseInfo.version}`
      )
      expect(onDismiss).toHaveBeenCalled()
    } finally {
      Object.defineProperty(navigator, 'platform', { value: originalPlatform, configurable: true })
    }
  })

  it('Linux unknown 安装方式引导前往 Release 页面', async () => {
    vi.resetModules()
    const originalPlatform = navigator.platform
    Object.defineProperty(navigator, 'platform', { value: 'Linux x86_64', configurable: true })

    try {
      const { UpdateModal: LinuxUpdateModal } = await import('./UpdateModal')
      const onOpenDownloadPage = vi.fn()

      render(
        <LinuxUpdateModal
          info={baseInfo}
          isDownloading={false}
          isRestarting={false}
          downloadProgress={null}
          downloadPath=""
          error="manual install required:"
          onDownload={vi.fn()}
          onApply={vi.fn()}
          onSkip={vi.fn()}
          onDismiss={vi.fn()}
          onOpenDownloadPage={onOpenDownloadPage}
        />
      )

      expect(screen.getByText(/无法确定当前 Linux 安装包类型/)).toBeInTheDocument()
      await userEvent.click(screen.getByRole('button', { name: '前往 Release 页面' }))
      expect(onOpenDownloadPage).toHaveBeenCalledWith(
        `https://github.com/hzhan516/medmemo/releases/tag/${baseInfo.version}`
      )
    } finally {
      Object.defineProperty(navigator, 'platform', { value: originalPlatform, configurable: true })
    }
  })

  it('Linux 手动安装命令支持一键复制', async () => {
    vi.resetModules()
    const originalPlatform = navigator.platform
    const originalClipboard = navigator.clipboard
    Object.defineProperty(navigator, 'platform', { value: 'Linux x86_64', configurable: true })
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    })

    try {
      const { UpdateModal: LinuxUpdateModal } = await import('./UpdateModal')
      render(
        <LinuxUpdateModal
          info={baseInfo}
          isDownloading={false}
          isRestarting={false}
          downloadProgress={null}
          downloadPath=""
          error='manual install required: sudo dpkg -i "/tmp/MedMemo.deb"'
          onDownload={vi.fn()}
          onApply={vi.fn()}
          onSkip={vi.fn()}
          onDismiss={vi.fn()}
          onOpenDownloadPage={vi.fn()}
        />
      )

      await userEvent.click(screen.getByRole('button', { name: '复制' }))
      expect(writeText).toHaveBeenCalledWith('sudo dpkg -i "/tmp/MedMemo.deb"')
    } finally {
      Object.defineProperty(navigator, 'platform', { value: originalPlatform, configurable: true })
      Object.defineProperty(navigator, 'clipboard', {
        value: originalClipboard,
        configurable: true,
      })
    }
  })
})
