package models

// SensitivityLevel 表示信息敏感度分级。
type SensitivityLevel int

// noinspection GoUnusedConst

const (
	P1Public       SensitivityLevel = iota // 公开信息，无需处理
	P2Internal                             // 内部信息，软替换（可恢复）
	P3Confidential                         // 机密信息，硬替换（不可逆）
)

// SensitiveEntity 表示识别到的敏感实体。
type SensitiveEntity struct {
	Text        string
	Type        string // 如 "姓名", "身份证号", "疾病名"
	Level       SensitivityLevel
	StartPos    int
	EndPos      int
	Score       float32 // 置信度 [0,1]，L2 NER 模型输出用
	Placeholder string  // 该实体被替换后的占位符，如 "{{phone_a3f7b2d1}}"
}

// DeidentifyResult 脱敏处理结果。
type DeidentifyResult struct {
	OriginalText string
	SafeText     string
	Entities     []SensitiveEntity
	Placeholder  map[string]string // 占位符 -> 原始值映射（P2 级用于还原）
	LocalRestore map[string]string // 本地还原映射（P2+P3），仅用于本地事实还原等场景，不出网。
}
