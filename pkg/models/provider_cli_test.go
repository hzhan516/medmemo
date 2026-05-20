package models

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadCLITokenFromFile_Kimi_Success 验证 Kimi 格式正确解析 access_token。
func TestReadCLITokenFromFile_Kimi_Success(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "kimi-code.json")
	content := `{"access_token":"eyJkummy","refresh_token":"rt_abc","expires_at":1735689600}`
	require.NoError(t, os.WriteFile(credPath, []byte(content), 0600))

	token, hint, err := ReadCLITokenFromFile(credPath)
	assert.NoError(t, err)
	assert.Equal(t, "eyJkummy", token)
	assert.Empty(t, hint)
}

// TestReadCLITokenFromFile_Kimi_MissingAccessToken 验证 Kimi 格式缺少 access_token 时降级。
func TestReadCLITokenFromFile_Kimi_MissingAccessToken(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "kimi-code.json")
	content := `{"refresh_token":"rt_abc","expires_at":1735689600}`
	require.NoError(t, os.WriteFile(credPath, []byte(content), 0600))

	token, hint, err := ReadCLITokenFromFile(credPath)
	assert.NoError(t, err)
	// 没有 access_token，Kimi 解析失败，继续尝试 ADC 格式
	// ADC 格式有 refresh_token，返回 refresh_token + hint
	assert.Equal(t, "rt_abc", token)
	assert.Equal(t, "refresh_token", hint)
}

// TestReadCLITokenFromFile_GeminiADC_Success 验证 gcloud ADC 格式正确解析 refresh_token。
func TestReadCLITokenFromFile_GeminiADC_Success(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "adc.json")
	content := `{"client_id":"cid","client_secret":"secret","refresh_token":"1//refresh_xyz","type":"authorized_user"}`
	require.NoError(t, os.WriteFile(credPath, []byte(content), 0600))

	token, hint, err := ReadCLITokenFromFile(credPath)
	assert.NoError(t, err)
	assert.Equal(t, "1//refresh_xyz", token)
	assert.Equal(t, "refresh_token", hint)
}

// TestReadCLITokenFromFile_FileNotExist 验证文件不存在返回错误。
func TestReadCLITokenFromFile_FileNotExist(t *testing.T) {
	token, hint, err := ReadCLITokenFromFile("/nonexistent/path/cred.json")
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Empty(t, hint)
	assert.Contains(t, err.Error(), "failed to read cli credential file")
}

// TestReadCLITokenFromFile_EmptyPath 验证空路径返回错误。
func TestReadCLITokenFromFile_EmptyPath(t *testing.T) {
	token, hint, err := ReadCLITokenFromFile("")
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Empty(t, hint)
	assert.Contains(t, err.Error(), "cli credential path is empty")
}

// TestReadCLITokenFromFile_EmptyFile 验证空文件返回错误。
func TestReadCLITokenFromFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "empty.json")
	require.NoError(t, os.WriteFile(credPath, []byte("   \n  "), 0600))

	token, hint, err := ReadCLITokenFromFile(credPath)
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Empty(t, hint)
	assert.Contains(t, err.Error(), "is empty")
}

// TestReadCLITokenFromFile_InvalidJSON_FallbackPlainText 验证无效 JSON 降级为纯文本 token。
func TestReadCLITokenFromFile_InvalidJSON_FallbackPlainText(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "plain.txt")
	require.NoError(t, os.WriteFile(credPath, []byte("sk-plain-token-123"), 0600))

	token, hint, err := ReadCLITokenFromFile(credPath)
	assert.NoError(t, err)
	assert.Equal(t, "sk-plain-token-123", token)
	assert.Empty(t, hint)
}

// TestReadCLITokenFromFile_PermissionDenied 验证权限不足返回错误。
func TestReadCLITokenFromFile_PermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("权限测试跳过 Windows")
	}

	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "secret.json")
	require.NoError(t, os.WriteFile(credPath, []byte(`{"access_token":"secret"}`), 0600))
	require.NoError(t, os.Chmod(credPath, 0000))
	defer os.Chmod(credPath, 0600) // 清理权限，避免 TempDir 清理失败

	token, hint, err := ReadCLITokenFromFile(credPath)
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Empty(t, hint)
	assert.Contains(t, err.Error(), "permission denied")
}

// TestReadCLITokenFromFile_ExpandHome 验证 ~ 展开为 home 目录。
func TestReadCLITokenFromFile_ExpandHome(t *testing.T) {
	token, hint, err := ReadCLITokenFromFile("~/nonexistent_test_file_12345.json")
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Empty(t, hint)
	// 路径应已展开，错误信息中不应包含 ~/"
	assert.NotContains(t, err.Error(), "~/")
}

// TestExpandPath 验证路径展开逻辑。
func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		contains string
	}{
		{"~/test.json", "test.json"},
		{"/absolute/path.json", "/absolute/path.json"},
		{"relative/path.json", "relative/path.json"},
		{"~", "~"}, // 单独的 ~ 不展开
	}

	for _, tt := range tests {
		result := expandPath(tt.input)
		if tt.input == "~/test.json" {
			assert.Equal(t, filepath.Join(home, "test.json"), result)
		} else {
			assert.Equal(t, tt.input, result)
		}
	}
}
