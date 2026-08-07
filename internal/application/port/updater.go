// Package port 定义更新服务的外部依赖接口（消费者端声明）。
// 遵循 Clean Architecture 规范：application 层定义接口，adapter/infrastructure 层提供实现。
package port

import (
	"context"
	"io"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
)

// Updater 定义更新信息获取与资产下载的端口。
type Updater interface {
	// FetchLatest 查询远程最新版本信息。
	// channel 为 stable 时过滤掉 prerelease，为 beta 时包含全部。
	FetchLatest(ctx context.Context, channel models.UpdateChannel) (*entity.UpdateInfo, error)

	// FetchByTag 根据指定 tag 查询 Release 信息，用于下载用户点击的特定版本。
	FetchByTag(ctx context.Context, tag string) (*entity.UpdateInfo, error)

	// Download 下载指定 URL 的资产到本地路径，并通过 progress 回调推送进度。
	// progress 参数可为 nil，表示不接收进度通知。
	Download(ctx context.Context, url, destPath string, progress func(downloaded, total int64)) error

	// VerifyChecksum 校验本地文件 SHA256 是否与预期值匹配。
	VerifyChecksum(path, expectedSHA256 string) error
}

// Installer 定义平台特定的更新安装端口。
type Installer interface {
	// Install 执行平台特定的安装/替换逻辑。
	// assetPath 为已下载且校验通过的资产文件路径。
	// 返回安装后的新二进制路径或错误。
	Install(assetPath string) (string, error)

	// Rollback 在更新失败时回滚到上一版本。
	Rollback() error

	// CurrentBinaryPath 返回当前运行的二进制文件路径。
	CurrentBinaryPath() string

	// InstallKind 返回当前安装方式标识，供更新器选择对应资产与安装策略。
	// Linux 返回 appimage/deb/rpm/unknown；其他平台返回对应平台标识。
	InstallKind() string
}

// ProgressReader 为 io.Reader 包装进度回调，用于下载进度追踪。
type ProgressReader struct {
	Reader     io.Reader
	Total      int64
	Downloaded int64
	Callback   func(downloaded, total int64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Downloaded += int64(n)
	if pr.Callback != nil {
		pr.Callback(pr.Downloaded, pr.Total)
	}
	return n, err
}
