package entity

import (
	"strings"
	"testing"

	"github.com/hzhan516/medmemo/pkg/models"
)

func TestNewFamilyMember(t *testing.T) {
	m := NewFamilyMember("张三", GenderMale)
	if m.ID == "" {
		t.Error("expected non-empty ID")
	}
	if m.Name != "张三" {
		t.Errorf("expected name %q, got %q", "张三", m.Name)
	}
	if m.Gender != GenderMale {
		t.Errorf("expected gender %v, got %v", GenderMale, m.Gender)
	}
	if len(m.Relations) != 0 {
		t.Errorf("expected empty relations, got %d", len(m.Relations))
	}
	if len(m.Diseases) != 0 {
		t.Errorf("expected empty diseases, got %d", len(m.Diseases))
	}
	if !strings.HasPrefix(string(m.ID), "member_") {
		t.Errorf("expected ID to start with 'member_', got %q", m.ID)
	}
}

func TestFamilyMember_AddRelation(t *testing.T) {
	m := NewFamilyMember("张三", GenderMale)
	err := m.AddRelation("member_456", RelationSpouse)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(m.Relations))
	}
	rel := m.Relations[0]
	if rel.ToMemberID != models.MemberID("member_456") {
		t.Errorf("expected ToMemberID %q, got %q", "member_456", rel.ToMemberID)
	}
	if rel.Type != RelationSpouse {
		t.Errorf("expected type %v, got %v", RelationSpouse, rel.Type)
	}
}
