package dto

import (
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
)

func TestToConversationDTO(t *testing.T) {
	conv := &entity.Conversation{
		ID:        "conv_123",
		Title:     "测试会话",
		Model:     models.ProviderKimi,
		CreatedAt: time.Unix(1700000000, 0),
		Messages: []entity.Message{
			{Role: models.RoleUser, Content: "你好", Timestamp: time.Unix(1700000001, 0)},
			{Role: models.RoleAssistant, Content: "你好！", Timestamp: time.Unix(1700000002, 0)},
		},
	}

	dto := ToConversationDTO(conv)

	if dto.ID != "conv_123" {
		t.Errorf("expected ID %q, got %q", "conv_123", dto.ID)
	}
	if dto.Title != "测试会话" {
		t.Errorf("expected Title %q, got %q", "测试会话", dto.Title)
	}
	if dto.Model != string(models.ProviderKimi) {
		t.Errorf("expected Model %q, got %q", models.ProviderKimi, dto.Model)
	}
	if dto.CreatedAt != 1700000000 {
		t.Errorf("expected CreatedAt 1700000000, got %d", dto.CreatedAt)
	}
	if len(dto.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(dto.Messages))
	}
	if dto.Messages[0].Role != "user" || dto.Messages[0].Content != "你好" {
		t.Errorf("unexpected first message: %+v", dto.Messages[0])
	}
	if dto.Messages[0].Timestamp != 1700000001 {
		t.Errorf("expected first message timestamp 1700000001, got %d", dto.Messages[0].Timestamp)
	}
}

func TestFromMessageDTO(t *testing.T) {
	tests := []struct {
		name     string
		dto      MessageDTO
		wantErr  bool
		wantRole models.Role
	}{
		{"valid user", MessageDTO{Role: "user", Content: "hello"}, false, models.RoleUser},
		{"valid assistant", MessageDTO{Role: "assistant", Content: "hi"}, false, models.RoleAssistant},
		{"valid system", MessageDTO{Role: "system", Content: "sys"}, false, models.RoleSystem},
		{"invalid role", MessageDTO{Role: "bot", Content: "x"}, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := FromMessageDTO(tt.dto)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FromMessageDTO() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if msg.Role != tt.wantRole {
				t.Errorf("expected role %v, got %v", tt.wantRole, msg.Role)
			}
			if msg.Content != tt.dto.Content {
				t.Errorf("expected content %q, got %q", tt.dto.Content, msg.Content)
			}
		})
	}
}
