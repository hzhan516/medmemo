package feedback

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReporter_Collect(t *testing.T) {
	t.Parallel()
	r := NewReporter("v1.2.3", "2026-05-20")
	info := r.Collect()

	assert.Equal(t, "v1.2.3", info.AppVersion)
	assert.Equal(t, "2026-05-20", info.BuildTime)
	assert.NotEmpty(t, info.GoVersion)
	assert.NotEmpty(t, info.OS)
	assert.NotEmpty(t, info.Arch)
}

func TestReporter_BuildIssueURL(t *testing.T) {
	t.Parallel()
	r := NewReporter("v0.5.0", "")
	info := &SystemInfo{
		AppVersion: "v0.5.0",
		GoVersion:  "go1.26.0",
		OS:         "darwin",
		Arch:       "arm64",
	}

	u := r.BuildIssueURL(info, "测试问题描述", "some error log")
	require.True(t, strings.HasPrefix(u, githubIssueBaseURL+"?"), "URL should start with base")

	// 解析 URL 并验证参数
	parsed, err := url.Parse(u)
	require.NoError(t, err)

	q := parsed.Query()
	assert.Equal(t, "[Bug] MedMemo v0.5.0 on darwin/arm64", q.Get("title"))

	body := q.Get("body")
	assert.Contains(t, body, "测试问题描述")
	assert.Contains(t, body, "App 版本")
	assert.Contains(t, body, "v0.5.0")
	assert.Contains(t, body, "go1.26.0")
	assert.Contains(t, body, "darwin")
	assert.Contains(t, body, "arm64")
	assert.Contains(t, body, "some error log")
	assert.Contains(t, body, "不包含任何对话内容")
}

func TestReporter_BuildIssueURL_EmptyDescription(t *testing.T) {
	t.Parallel()
	r := NewReporter("v0.1.0", "")
	info := &SystemInfo{AppVersion: "v0.1.0", OS: "linux", Arch: "amd64"}

	u := r.BuildIssueURL(info, "", "")
	parsed, err := url.Parse(u)
	require.NoError(t, err)

	body := parsed.Query().Get("body")
	assert.Contains(t, body, "（用户未提供描述）")
	assert.NotContains(t, body, "## 错误日志") // 没有错误日志时不应包含该区块
}

func TestSanitizeLog(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		input      string
		contains   string
		notContain string
	}{
		{
			name:     "保留普通日志行",
			input:    "INFO: application started\nDEBUG: loading config",
			contains: "application started",
		},
		{
			name:       "脱敏包含 api_key 的行",
			input:      "request header: api_key=sk-1234567890abcdef",
			contains:   "[REDACTED",
			notContain: "sk-1234567890abcdef",
		},
		{
			name:       "脱敏包含 token 的行",
			input:      "auth token: bearer abcdef123456",
			contains:   "[REDACTED",
			notContain: "abcdef123456",
		},
		{
			name:       "脱敏包含 password 的行",
			input:      "db password: mysecretpassword123",
			contains:   "[REDACTED",
			notContain: "mysecretpassword123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLog(tt.input)
			assert.Contains(t, result, tt.contains)
			if tt.notContain != "" {
				assert.NotContains(t, result, tt.notContain)
			}
		})
	}
}

func TestReadAppLogFile_NotExist(t *testing.T) {
	t.Parallel()
	// 使用不存在的临时目录
	content, err := ReadAppLogFile("/tmp/nonexistent_medmemo_dir_12345")
	require.NoError(t, err)
	assert.Equal(t, "", content)
}
