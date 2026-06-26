//go:build windows

package main

import "context"

// restartAfterUpdate Windows 由 NSIS 安装完成后自动启动新版本，这里直接返回。
func restartAfterUpdate(_ context.Context, _ string) error {
	return nil
}
