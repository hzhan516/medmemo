// Package entity 定义版本提示相关的领域模型。
// 遵循 AGENTS.md 零外部依赖铁律，仅使用 Go 标准库。
package entity

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// VersionNote 描述一个版本的功能与修复内容，供前端弹窗展示。
type VersionNote struct {
	Version  string   `json:"version"`  // 版本号，如 "v1.0"
	Title    string   `json:"title"`    // 版本标题
	Features []string `json:"features"` // 新增功能列表
	Fixes    []string `json:"fixes"`    // 问题修复列表
}

//go:embed changelog/zh-Hans.json
var versionNotesJSON []byte

// AllVersionNotes 按版本升序排列的全部版本提示数据。
// 数据来源于 changelog/zh-Hans.json，运行时通过 embed 加载解析。
var AllVersionNotes []VersionNote

func init() {
	if err := json.Unmarshal(versionNotesJSON, &AllVersionNotes); err != nil {
		panic(fmt.Errorf("failed to unmarshal version notes: %w", err))
	}
}
