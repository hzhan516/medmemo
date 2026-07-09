package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpenAIAdapter_Chat_Success 验证非流式对话正常响应。
func TestOpenAIAdapter_Chat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		resp := chatResponse{
			Choices: []choice{
				{
					Message:      message{Role: "assistant", Content: "你好，有什么可以帮你的？"},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "test-model", 0, 30*time.Second)
	msgs := []models.Message{
		{Role: models.RoleUser, Content: "你好"},
	}

	reply, err := adapter.Chat(context.Background(), msgs)
	require.NoError(t, err)
	assert.Equal(t, "你好，有什么可以帮你的？", reply)
}

func TestOpenAIAdapter_Chat_ProviderBaseWithPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1beta/openai/chat/completions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(chatResponse{
			Choices: []choice{{Message: message{Role: "assistant", Content: "ok"}}},
		})
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL+"/v1beta/openai", "gemini-2.5-flash", 0, 30*time.Second)
	reply, err := adapter.Chat(context.Background(), []models.Message{{Role: models.RoleUser, Content: "hi"}})
	require.NoError(t, err)
	assert.Equal(t, "ok", reply)
}

// TestOpenAIAdapter_Chat_Unauthorized 验证 401 认证失败错误映射。
func TestOpenAIAdapter_Chat_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := chatResponse{
			Error: &apiError{
				Message: "Invalid Authentication",
				Type:    "invalid_request_error",
				Code:    "invalid_api_key",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("wrong-key", server.URL, "test-model", 0, 30*time.Second)
	msgs := []models.Message{{Role: models.RoleUser, Content: "test"}}

	_, err := adapter.Chat(context.Background(), msgs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API 认证失败")
	var apiErr *apiError
	assert.True(t, errors.As(err, &apiErr))
}

// TestOpenAIAdapter_Chat_RateLimit 验证 429 速率限制时重试后成功。
func TestOpenAIAdapter_Chat_RateLimit(t *testing.T) {
	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			resp := chatResponse{
				Error: &apiError{
					Message: "Rate limit exceeded",
					Type:    "rate_limit_error",
					Code:    "rate_limit_exceeded",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		resp := chatResponse{
			Choices: []choice{
				{Message: message{Role: "assistant", Content: "hi"}, FinishReason: "stop"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "test-model", 0, 30*time.Second)
	msgs := []models.Message{{Role: models.RoleUser, Content: "test"}}

	reply, err := adapter.Chat(context.Background(), msgs)
	require.NoError(t, err)
	assert.Equal(t, "hi", reply)
	assert.Equal(t, 2, callCount)
}

// TestOpenAIAdapter_Chat_ModelNotFound 验证 404 模型不存在错误映射。
func TestOpenAIAdapter_Chat_ModelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := chatResponse{
			Error: &apiError{
				Message: "The model 'unknown-model' does not exist",
				Type:    "invalid_request_error",
				Code:    "model_not_found",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "unknown-model", 0, 30*time.Second)
	msgs := []models.Message{{Role: models.RoleUser, Content: "test"}}

	_, err := adapter.Chat(context.Background(), msgs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "模型不存在")
}

// TestOpenAIAdapter_Chat_Timeout 验证网络超时错误处理。
func TestOpenAIAdapter_Chat_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "test-model", 0, 30*time.Second)
	// 将超时设为极短，确保触发超时
	adapter.client.Timeout = 1 * time.Millisecond

	msgs := []models.Message{{Role: models.RoleUser, Content: "test"}}

	_, err := adapter.Chat(context.Background(), msgs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send chat request")
}

// TestOpenAIAdapter_StreamChat_Success 验证 SSE 流式响应正常解析。
func TestOpenAIAdapter_StreamChat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))

		// 解析请求体，确认 stream=true
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		assert.True(t, req.Stream)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []string{
			`{"choices":[{"delta":{"content":"你好"},"finish_reason":""}]}`,
			`{"choices":[{"delta":{"content":"，"},"finish_reason":""}]}`,
			`{"choices":[{"delta":{"content":"世界"},"finish_reason":""}]}`,
			`{"choices":[{"delta":{"content":"！"},"finish_reason":"stop"}]}`,
		}
		for _, c := range chunks {
			_, _ = w.Write([]byte("data: " + c + "\n\n"))
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "test-model", 0, 30*time.Second)
	msgs := []models.Message{{Role: models.RoleUser, Content: "打招呼"}}

	var collected strings.Builder
	usage, err := adapter.StreamChat(context.Background(), msgs, func(chunk string) {
		collected.WriteString(chunk)
	})
	require.NoError(t, err)
	assert.Equal(t, "你好，世界！", collected.String())
	require.NotNil(t, usage)
	assert.Equal(t, "stop", usage.FinishReason)
}

// TestOpenAIAdapter_StreamChat_EmptyContent 验证流式响应中空的 content 不触发回调。
func TestOpenAIAdapter_StreamChat_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// 第一条有内容，第二条空内容，第三条 finish
		chunks := []string{
			`{"choices":[{"delta":{"content":"hi"},"finish_reason":""}]}`,
			`{"choices":[{"delta":{"content":""},"finish_reason":""}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
		}
		for _, c := range chunks {
			_, _ = w.Write([]byte("data: " + c + "\n\n"))
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "test-model", 0, 30*time.Second)
	msgs := []models.Message{{Role: models.RoleUser, Content: "test"}}

	callCount := 0
	usage, err := adapter.StreamChat(context.Background(), msgs, func(_ string) {
		callCount++
	})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
	require.NotNil(t, usage)
	assert.Equal(t, "stop", usage.FinishReason)
}

// TestOpenAIAdapter_StreamChat_ErrorStatus 验证流式请求返回错误状态码。
func TestOpenAIAdapter_StreamChat_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := streamChunk{
			Error: &apiError{
				Message: "Invalid API Key",
				Code:    "invalid_api_key",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("bad-key", server.URL, "test-model", 0, 30*time.Second)
	msgs := []models.Message{{Role: models.RoleUser, Content: "test"}}

	usage, err := adapter.StreamChat(context.Background(), msgs, func(_ string) {})
	require.Error(t, err)
	assert.Nil(t, usage)
	assert.Contains(t, err.Error(), "API 认证失败")
}

// TestOpenAIAdapter_CheckAvailability_Success 验证连通性检测成功。
func TestOpenAIAdapter_CheckAvailability_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/models", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "test-model", 0, 30*time.Second)
	ok, reason := adapter.CheckAvailability(context.Background())
	assert.True(t, ok)
	assert.Equal(t, "available", reason)
}

// TestOpenAIAdapter_CheckAvailability_NoKey 验证未配置 API Key 时返回不可用。
func TestOpenAIAdapter_CheckAvailability_NoKey(t *testing.T) {
	adapter := NewOpenAIAdapter("", "https://api.example.com", "test-model", 0, 30*time.Second)
	ok, reason := adapter.CheckAvailability(context.Background())
	assert.False(t, ok)
	assert.Equal(t, "API key not configured", reason)
}

// TestOpenAIAdapter_CheckAvailability_Unauthorized 验证 401 时返回认证失败。
func TestOpenAIAdapter_CheckAvailability_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("bad-key", server.URL, "test-model", 0, 30*time.Second)
	ok, reason := adapter.CheckAvailability(context.Background())
	assert.False(t, ok)
	assert.Contains(t, reason, "invalid API key")
}

// TestOpenAIAdapter_CheckAvailability_NetworkError 验证网络不可达时返回失败。
func TestOpenAIAdapter_CheckAvailability_NetworkError(t *testing.T) {
	// 使用一个必然不可达的地址
	adapter := NewOpenAIAdapter("test-key", "http://127.0.0.1:1", "test-model", 0, 30*time.Second)
	adapter.client.Timeout = 100 * time.Millisecond
	ok, reason := adapter.CheckAvailability(context.Background())
	assert.False(t, ok)
	assert.Contains(t, reason, "connection failed")
}

// TestOpenAIAdapter_Chat_RateLimitNoErrorBody 验证 429 无错误体时不包含 <nil>。
func TestOpenAIAdapter_Chat_RateLimitNoErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "test-model", 0, 30*time.Second)
	adapter.client.Timeout = 5 * time.Second
	msgs := []models.Message{{Role: models.RoleUser, Content: "test"}}

	_, err := adapter.Chat(context.Background(), msgs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "请求过于频繁")
	assert.NotContains(t, err.Error(), "<nil>")
}

// TestOpenAIAdapter_StreamChat_RateLimitRetry 验证 429 时重试后成功。
func TestOpenAIAdapter_StreamChat_RateLimitRetry(t *testing.T) {
	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "test-model", 0, 30*time.Second)
	msgs := []models.Message{{Role: models.RoleUser, Content: "test"}}

	var collected strings.Builder
	usage, err := adapter.StreamChat(context.Background(), msgs, func(chunk string) {
		collected.WriteString(chunk)
	})
	require.NoError(t, err)
	assert.Equal(t, 3, callCount)
	assert.Equal(t, "ok", collected.String())
	require.NotNil(t, usage)
	assert.Equal(t, "stop", usage.FinishReason)
}

// TestOpenAIAdapter_Chat_RateLimitRetryExhausted 验证 429 重试耗尽后返回错误。
func TestOpenAIAdapter_Chat_RateLimitRetryExhausted(t *testing.T) {
	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "test-model", 0, 30*time.Second)
	adapter.client.Timeout = 5 * time.Second
	msgs := []models.Message{{Role: models.RoleUser, Content: "test"}}

	_, err := adapter.Chat(context.Background(), msgs)
	require.Error(t, err)
	assert.Equal(t, 4, callCount) // 初始 1 次 + 3 次重试
	assert.Contains(t, err.Error(), "请求过于频繁")
}

// TestOpenAIAdapter_Chat_EmptyChoices 验证空 choices 时返回错误。
func TestOpenAIAdapter_Chat_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := chatResponse{Choices: []choice{}}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "test-model", 0, 30*time.Second)
	msgs := []models.Message{{Role: models.RoleUser, Content: "test"}}

	_, err := adapter.Chat(context.Background(), msgs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty response")
}

// TestOpenAICompatibleEndpoint 验证 endpoint 拼接逻辑在各种 baseURL 下的正确性。
func TestOpenAICompatibleEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		resource string
		want     string
	}{
		{
			name:     "空 baseURL",
			baseURL:  "",
			resource: "chat/completions",
			want:     "/chat/completions",
		},
		{
			name:     "无路径 baseURL",
			baseURL:  "https://api.example.com",
			resource: "chat/completions",
			want:     "https://api.example.com/v1/chat/completions",
		},
		{
			name:     "无路径 baseURL 带斜杠 + resource 带斜杠",
			baseURL:  "https://api.example.com/",
			resource: "/chat/completions",
			want:     "https://api.example.com/v1/chat/completions",
		},
		{
			name:     "含版本路径 baseURL",
			baseURL:  "https://api.example.com/v1beta",
			resource: "chat/completions",
			want:     "https://api.example.com/v1beta/chat/completions",
		},
		{
			name:     "含版本路径 baseURL 带斜杠",
			baseURL:  "https://api.example.com/v1beta/",
			resource: "chat/completions",
			want:     "https://api.example.com/v1beta/chat/completions",
		},
		{
			name:     "自定义路径 baseURL",
			baseURL:  "https://api.example.com/custom/path",
			resource: "embeddings",
			want:     "https://api.example.com/custom/path/embeddings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := openAICompatibleEndpoint(tt.baseURL, tt.resource)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestOpenAIAdapter_streamClient_timeout0 验证流式客户端 Timeout 被显式设为 0（由 context 控制生命周期）。
func TestOpenAIAdapter_streamClient_timeout0(t *testing.T) {
	adapter := NewOpenAIAdapter("test-key", "https://api.example.com", "test-model", 0, 30*time.Second)
	require.NotNil(t, adapter.streamClient)
	assert.Equal(t, time.Duration(0), adapter.streamClient.Timeout)
}

// TestOpenAIAdapter_doWithRetry_maxRetries3 验证 doWithRetry 在遇到持续 429 时最多重试 3 次（共 4 次请求）后返回错误。
// 注意：生产代码中退避间隔为 2s/4s/8s，本测试执行耗时约 14s。
func TestOpenAIAdapter_doWithRetry_maxRetries3(t *testing.T) {
	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "test-model", 0, 30*time.Second)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(`{}`))
	require.NoError(t, err)

	_, err = adapter.doWithRetry(adapter.client, req)
	require.Error(t, err)
	assert.Equal(t, 4, callCount) // 初始 1 次 + 3 次重试
	assert.Contains(t, err.Error(), "请求过于频繁")
}

// TestOpenAIAdapter_StreamChat_ServerError 验证流式请求遇到持续 500 服务端错误时返回错误。
// doWithRetry 对 5xx 同样会执行 3 次重试，本测试执行耗时约 14s。
func TestOpenAIAdapter_StreamChat_ServerError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode: ~14s due to exponential backoff")
	}
	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "test-model", 0, 30*time.Second)
	msgs := []models.Message{{Role: models.RoleUser, Content: "test"}}

	usage, err := adapter.StreamChat(context.Background(), msgs, func(_ string) {})
	require.Error(t, err)
	assert.Nil(t, usage)
	assert.Equal(t, 4, callCount) // 初始 1 次 + 3 次重试
	assert.Contains(t, err.Error(), "failed to send stream request")
}

// TestOpenAIAdapter_Chat_500Error 验证非流式请求遇到持续 500 服务端错误时，重试耗尽后返回错误。
// 退避间隔为 2s/4s/8s，本测试执行耗时约 14s。
func TestOpenAIAdapter_Chat_500Error(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode: ~14s due to exponential backoff")
	}
	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "test-model", 0, 30*time.Second)
	msgs := []models.Message{{Role: models.RoleUser, Content: "test"}}

	_, err := adapter.Chat(context.Background(), msgs)
	require.Error(t, err)
	assert.Equal(t, 4, callCount) // 初始 1 次 + 3 次重试
	assert.Contains(t, err.Error(), "failed to send chat request")
}

// TestOpenAIAdapter_Chat_InvalidJSON 验证服务端返回 200 但响应体为非法 JSON 时返回解析错误。
func TestOpenAIAdapter_Chat_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	adapter := NewOpenAIAdapter("test-key", server.URL, "test-model", 0, 30*time.Second)
	msgs := []models.Message{{Role: models.RoleUser, Content: "test"}}

	_, err := adapter.Chat(context.Background(), msgs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal chat response")
}
