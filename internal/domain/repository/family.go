package repository

import (
	"context"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
)

// FamilyRepository 定义家族成员与关系的持久化接口。
type FamilyRepository interface {
	// SaveMember 保存或更新家族成员。
	SaveMember(ctx context.Context, member *entity.FamilyMember) error

	// GetMemberByID 按 ID 获取家族成员。
	GetMemberByID(ctx context.Context, id models.MemberID) (*entity.FamilyMember, error)

	// ListAllMembers 列出所有家族成员。
	ListAllMembers(ctx context.Context) ([]*entity.FamilyMember, error)

	// DeleteMember 删除家族成员及其关联关系。
	DeleteMember(ctx context.Context, id models.MemberID) error

	// FindRelations 查询某成员的所有关系。
	FindRelations(ctx context.Context, id models.MemberID) ([]entity.FamilyRelation, error)

	// FindByDisease 按疾病名称检索家族中有该病历史的成员。
	FindByDisease(ctx context.Context, diseaseName string) ([]*entity.FamilyMember, error)
}
