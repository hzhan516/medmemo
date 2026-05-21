package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/medmemo/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalAdapter_Chat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/chat", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var req ollamaChatRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "llama3.1", req.Model)
		assert.False(t, req.Stream)

		resp := ollamaChatResponse{
			Message: ollamaMessage{Role: "assistant", Content: "Hello from Ollama"},
			Done:    true,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewLocalAdapter(server.URL, "llama3.1")
	msgs := []models.Message{{Role: models.RoleUser, Content: "Hi"}}

	reply, err := adapter.Chat(context.Background(), msgs)
	require.NoError(t, err)
	assert.Equal(t, "Hello from Ollama", reply)
}

func TestLocalAdapter_Chat_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaChatResponse{
			Error: "model not found",
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewLocalAdapter(server.URL, "unknown-model")
	msgs := []models.Message{{Role: models.RoleUser, Content: "Hi"}}

	_, err := adapter.Chat(context.Background(), msgs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model not found")
}

func TestLocalAdapter_StreamChat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaChatRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.True(t, req.Stream)

		w.Header().Set("Content-Type", "application/x-ndjson")
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		chunks := []string{"Hello", " ", "world", "!"}
		for _, chunk := range chunks {
			resp := ollamaChatResponse{
				Message: ollamaMessage{Role: "assistant", Content: chunk},
				Done:    false,
			}
			data, _ := json.Marshal(resp)
			_, _ = w.Write(data)
			_, _ = w.Write([]byte("\n"))
			flusher.Flush()
		}
		// 发送结束标记
		endResp := ollamaChatResponse{Done: true}
		data, _ := json.Marshal(endResp)
		_, _ = w.Write(data)
		_, _ = w.Write([]byte("\n"))
	}))
	defer server.Close()

	adapter := NewLocalAdapter(server.URL, "llama3.1")
	msgs := []models.Message{{Role: models.RoleUser, Content: "Hi"}}

	var builder strings.Builder
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usage, err := adapter.StreamChat(ctx, msgs, func(chunk string) {
		builder.WriteString(chunk)
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello world!", builder.String())
	assert.Nil(t, usage) // 无 usage 数据时返回 nil
}

func TestLocalAdapter_CheckAvailability_Available(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/tags", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := NewLocalAdapter(server.URL, "llama3.1")
	ok, msg := adapter.CheckAvailability(context.Background())
	assert.True(t, ok)
	assert.Equal(t, "available", msg)
}

func TestLocalAdapter_CheckAvailability_Unavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := NewLocalAdapter(server.URL, "llama3.1")
	ok, msg := adapter.CheckAvailability(context.Background())
	assert.False(t, ok)
	assert.Contains(t, msg, "500")
}

func TestLocalAdapter_CheckAvailability_NotReachable(t *testing.T) {
	// 使用一个不可达的地址
	adapter := NewLocalAdapter("http://localhost:59999", "llama3.1")
	ok, msg := adapter.CheckAvailability(context.Background())
	assert.False(t, ok)
	assert.Contains(t, msg, "not reachable")
}
