package usecase

import (
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestSensitiveDetector_NilFact(t *testing.T) {
	t.Parallel()
	sd := NewSensitiveDetector()
	assert.False(t, sd.Detect(nil))
}

func TestSensitiveDetector_NormalFact(t *testing.T) {
	t.Parallel()
	sd := NewSensitiveDetector()
	f := entity.NewExtractedFact("用户", "喜欢", "跑步", 0.8, []string{"msg_001"})
	assert.False(t, sd.Detect(f))
}

func TestSensitiveDetector_IDCard(t *testing.T) {
	t.Parallel()
	sd := NewSensitiveDetector()
	f := entity.NewExtractedFact("用户", "身份证", "110101199001011234", 0.9, []string{"msg_001"})
	assert.True(t, sd.Detect(f))
}

func TestSensitiveDetector_PhoneNumber(t *testing.T) {
	t.Parallel()
	sd := NewSensitiveDetector()
	f := entity.NewExtractedFact("用户", "手机号", "13800138000", 0.9, []string{"msg_001"})
	assert.True(t, sd.Detect(f))
}

func TestSensitiveDetector_Email(t *testing.T) {
	t.Parallel()
	sd := NewSensitiveDetector()
	f := entity.NewExtractedFact("用户", "邮箱", "user@example.com", 0.9, []string{"msg_001"})
	assert.True(t, sd.Detect(f))
}

func TestSensitiveDetector_MedicalDisease(t *testing.T) {
	t.Parallel()
	sd := NewSensitiveDetector()
	f := entity.NewExtractedFact("用户", "患有", "高血压", 0.9, []string{"msg_001"})
	assert.True(t, sd.Detect(f))
}

func TestSensitiveDetector_MedicalDrug(t *testing.T) {
	t.Parallel()
	sd := NewSensitiveDetector()
	f := entity.NewExtractedFact("用户", "服用", "阿司匹林", 0.9, []string{"msg_001"})
	assert.True(t, sd.Detect(f))
}

func TestSensitiveDetector_MixedContent(t *testing.T) {
	t.Parallel()
	sd := NewSensitiveDetector()
	// subject 含敏感词，predicate/object 正常
	f := entity.NewExtractedFact("糖尿病患者", "喜欢", "跑步", 0.8, []string{"msg_001"})
	assert.True(t, sd.Detect(f))
}

func TestSensitiveDetector_BankCard(t *testing.T) {
	t.Parallel()
	sd := NewSensitiveDetector()
	f := entity.NewExtractedFact("用户", "银行卡", "6222021234567890123", 0.9, []string{"msg_001"})
	assert.True(t, sd.Detect(f))
}
