//go:build !wireinject
// +build !wireinject

package main

import "fmt"

// InitializeApp 的占位实现，供非 Wire 构建使用。
// 实际依赖注入代码由 wire_gen.go 提供（运行 make wire 生成）。
func InitializeApp() (*App, func(), error) {
	return nil, nil, fmt.Errorf("wire_gen.go not generated yet; run `make wire`")
}
