package repository

import (
	"context"
	"fmt"

	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/pkg/models"
)

// FamilyRepoKuzu 基于 Kùzǔ 图数据库的家族关系仓库实现。
type FamilyRepoKuzu struct {
	// TODO(作者): 接入 Kùzǔ Go 绑定 [Issue#016]
}

// NewFamilyRepoKuzu 构造函数。
func NewFamilyRepoKuzu() *FamilyRepoKuzu {
	return &FamilyRepoKuzu{}
}

// SaveMember 实现 port.FamilyRepository。
func (r *FamilyRepoKuzu) SaveMember(ctx context.Context, member *entity.FamilyMember) error {
	return fmt.Errorf("FamilyRepoKuzu.SaveMember not implemented")
}

// GetMemberByID 实现 port.FamilyRepository。
func (r *FamilyRepoKuzu) GetMemberByID(ctx context.Context, id models.MemberID) (*entity.FamilyMember, error) {
	return nil, fmt.Errorf("FamilyRepoKuzu.GetMemberByID not implemented")
}

// ListAllMembers 实现 port.FamilyRepository。
func (r *FamilyRepoKuzu) ListAllMembers(ctx context.Context) ([]*entity.FamilyMember, error) {
	return nil, fmt.Errorf("FamilyRepoKuzu.ListAllMembers not implemented")
}

// DeleteMember 实现 port.FamilyRepository。
func (r *FamilyRepoKuzu) DeleteMember(ctx context.Context, id models.MemberID) error {
	return fmt.Errorf("FamilyRepoKuzu.DeleteMember not implemented")
}

// FindRelations 实现 port.FamilyRepository。
func (r *FamilyRepoKuzu) FindRelations(ctx context.Context, id models.MemberID) ([]entity.FamilyRelation, error) {
	return nil, fmt.Errorf("FamilyRepoKuzu.FindRelations not implemented")
}

// FindByDisease 实现 port.FamilyRepository。
func (r *FamilyRepoKuzu) FindByDisease(ctx context.Context, diseaseName string) ([]*entity.FamilyMember, error) {
	// TODO(作者): Cypher 查询实现 [Issue#017]
	return nil, fmt.Errorf("FamilyRepoKuzu.FindByDisease not implemented")
}
