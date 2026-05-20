package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpenAICompatibleClient_Chat_Success 验证正常 SSE 流式响应。
func TestOpenAICompatibleClient_Chat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))

		// 验证请求体
		var reqBody chatCompletionRequest
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, "gpt-4o", reqBody.Model)
		assert.True(t, reqBody.Stream)
		assert.Equal(t, 0.5, reqBody.Temperature)
		require.Len(t, reqBody.Messages, 1)
		assert.Equal(t, "user", reqBody.Messages[0].Role)
		assert.Equal(t, "你好", reqBody.Messages[0].Content)

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

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{
		APIHost:     server.URL,
		APIKey:      "test-key",
		ModelID:     "gpt-4o",
		Temperature: 0.5,
	}
	req := ChatRequest{
		Messages: []models.Message{{Role: models.RoleUser, Content: "你好"}},
	}

	ch, err := client.Chat(context.Background(), req, cfg)
	require.NoError(t, err)

	var contents []string
	for chunk := range ch {
		switch chunk.Type {
		case ChunkContent:
			contents = append(contents, chunk.Payload)
		case ChunkDone:
			// 正常结束
		case ChunkError:
			t.Fatalf("unexpected error chunk: %s", chunk.Payload)
		}
	}

	assert.Equal(t, []string{"你好", "，", "世界", "！"}, contents)
}

// TestOpenAICompatibleClient_Chat_EmptyContent 验证空 content 不触发 content chunk。
func TestOpenAICompatibleClient_Chat_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

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

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, ModelID: "m"}
	ch, err := client.Chat(context.Background(), ChatRequest{}, cfg)
	require.NoError(t, err)

	contentCount := 0
	for chunk := range ch {
		if chunk.Type == ChunkContent {
			contentCount++
		}
	}
	assert.Equal(t, 1, contentCount)
}

// TestOpenAICompatibleClient_Chat_JSONParseError 验证非法 JSON 行被忽略。
func TestOpenAICompatibleClient_Chat_JSONParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("data: not-json\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, ModelID: "m"}
	ch, err := client.Chat(context.Background(), ChatRequest{}, cfg)
	require.NoError(t, err)

	var contents []string
	for chunk := range ch {
		if chunk.Type == ChunkContent {
			contents = append(contents, chunk.Payload)
		}
	}
	assert.Equal(t, []string{"ok"}, contents)
}

// TestOpenAICompatibleClient_Chat_ErrorStatus 验证 HTTP 错误状态码返回 LLMError。
func TestOpenAICompatibleClient_Chat_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := sseChunk{
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

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, APIKey: "bad-key", ModelID: "m"}
	_, err := client.Chat(context.Background(), ChatRequest{}, cfg)

	require.Error(t, err)
	llmErr, ok := err.(*LLMError)
	require.True(t, ok, "expected *LLMError")
	assert.Equal(t, "invalid_api_key", llmErr.Code)
	assert.Contains(t, llmErr.Message, "认证失败")
	assert.False(t, llmErr.Retryable)
}

// TestOpenAICompatibleClient_Chat_NetworkTimeout 验证网络超时返回 Retryable 错误。
func TestOpenAICompatibleClient_Chat_NetworkTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{
		APIHost: server.URL,
		ModelID: "m",
		Timeout: 1 * time.Millisecond,
	}
	_, err := client.Chat(context.Background(), ChatRequest{}, cfg)

	require.Error(t, err)
	llmErr, ok := err.(*LLMError)
	require.True(t, ok)
	assert.Equal(t, "network_error", llmErr.Code)
	assert.True(t, llmErr.Retryable)
}

// TestOpenAICompatibleClient_Chat_MissingAPIHost 验证 APIHost 为空时返回错误。
func TestOpenAICompatibleClient_Chat_MissingAPIHost(t *testing.T) {
	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{ModelID: "m"}
	_, err := client.Chat(context.Background(), ChatRequest{}, cfg)

	require.Error(t, err)
	llmErr, ok := err.(*LLMError)
	require.True(t, ok)
	assert.Equal(t, "missing_api_host", llmErr.Code)
}

