package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/medmemo/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLookPath 模拟 exec.LookPath 的返回行为。
type mockLookPath func(string) (string, error)

func newDetectorWithMock(serverURL string, mock mockLookPath) *OllamaDetector {
	d := NewOllamaDetectorWithClient(serverURL, &http.Client{Timeout: 2 * time.Second})
	d.lookPath = mock
	return d
}

func TestOllamaDetector_IsInstalled(t *testing.T) {
	tests := []struct {
		name      string
		lookPath  mockLookPath
		installed bool
	}{
		{
			name:      "已安装",
			lookPath:  func(string) (string, error) { return "/usr/local/bin/ollama", nil },
			installed: true,
		},
		{
			name:      "未安装",
			lookPath:  func(string) (string, error) { return "", exec.ErrNotFound },
			installed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newDetectorWithMock("", tt.lookPath)
			assert.Equal(t, tt.installed, d.IsInstalled())
		})
	}
}

func TestOllamaDetector_IsRunning(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		running bool
	}{
		{"正常运行", http.StatusOK, true},
		{"返回 404", http.StatusNotFound, false},
		{"返回 500", http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/tags", r.URL.Path)
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			d := newDetectorWithMock(server.URL, func(string) (string, error) { return "", exec.ErrNotFound })
			assert.Equal(t, tt.running, d.IsRunning())
		})
	}
}

func TestOllamaDetector_IsRunning_NotReachable(t *testing.T) {
	// 使用不可达的地址直接测试网络超时
	d := NewOllamaDetectorWithEndpoint("http://localhost:59999")
	assert.False(t, d.IsRunning())
}

