//go:build linux

package main

import (
	"context"
	"fmt"
	"os/exec"
)

// restartAfterUpdate 启动新 AppImage，由调用方执行 runtime.Quit。
func restartAfterUpdate(_ context.Context, installedPath string) error {
	if err := exec.Command(installedPath).Start(); err != nil {
		return fmt.Errorf("failed to start new AppImage: %w", err)
	}
	return nil
}