// TestOpenAICompatibleClient_Chat_MissingModelID 验证 ModelID 为空时返回错误。
func TestOpenAICompatibleClient_Chat_MissingModelID(t *testing.T) {
	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: "http://example.com"}
	_, err := client.Chat(context.Background(), ChatRequest{}, cfg)

	require.Error(t, err)
	llmErr, ok := err.(*LLMError)
	require.True(t, ok)
	assert.Equal(t, "missing_model_id", llmErr.Code)
}

// TestOpenAICompatibleClient_Chat_ContextCancelled 验证 context 取消时流式中断。
func TestOpenAICompatibleClient_Chat_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// 慢速发送，确保 context 取消发生在流读取中
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"a\"},\"finish_reason\":\"\"}]}\n\n"))
		flusher.Flush()
		time.Sleep(500 * time.Millisecond)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"b\"},\"finish_reason\":\"\"}]}\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, ModelID: "m"}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	ch, err := client.Chat(ctx, ChatRequest{}, cfg)
	require.NoError(t, err)

	var gotError bool
	for chunk := range ch {
		if chunk.Type == ChunkError {
			gotError = true
		}
	}
	assert.True(t, gotError, "expected error chunk when context is cancelled")
}

// TestOpenAICompatibleClient_Chat_NoAPIKey 验证 APIKey 为空时仍能调用（部分本地端点无需认证）。
func TestOpenAICompatibleClient_Chat_NoAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, ModelID: "m"}
	ch, err := client.Chat(context.Background(), ChatRequest{}, cfg)
	require.NoError(t, err)

	var content string
	for chunk := range ch {
		if chunk.Type == ChunkContent {
			content += chunk.Payload
		}
	}
	assert.Equal(t, "ok", content)
}

// TestOpenAICompatibleClient_Chat_DefaultTemperature 验证默认温度值为 0.7。
func TestOpenAICompatibleClient_Chat_DefaultTemperature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody chatCompletionRequest
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, 0.7, reqBody.Temperature)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, ModelID: "m"}
	ch, err := client.Chat(context.Background(), ChatRequest{}, cfg)
	require.NoError(t, err)
	for range ch {
	}
}

// TestOpenAICompatibleClient_Chat_ZeroTemperatureConfig 验证 config.Temperature=0 且 req.Temperature=0 时使用默认 0.7。
func TestOpenAICompatibleClient_Chat_ZeroTemperatureConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody chatCompletionRequest
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, 0.7, reqBody.Temperature)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, ModelID: "m", Temperature: 0}
	ch, err := client.Chat(context.Background(), ChatRequest{Temperature: 0}, cfg)
	require.NoError(t, err)
	for range ch {
	}
}

// TestOpenAICompatibleClient_Chat_InvalidLinePrefix 验证不以 data: 开头的 SSE 行被忽略。
func TestOpenAICompatibleClient_Chat_InvalidLinePrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event: message\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n"))
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, ModelID: "m"}
	ch, err := client.Chat(context.Background(), ChatRequest{}, cfg)
	require.NoError(t, err)

	var content string
	for chunk := range ch {
		if chunk.Type == ChunkContent {
			content += chunk.Payload
		}
	}
	assert.Equal(t, "ok", content)
}

// TestOpenAICompatibleClient_Chat_EmptyChoices 验证 choices 为空数组时不触发 content。
func TestOpenAICompatibleClient_Chat_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"choices\":[],\"id\":\"test\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n"))
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, ModelID: "m"}
	ch, err := client.Chat(context.Background(), ChatRequest{}, cfg)
	require.NoError(t, err)

	var content string
	for chunk := range ch {
		if chunk.Type == ChunkContent {
			content += chunk.Payload
		}
	}
	assert.Equal(t, "ok", content)
}

// TestOpenAICompatibleClient_Chat_RequestTemperatureOverride 验证请求级温度覆盖配置级。
func TestOpenAICompatibleClient_Chat_RequestTemperatureOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody chatCompletionRequest
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, 1.2, reqBody.Temperature)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, ModelID: "m", Temperature: 0.5}
	ch, err := client.Chat(context.Background(), ChatRequest{Temperature: 1.2}, cfg)
	require.NoError(t, err)
	for range ch {
	}
}

