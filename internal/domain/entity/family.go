package entity

import (
	"fmt"
	"time"

	"github.com/medmemo/medmemo/pkg/models"
)

// RelationType 表示家族成员间的亲属关系类型。
type RelationType string

const (
	RelationParent  RelationType = "PARENT"
	RelationChild   RelationType = "CHILD"
	RelationSpouse  RelationType = "SPOUSE"
	RelationSibling RelationType = "SIBLING"
)

// Gender 表示性别。
type Gender string

const (
	GenderMale   Gender = "MALE"
	GenderFemale Gender = "FEMALE"
	GenderOther  Gender = "OTHER"
)

// FamilyMember 表示家族健康图谱中的一个成员节点。
type FamilyMember struct {
	ID        models.MemberID
	Name      string
	Gender    Gender
	BirthDate *time.Time // 指针类型允许为空
	Relations []FamilyRelation
	Diseases  []DiseaseRecord
	CreatedAt time.Time
}

// FamilyRelation 表示两个家族成员之间的关系边。
type FamilyRelation struct {
	ToMemberID models.MemberID
	Type       RelationType
}

// DiseaseRecord 表示成员的一条疾病历史记录。
type DiseaseRecord struct {
	DiseaseName string
	DiagnosedAt *time.Time
	Status      string // 如 " cured", "managing", "chronic"
	Notes       string
}

// NewFamilyMember 创建家族成员实体，ID 由系统生成。
func NewFamilyMember(name string, gender Gender) *FamilyMember {
	now := time.Now()
	return &FamilyMember{
		ID:        models.MemberID(fmt.Sprintf("member_%d", now.UnixNano())),
		Name:      name,
		Gender:    gender,
		Relations: make([]FamilyRelation, 0),
		Diseases:  make([]DiseaseRecord, 0),
		CreatedAt: now,
	}
}

// AddRelation 添加亲属关系，返回 error 以支持环检测等校验。
func (m *FamilyMember) AddRelation(to models.MemberID, relType RelationType) error {
	// TODO(作者): 实现血缘关系环检测 [Issue#002]
	m.Relations = append(m.Relations, FamilyRelation{
		ToMemberID: to,
		Type:       relType,
	})
	return nil
}
