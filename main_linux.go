//go:build linux

// Package main 的 Linux 平台特定初始化。
// 包含 WebKit2GTK 信号栈冲突的根治方案（CGO sigaction 拦截器）。
package main

/*
#cgo LDFLAGS: -ldl
#define _GNU_SOURCE
#include <dlfcn.h>
#include <signal.h>
#include <stdio.h>

typedef int (*real_sigaction_t)(int, const struct sigaction *, struct sigaction *);

// 拦截所有 sigaction 调用，强制追加 SA_ONSTACK 标志。
// WebKit2GTK 安装的信号处理器缺少 SA_ONSTACK，导致 Go 1.21+ 运行时 fatal error。
// 此拦截器确保所有信号处理器（包括 WebKit 的）都使用 Go 运行时设置的信号栈。
int sigaction(int signum, const struct sigaction *act, struct sigaction *oldact) {
	static real_sigaction_t real_sigaction = NULL;
	if (!real_sigaction) {
		real_sigaction = (real_sigaction_t)dlsym(RTLD_NEXT, "sigaction");
	}
	if (act) {
		struct sigaction newact = *act;
		newact.sa_flags |= SA_ONSTACK;
		return real_sigaction(signum, &newact, oldact);
	}
	return real_sigaction(signum, act, oldact);
}
*/
import "C"

import "runtime"

func init() {
	// 将主 goroutine 绑定到当前 OS 线程，确保 GTK/WebKit 的初始化、
	// 事件循环和所有 C 回调都在同一个线程上执行，避免 Go 的 goroutine
	// 调度器在 CGO 调用期间切换线程导致的信号栈不匹配。
	runtime.LockOSThread()
}
