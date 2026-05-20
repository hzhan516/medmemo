package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadCLITokenFromFile 读取 CLI 凭证文件并解析 token。
//
// 支持格式（按优先级尝试）：
//  1. Kimi CLI: ~/.kimi/credentials/kimi-code.json {"access_token":"...","refresh_token":"..."}
//  2. gcloud ADC: ~/.config/gcloud/application_default_credentials.json {"refresh_token":"...","type":"authorized_user"}
//  3. 纯文本: 文件内容直接作为 token 字符串
//
// providerHint 取值：
//   - ""（空字符串）：token 可直接用于 HTTP Bearer 认证
//   - "refresh_token"：返回的是 refresh_token，需 TASK-045 刷新为 access_token
func ReadCLITokenFromFile(path string) (token, providerHint string, err error) {
	if path == "" {
		return "", "", fmt.Errorf("cli credential path is empty")
	}

	path = expandPath(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("failed to read cli credential file %s: %w", path, err)
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return "", "", fmt.Errorf("cli credential file %s is empty", path)
	}

	// 尝试解析为 Kimi 格式
	var kimiCred struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresAt    int64  `json:"expires_at"`
	}
	if err := json.Unmarshal(data, &kimiCred); err == nil && kimiCred.AccessToken != "" {
		return kimiCred.AccessToken, "", nil
	}

	// 尝试解析为 gcloud ADC 格式
	var adc struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RefreshToken string `json:"refresh_token"`
		Type         string `json:"type"`
	}
	if err := json.Unmarshal(data, &adc); err == nil && adc.RefreshToken != "" {
		return adc.RefreshToken, "refresh_token", nil
	}

	// 兜底：纯文本 token
	return trimmed, "", nil
}

// expandPath 将路径中的 ~ 展开为用户主目录。
func expandPath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