// TestOpenAICompatibleClient_FetchModels_Success 验证正常拉取模型列表。
func TestOpenAICompatibleClient_FetchModels_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/models", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		resp := modelsListResponse{
			Object: "list",
			Data: []ModelInfo{
				{ID: "gpt-4o", Object: "model", Created: 1700000000, OwnedBy: "openai"},
				{ID: "gpt-4o-mini", Object: "model", Created: 1700000001, OwnedBy: "openai"},
			},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, APIKey: "test-key"}
	models, err := client.FetchModels(context.Background(), cfg)
	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, "gpt-4o", models[0].ID)
	assert.Equal(t, "openai", models[0].OwnedBy)
}

// TestOpenAICompatibleClient_FetchModels_Unauthorized 验证 401 返回 LLMError。
func TestOpenAICompatibleClient_FetchModels_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, APIKey: "bad-key"}
	_, err := client.FetchModels(context.Background(), cfg)

	require.Error(t, err)
	llmErr, ok := err.(*LLMError)
	require.True(t, ok)
	assert.False(t, llmErr.Retryable)
	assert.Contains(t, llmErr.Message, "认证失败")
}

// TestOpenAICompatibleClient_FetchModels_NetworkError 验证网络错误返回 Retryable 错误。
func TestOpenAICompatibleClient_FetchModels_NetworkError(t *testing.T) {
	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{
		APIHost: "http://127.0.0.1:1",
		APIKey:  "key",
		Timeout: 50 * time.Millisecond,
	}
	_, err := client.FetchModels(context.Background(), cfg)

	require.Error(t, err)
	llmErr, ok := err.(*LLMError)
	require.True(t, ok)
	assert.Equal(t, "network_error", llmErr.Code)
	assert.True(t, llmErr.Retryable)
}

// TestOpenAICompatibleClient_FetchModels_MissingAPIHost 验证 APIHost 为空时返回错误。
func TestOpenAICompatibleClient_FetchModels_MissingAPIHost(t *testing.T) {
	client := NewOpenAICompatibleClient()
	_, err := client.FetchModels(context.Background(), ProviderConfig{})
	require.Error(t, err)
	llmErr, ok := err.(*LLMError)
	require.True(t, ok)
	assert.Equal(t, "missing_api_host", llmErr.Code)
}

// TestOpenAICompatibleClient_FetchModels_EmptyList 验证空列表正常解析。
func TestOpenAICompatibleClient_FetchModels_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := modelsListResponse{Object: "list", Data: []ModelInfo{}}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	models, err := client.FetchModels(context.Background(), ProviderConfig{APIHost: server.URL})
	require.NoError(t, err)
	assert.Empty(t, models)
}

