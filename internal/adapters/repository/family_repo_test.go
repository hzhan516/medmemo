package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/hzhan516/medmemo/internal/domain/entity"
)

func TestFamilyRepoKuzu_BasicOperations(t *testing.T) {
	repo := NewFamilyRepoKuzu()
	ctx := context.Background()

	t.Run("SaveMember", func(t *testing.T) {
		err := repo.SaveMember(ctx, &entity.FamilyMember{})
		assert.Error(t, err)
	})

	t.Run("GetMemberByID", func(t *testing.T) {
		_, err := repo.GetMemberByID(ctx, "member_1")
		assert.Error(t, err)
	})

	t.Run("ListAllMembers", func(t *testing.T) {
		_, err := repo.ListAllMembers(ctx)
		assert.Error(t, err)
	})

	t.Run("DeleteMember", func(t *testing.T) {
		err := repo.DeleteMember(ctx, "member_1")
		assert.Error(t, err)
	})

	t.Run("FindRelations", func(t *testing.T) {
		_, err := repo.FindRelations(ctx, "member_1")
		assert.Error(t, err)
	})

	t.Run("FindByDisease", func(t *testing.T) {
		_, err := repo.FindByDisease(ctx, "高血压")
		assert.Error(t, err)
	})
}
