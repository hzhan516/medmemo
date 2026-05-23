package entity

import (
	"strings"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/pkg/models"
)

func TestNewConversation(t *testing.T) {
	conv := NewConversation(models.ProviderKimi)
	if conv.ID == "" {
		t.Error("expected non-empty ID")
	}
	if conv.Title != "" {
		t.Errorf("expected empty title, got %q", conv.Title)
	}
	if conv.Model != models.ProviderKimi {
		t.Errorf("expected model %v, got %v", models.ProviderKimi, conv.Model)
	}
	if len(conv.Messages) != 0 {
		t.Errorf("expected empty messages, got %d", len(conv.Messages))
	}
	if conv.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if conv.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestConversation_AddMessage(t *testing.T) {
	conv := NewConversation(models.ProviderKimi)
	before := conv.UpdatedAt
	time.Sleep(10 * time.Millisecond) // 确保时间戳有变化

	conv.AddMessage(models.RoleUser, "hello")

	if len(conv.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(conv.Messages))
	}
	msg := conv.Messages[0]
	if msg.Role != models.RoleUser {
		t.Errorf("expected role %v, got %v", models.RoleUser, msg.Role)
	}
	if msg.Content != "hello" {
		t.Errorf("expected content %q, got %q", "hello", msg.Content)
	}
	if !strings.HasPrefix(msg.ID, "msg_") {
		t.Errorf("expected message ID to start with 'msg_', got %q", msg.ID)
	}
	if msg.Timestamp.IsZero() {
		t.Error("expected message timestamp to be set")
	}
	if !conv.UpdatedAt.After(before) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestConversation_Rename(t *testing.T) {
	conv := NewConversation(models.ProviderKimi)
	before := conv.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	conv.Rename("new title")

	if conv.Title != "new title" {
		t.Errorf("expected title %q, got %q", "new title", conv.Title)
	}
	if !conv.UpdatedAt.After(before) {
		t.Error("expected UpdatedAt to be updated")
	}
}