// TestOpenAICompatibleClient_Chat_500Retryable 验证 500 错误为 Retryable。
func TestOpenAICompatibleClient_Chat_500Retryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"server overload","code":"server_error"}}`))
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, ModelID: "m"}
	_, err := client.Chat(context.Background(), ChatRequest{}, cfg)

	require.Error(t, err)
	llmErr, ok := err.(*LLMError)
	require.True(t, ok)
	assert.True(t, llmErr.Retryable)
	assert.Contains(t, llmErr.Message, "server overload")
}

// TestOpenAICompatibleClient_Chat_429RateLimit 验证 429 错误为 Retryable。
func TestOpenAICompatibleClient_Chat_429RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, ModelID: "m"}
	_, err := client.Chat(context.Background(), ChatRequest{}, cfg)

	require.Error(t, err)
	llmErr, ok := err.(*LLMError)
	require.True(t, ok)
	assert.True(t, llmErr.Retryable)
	assert.Contains(t, llmErr.Message, "频繁")
}

// TestOpenAICompatibleClient_Chat_FinishReasonStop 验证 finish_reason=stop 时发送 done。
func TestOpenAICompatibleClient_Chat_FinishReasonStop(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n"))
		// 不发送 [DONE]
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, ModelID: "m"}
	ch, err := client.Chat(context.Background(), ChatRequest{}, cfg)
	require.NoError(t, err)

	var types []StreamChunkType
	for chunk := range ch {
		types = append(types, chunk.Type)
	}
	require.Len(t, types, 2)
	assert.Equal(t, ChunkContent, types[0])
	assert.Equal(t, ChunkDone, types[1])
}

// TestOpenAICompatibleClient_Chat_SSEErrorEvent 验证 SSE 流中的 error 事件。
func TestOpenAICompatibleClient_Chat_SSEErrorEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"error\":{\"message\":\"content filter\",\"code\":\"content_filter\"}}\n\n"))
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient()
	cfg := ProviderConfig{APIHost: server.URL, ModelID: "m"}
	ch, err := client.Chat(context.Background(), ChatRequest{}, cfg)
	require.NoError(t, err)

	var gotError bool
	for chunk := range ch {
		if chunk.Type == ChunkError && strings.Contains(chunk.Payload, "content filter") {
			gotError = true
		}
	}
	assert.True(t, gotError)
}

// TestLLMError_Error 验证 LLMError 的 Error() 格式化输出。
func TestLLMError_Error(t *testing.T) {
	err := &LLMError{Code: "test_code", Message: "test message", Retryable: true}
	assert.Contains(t, err.Error(), "test_code")
	assert.Contains(t, err.Error(), "test message")
	assert.Contains(t, err.Error(), "retryable=true")
}

// TestAPIError_Error 验证 apiError 的 Error() 格式化输出。
func TestAPIError_Error(t *testing.T) {
	// Code 非空分支
	e1 := &apiError{Code: "invalid_key", Message: "bad key"}
	assert.Equal(t, "api error [invalid_key]: bad key", e1.Error())

	// Code 为空分支
	e2 := &apiError{Message: "generic error"}
	assert.Equal(t, "api error: generic error", e2.Error())
}

// TestOpenAICompatibleClient_WithHTTPClient 验证使用外部 HTTPClient 构造。
func TestOpenAICompatibleClient_WithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 5 * time.Second}
	client := NewOpenAICompatibleClientWithHTTPClient(customClient)
	require.NotNil(t, client)
	assert.Equal(t, customClient, client.client)
}

// TestClassifyHTTPError 验证 HTTP 错误分类逻辑。
func TestClassifyHTTPError(t *testing.T) {
	tests := []struct {
		statusCode int
		apiErr     *apiError
		wantRetry  bool
		wantMsg    string
	}{
		{http.StatusUnauthorized, nil, false, "认证失败"},
		{http.StatusForbidden, nil, false, "被拒绝"},
		{http.StatusTooManyRequests, nil, true, "频繁"},
		{http.StatusNotFound, nil, false, "不存在"},
		{http.StatusInternalServerError, &apiError{Message: "overload"}, true, "overload"},
		{http.StatusBadRequest, nil, false, "失败"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			_, msg, retryable := classifyHTTPError(tt.statusCode, tt.apiErr)
			assert.Equal(t, tt.wantRetry, retryable)
			assert.Contains(t, msg, tt.wantMsg)
		})
	}
}

// TestIsNetworkError 验证网络错误判断。
func TestIsNetworkError(t *testing.T) {
	assert.True(t, isNetworkError(fmt.Errorf("connection refused")))
	assert.False(t, isNetworkError(nil))
}

// TestProviderFactory 验证 ProviderFactory 能根据配置创建对应适配器。
func TestProviderFactory(t *testing.T) {
	// 云端 Provider 路由到 OpenAIAdapter
	cfg := &entity.AppConfig{
		ProviderType: models.ProviderKimi,
		APIEndpoint:  "https://api.moonshot.cn",
		DefaultModel: "moonshot-v1-8k",
	}
	client := ProviderFactory(cfg)
	require.NotNil(t, client)

	// Ollama 本地 Provider 路由到 LocalAdapter
	cfg2 := &entity.AppConfig{
		ProviderType: models.ProviderOllama,
		APIEndpoint:  "http://localhost:11434",
		DefaultModel: "llama3",
	}
	client2 := ProviderFactory(cfg2)
	require.NotNil(t, client2)

	// 通用本地 Provider 路由到 OpenAIAdapter（apiKey 为空）
	cfg3 := &entity.AppConfig{
		ProviderType: models.ProviderLocal,
		APIEndpoint:  "http://localhost:8080",
		DefaultModel: "llama3",
	}
	client3 := ProviderFactory(cfg3)
	require.NotNil(t, client3)
}
