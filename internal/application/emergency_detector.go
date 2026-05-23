// Package application 实现应用用例层，编排领域对象完成完整业务流程。
package application

import (
	"strings"
)

// EmergencyLevel 表示紧急症状等级。
type EmergencyLevel string

const (
	LevelA    EmergencyLevel = "A"    // A级：立即就医，危及生命
	LevelB    EmergencyLevel = "B"    // B级：尽快就医，需及时关注
	LevelNone EmergencyLevel = "none" // 无紧急症状
)

// EmergencyCheckResult 紧急症状检测结果。
type EmergencyCheckResult struct {
	Level          EmergencyLevel // A / B / none
	Message        string         // 提示信息
	Action         string         // 建议操作
	MatchedKeyword string         // 命中的关键词（用于调试和反馈）
}

// aLevelKeywords 定义 A 级紧急症状关键词（立即就医）。
// 采用分词模式：空格分隔的多个词必须同时在文本中出现才算命中。
// 纯内存匹配，延迟 <5ms，不依赖 AI 响应或网络。
var aLevelKeywords = []string{
	// 心血管急症
	"胸痛 呼吸困难", "胸痛 喘不上气", "胸痛 放射", "胸痛 出汗",
	"剧烈胸痛", "胸闷 濒死感", "心悸 晕厥", "心跳骤停", "心脏骤停",
	// 神经系统急症
	"意识丧失", "昏迷", "昏倒", "不省人事", "抽搐", "癫痫发作", "持续抽搐",
	"突发偏瘫", "口角歪斜", "言语不清", "剧烈头痛 呕吐",
	"脑出血", "脑卒中", "中风",
	// 呼吸系统急症
	"严重呼吸困难", "窒息", "卡喉", "异物卡喉", "呼吸停止", "气道阻塞",
	"喉头水肿", "哮喘 危重", "喘不上气 无法说话",
	// 外伤/出血
	"大出血", "大量出血", "血流不止", "严重外伤", "车祸", "坠落", "高空坠落",
	"刀伤", "刺伤", "断肢", "开放性骨折", "头部外伤 昏迷",
	"烧伤 大面积", "三度烧伤", "化学烧伤", "电击伤", "溺水", "溺亡",
	// 中毒
	"中毒", "误食", "药物过量", "服药过量", "农药中毒", "一氧化碳中毒",
	"食物中毒 休克",
	// 严重过敏
	"严重过敏", "过敏性休克", "喉咙肿胀", "喉头水肿", "皮疹 呼吸困难",
	"血管性水肿",
	// 妇产科急症
	"孕妇 出血", "孕妇 腹痛", "孕妇 破水", "产前大出血", "产后大出血",
	"胎动消失", "宫外孕破裂",
	// 儿科急症
	"新生儿 发热", "婴儿 高烧", "婴儿 拒食", "高热惊厥", "婴儿 呼吸困难",
	// 其他危急
	"休克", "低血压 昏迷", "低血糖 昏迷", "酮症酸中毒", "高热 昏迷",
}

// bLevelKeywords 定义 B 级紧急症状关键词（尽快就医）。
var bLevelKeywords = []string{
	// 发热
	"持续高热", "高烧三天", "发烧超过三天", "发热 三天", "反复发热 一周",
	"低热 消瘦", "午后潮热",
	// 消化系统
	"剧烈腹痛", "肚子剧痛", "腹痛难忍", "腹痛 板状腹", "呕血", "黑便",
	"便血", "持续呕吐", "无法进食", "严重腹泻", "腹泻 脱水", "黄疸 加重",
	"腹胀 不排气",
	// 泌尿系统
	"血尿", "尿血", "尿液带血", "排尿困难", "尿潴留", "少尿 无尿", "腰痛 发热",
	// 眼科
	"视力突然下降", "突然看不见", "视力模糊", "视物变形", "飞蚊症 闪光",
	"眼痛 头痛", "眼红 视力下降",
	// 神经系统
	"头痛 颈部僵硬", "头痛 呕吐", "突发眩晕", "肢体无力", "半身麻木",
	"手脚麻木 无力", "行走不稳",
	// 心血管
	"心悸 胸闷", "心跳过快", "心律不齐", "胸痛 不剧烈", "水肿 气短",
	// 内分泌/代谢
	"血糖 极高", "血糖 极低", "糖尿病 昏迷", "多饮 多尿 消瘦",
	// 其他
	"咯血", "痰中带血", "淋巴结肿大", "不明原因消瘦", "不明原因发热",
	"皮肤瘀斑", "关节红肿热痛",
}

// EvaluateEmergency 检查文本是否包含紧急症状。
// 按 A级 > B级 优先级匹配，命中即短路返回。
// 采用本地关键词匹配，延迟 <5ms，独立于 AI 回复流程。
func EvaluateEmergency(text string) *EmergencyCheckResult {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return &EmergencyCheckResult{
			Level:   LevelNone,
			Message: "",
			Action:  "",
		}
	}

	// A 级检测（立即就医）
	for _, kw := range aLevelKeywords {
		if containsAll(trimmed, kw) {
			return &EmergencyCheckResult{
				Level:          LevelA,
				Message:        "检测到可能危及生命的紧急症状，请立即就医或拨打急救电话（如 120）。",
				Action:         "立即就医 / 拨打 120",
				MatchedKeyword: kw,
			}
		}
	}

	// B 级检测（尽快就医）
	for _, kw := range bLevelKeywords {
		if containsAll(trimmed, kw) {
			return &EmergencyCheckResult{
				Level:          LevelB,
				Message:        "检测到可能需要尽快就医的症状，建议尽快前往医院就诊。",
				Action:         "尽快就医",
				MatchedKeyword: kw,
			}
		}
	}

	return &EmergencyCheckResult{
		Level:   LevelNone,
		Message: "",
		Action:  "",
	}
}

// containsAll 检查文本中是否同时包含关键词中的所有分词。
// 关键词以空格分隔多个词，所有词都出现时返回 true。
// 匹配不区分大小写。
func containsAll(text, keyword string) bool {
	parts := strings.Fields(keyword)
	lowerText := strings.ToLower(text)
	for _, part := range parts {
		if !strings.Contains(lowerText, strings.ToLower(part)) {
			return false
		}
	}
	return true
}
