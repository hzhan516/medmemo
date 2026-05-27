// Package usecase 应用用例层，编排领域对象完成完整业务流程。
package usecase

import (
	"regexp"
	"strings"

	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// =============================================================================
// 敏感信息检测器
// =============================================================================

// SensitiveDetector 检测事实三元组中是否包含敏感信息。
// 敏感信息分为两类：
//  1. PII（个人身份信息）：身份证号、手机号、银行卡、邮箱
//  2. 医学敏感信息：疾病名、药品名等
//
// 检测仅作标记用途，不影响事实的正常存储与检索流程。
type SensitiveDetector struct {
	piiPatterns       []*regexp.Regexp
	medicalKeywords   []string
	medicalKeywordSet map[string]struct{}
}

// NewSensitiveDetector 创建敏感信息检测器，预编译所有正则规则。
func NewSensitiveDetector() *SensitiveDetector {
	sd := &SensitiveDetector{
		medicalKeywords: defaultMedicalKeywords(),
	}

	// 预编译 PII 正则模式
	sd.piiPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\b\d{17}[\dXx]\b`),                               // 18位身份证号
		regexp.MustCompile(`\b\d{15}\b`),                                     // 15位身份证号
		regexp.MustCompile(`\b1[3-9]\d{9}\b`),                                // 大陆手机号
		regexp.MustCompile(`\b\d{16,19}\b`),                                  // 银行卡号
		regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`), // 邮箱
	}

	// 构建医学关键词快速查找集合
	sd.medicalKeywordSet = make(map[string]struct{}, len(sd.medicalKeywords))
	for _, kw := range sd.medicalKeywords {
		sd.medicalKeywordSet[strings.ToLower(kw)] = struct{}{}
	}

	return sd
}

// Detect 检测给定事实是否包含敏感信息，返回 true 表示敏感。
func (sd *SensitiveDetector) Detect(f *entity.ExtractedFact) bool {
	if f == nil {
		return false
	}

	// 合并三元组文本进行统一检测
	combined := f.Subject + " " + f.Predicate + " " + f.Object

	// PII 检测
	for _, re := range sd.piiPatterns {
		if re.MatchString(combined) {
			return true
		}
	}

	// 医学敏感关键词检测
	lower := strings.ToLower(combined)
	for kw := range sd.medicalKeywordSet {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	return false
}

// defaultMedicalKeywords 返回内置的医学敏感关键词列表。
// 覆盖常见疾病名和药品名，作为初始版本。后续可通过外部规则文件扩展。
func defaultMedicalKeywords() []string {
	return []string{
		// 常见疾病
		"高血压", "糖尿病", "心脏病", "冠心病", "脑梗", "中风", "肿瘤", "癌症",
		"白血病", "艾滋病", "肝炎", "肺结核", "哮喘", "癫痫", "抑郁症",
		"焦虑症", "精神分裂症", "帕金森", "阿尔茨海默", "痴呆",
		"肾衰竭", "尿毒症", "肝硬化", "胆结石", "肾结石",
		"心肌梗死", "心力衰竭", "心律失常", "动脉硬化",
		"脑出血", "脑血栓", "偏瘫", "截瘫",
		"红斑狼疮", "类风湿", "强直性脊柱炎", "牛皮癣",
		"甲亢", "甲减", "甲状腺", "糖尿病足",
		"白内障", "青光眼", "视网膜病变",
		"骨质疏松", "骨折", "腰椎间盘突出", "颈椎病",
		// 常见药品
		"阿司匹林", "布洛芬", "对乙酰氨基酚", "头孢", "阿莫西林",
		"青霉素", "红霉素", "阿奇霉素", "左氧氟沙星", "甲硝唑",
		"二甲双胍", "胰岛素", "格列", "他汀", "硝苯地平",
		"氨氯地平", "缬沙坦", "厄贝沙坦", "贝那普利", "美托洛尔",
		"地高辛", "硝酸甘油", "华法林", "氯吡格雷",
		"奥美拉唑", "雷贝拉唑", "泮托拉唑",
		"蒙脱石散", "多潘立酮", "莫沙必利",
		"西替利嗪", "氯雷他定", "扑尔敏",
		"地塞米松", "泼尼松", "甲泼尼龙",
		"丙硫氧嘧啶", "甲巯咪唑",
		"左旋甲状腺素", "优甲乐",
	}
}
