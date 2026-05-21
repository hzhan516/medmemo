// Package auth 实现 Service Account 认证解析。
// 专用于 Vertex AI（Google Cloud）的 Service Account JSON 密钥处理。
package auth

import (
	"encoding/json"
	"fmt"
)

// ServiceAccountJSON Google Service Account 密钥文件结构。
type ServiceAccountJSON struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

// ParseServiceAccountJSON 解析 Google Service Account JSON 密钥内容。
// 返回提取后的 project_id、client_email、private_key。
// 调用方应在获取返回值后立即丢弃原始 JSON 字符串。
func ParseServiceAccountJSON(jsonStr string) (projectID, clientEmail, privateKey string, err error) {
	if jsonStr == "" {
		return "", "", "", fmt.Errorf("service account JSON is empty")
	}

	var sa ServiceAccountJSON
	if err := json.Unmarshal([]byte(jsonStr), &sa); err != nil {
		return "", "", "", fmt.Errorf("failed to parse service account JSON: %w", err)
	}

	if sa.Type != "service_account" {
		return "", "", "", fmt.Errorf("invalid service account type: expected 'service_account', got '%s'", sa.Type)
	}
	if sa.ProjectID == "" {
		return "", "", "", fmt.Errorf("service account JSON missing project_id")
	}
	if sa.ClientEmail == "" {
		return "", "", "", fmt.Errorf("service account JSON missing client_email")
	}
	if sa.PrivateKey == "" {
		return "", "", "", fmt.Errorf("service account JSON missing private_key")
	}

	return sa.ProjectID, sa.ClientEmail, sa.PrivateKey, nil
}
