//go:build windows

package ai

import (
	"os/exec"
	"syscall"
)

// setDetachedAttr 为 Windows 设置新进程组标志，
// 使 ollama serve 在独立进程组中运行。
func setDetachedAttr(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP
}
