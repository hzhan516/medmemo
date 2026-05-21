package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CLICredentials 从 CLI 凭证文件解析出的认证信息。
type CLICredentials struct {
	AccessToken  string // 可直接用于 HTTP Bearer 的 access_token
	RefreshToken string // 用于换取新 access_token 的 refresh_token
	ClientID     string // OAuth client_id（refresh 流程需要）
	ClientSecret string // OAuth client_secret（refresh 流程需要）
	ExpiresAt    int64  // access_token 过期时间戳（秒）
	ProviderHint string // "" 表示 AccessToken 可用；"refresh_token" 表示只有 RefreshToken
}

// ReadCLICredentials 读取 CLI 凭证文件并解析认证信息。
// 支持 Kimi CLI、gcloud ADC、纯文本三种格式。
func ReadCLICredentials(path string) (*CLICredentials, error) {
	if path == "" {
		return nil, fmt.Errorf("cli credential path is empty")
	}

	path = expandPath(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read cli credential file %s: %w", path, err)
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("cli credential file %s is empty", path)
	}

	// Kimi 格式
	var kimiCred struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		ExpiresAt    int64  `json:"expires_at"`
	}
	if err := json.Unmarshal(data, &kimiCred); err == nil {
		if kimiCred.AccessToken != "" {
			return &CLICredentials{
				AccessToken:  kimiCred.AccessToken,
				RefreshToken: kimiCred.RefreshToken,
				ClientID:     kimiCred.ClientID,
				ClientSecret: kimiCred.ClientSecret,
				ExpiresAt:    kimiCred.ExpiresAt,
				ProviderHint: "",
			}, nil
		}
		if kimiCred.RefreshToken != "" {
			return &CLICredentials{
				RefreshToken: kimiCred.RefreshToken,
				ClientID:     kimiCred.ClientID,
				ClientSecret: kimiCred.ClientSecret,
				ExpiresAt:    kimiCred.ExpiresAt,
				ProviderHint: "refresh_token",
			}, nil
		}
	}

	// gcloud ADC 格式
	var adc struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RefreshToken string `json:"refresh_token"`
		Type         string `json:"type"`
	}
	if err := json.Unmarshal(data, &adc); err == nil && adc.RefreshToken != "" {
		return &CLICredentials{
			RefreshToken: adc.RefreshToken,
			ClientID:     adc.ClientID,
			ClientSecret: adc.ClientSecret,
			ProviderHint: "refresh_token",
		}, nil
	}

	// 兜底：纯文本 token
	return &CLICredentials{
		AccessToken:  trimmed,
		ProviderHint: "",
	}, nil
}

// ReadCLITokenFromFile 读取 CLI 凭证文件并解析 token。
// 内部委托 ReadCLICredentials。
func ReadCLITokenFromFile(path string) (token, providerHint string, err error) {
	creds, err := ReadCLICredentials(path)
	if err != nil {
		return "", "", err
	}
	if creds.AccessToken != "" {
		return creds.AccessToken, "", nil
	}
	if creds.RefreshToken != "" {
		return creds.RefreshToken, "refresh_token", nil
	}
	return "", "", fmt.Errorf("no token found in credential file %s", path)
}

// ExpandPath 将路径中的 ~ 展开为用户主目录。
func ExpandPath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}

// expandPath ExpandPath 的内部别名。
func expandPath(path string) string {
	return ExpandPath(path)
}
