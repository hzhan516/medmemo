//go:build darwin

package main

import (
	"context"
	"fmt"
	"os/exec"
)

// restartAfterUpdate 通过 open 启动新 .app，由调用方执行 runtime.Quit。
func restartAfterUpdate(_ context.Context, installedPath string) error {
	if err := exec.Command("open", installedPath).Start(); err != nil {
		return fmt.Errorf("failed to open new app bundle: %w", err)
	}
	return nil
}
