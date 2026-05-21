//go:build !windows

package ai

import (
	"os/exec"
	"syscall"
)

// setDetachedAttr 为 Unix 系统设置进程脱离属性，
// 使 ollama serve 在独立进程组中运行，避免接收父进程的 SIGHUP。
func setDetachedAttr(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}