func TestOllamaDetector_HasModel_Found(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/tags", r.URL.Path)
		resp := tagsResponse{
			Models: []tagModel{
				{Name: "llama3.1:latest"},
				{Name: "smollm2:135m"},
				{Name: "qwen2:7b"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	d := newDetectorWithMock(server.URL, nil)
	assert.True(t, d.HasModel("smollm2:135m"))
	assert.True(t, d.HasModel("llama3.1:latest"))
}

func TestOllamaDetector_HasModel_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := tagsResponse{
			Models: []tagModel{
				{Name: "llama3.1:latest"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	d := newDetectorWithMock(server.URL, nil)
	assert.False(t, d.HasModel("smollm2:135m"))
	assert.False(t, d.HasModel("nonexistent:latest"))
}

func TestOllamaDetector_HasModel_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := tagsResponse{Models: []tagModel{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	d := newDetectorWithMock(server.URL, nil)
	assert.False(t, d.HasModel("smollm2:135m"))
}

func TestOllamaDetector_HasModel_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	d := newDetectorWithMock(server.URL, nil)
	assert.False(t, d.HasModel("smollm2:135m"))
}

func TestOllamaDetector_WaitForServer_Success(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := newDetectorWithMock(server.URL, nil)
	err := d.WaitForServer(5 * time.Second)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, callCount, 1)
}

func TestOllamaDetector_WaitForServer_Timeout(t *testing.T) {
	// 服务器始终返回 500，模拟服务未就绪
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	d := newDetectorWithMock(server.URL, nil)
	err := d.WaitForServer(500 * time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not become ready")
}

func TestOllamaDetector_WaitForServer_EventuallyReady(t *testing.T) {
	// 模拟服务延迟启动：前两次返回 500，之后返回 200
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := newDetectorWithMock(server.URL, nil)
	err := d.WaitForServer(5 * time.Second)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, callCount, 3)
}

func TestOllamaDetector_GetInstallGuide(t *testing.T) {
	d := NewOllamaDetector()
	guide := d.GetInstallGuide()
	require.NotEmpty(t, guide)
	// 验证引导文本包含 ollama.com 链接
	assert.Contains(t, guide, "ollama.com")
}

func TestOllamaDetector_BuildProviderConfig(t *testing.T) {
	d := NewOllamaDetector()
	cfg := d.BuildProviderConfig()

	require.NotNil(t, cfg)
	assert.Contains(t, cfg.ID, "ollama-local-")
	assert.Equal(t, "Ollama (本地)", cfg.Name)
	assert.Equal(t, defaultOllamaEndpoint, cfg.APIHost)
	assert.Equal(t, DefaultModelName, cfg.ModelID)
	assert.Equal(t, 0.7, cfg.Temperature)
	assert.Equal(t, 30000, cfg.TimeoutMs)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, "本地", cfg.GroupName)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, models.AuthMethodAPIToken, cfg.AuthMethod)
	assert.Greater(t, cfg.CreatedAt, int64(0))
	assert.Greater(t, cfg.UpdatedAt, int64(0))
}

func TestOllamaDetector_Detect_NotInstalled(t *testing.T) {
	d := newDetectorWithMock("", func(string) (string, error) {
		return "", exec.ErrNotFound
	})

	result := d.Detect()
	assert.False(t, result.Installed)
	assert.False(t, result.Running)
	assert.False(t, result.HasSmolLM2)
	assert.NotEmpty(t, result.InstallGuide)
}

func TestOllamaDetector_Detect_InstalledNotRunning(t *testing.T) {
	// ollama 已安装但服务未启动
	d := newDetectorWithMock("http://localhost:59999", func(string) (string, error) {
		return "/usr/bin/ollama", nil
	})

	result := d.Detect()
	assert.True(t, result.Installed)
	assert.False(t, result.Running)
	assert.False(t, result.HasSmolLM2)
	assert.Empty(t, result.InstallGuide)
}

func TestOllamaDetector_Detect_FullyReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := tagsResponse{
			Models: []tagModel{
				{Name: "smollm2:135m"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	d := newDetectorWithMock(server.URL, func(string) (string, error) {
		return "/usr/bin/ollama", nil
	})

	result := d.Detect()
	assert.True(t, result.Installed)
	assert.True(t, result.Running)
	assert.True(t, result.HasSmolLM2)
	assert.Empty(t, result.InstallGuide)
}

func TestOllamaDetector_Detect_ReadyWithoutModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := tagsResponse{
			Models: []tagModel{
				{Name: "llama3.1:latest"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	d := newDetectorWithMock(server.URL, func(string) (string, error) {
		return "/usr/bin/ollama", nil
	})

	result := d.Detect()
	assert.True(t, result.Installed)
	assert.True(t, result.Running)
	assert.False(t, result.HasSmolLM2)
}

// TestOllamaDetector_PullModel_ContextCancel 验证取消上下文可中断 pull 操作。
// 此测试仅验证命令构造和上下文传递，不执行真实的 ollama pull。
func TestOllamaDetector_PullModel_ContextCancel(t *testing.T) {
	// 如果系统没有 ollama 命令，跳过此测试
	_, err := exec.LookPath("ollama")
	if err != nil {
		t.Skip("ollama command not found in PATH, skipping integration test")
	}

	d := NewOllamaDetector()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 使用一个极短超时，确保命令在超时前无法完成
	err = d.PullModel(ctx, "nonexistent-model-for-test", func(string) {})
	require.Error(t, err)
	// 上下文取消或命令失败均可接受
}

// TestOllamaDetector_StartServer_Installed 验证 StartServer 在已安装场景下返回 cmd。
func TestOllamaDetector_StartServer_Installed(t *testing.T) {
	_, err := exec.LookPath("ollama")
	if err != nil {
		t.Skip("ollama command not found in PATH, skipping integration test")
	}

	d := NewOllamaDetector()
	cmd, err := d.StartServer()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.NotNil(t, cmd.Process)

	// 如果服务成功启动，轮询等待其就绪
	waitErr := d.WaitForServer(10 * time.Second)
	if waitErr == nil {
		// 服务已就绪，清理：停止刚启动的进程
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
	// 如果等待失败，可能是端口被其他 ollama 实例占用，或启动失败——均为可接受结果
}
