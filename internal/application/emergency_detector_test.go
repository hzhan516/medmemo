package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateEmergency_ALevel(t *testing.T) {
		t.Parallel()
	tests := []struct {
		name     string
		text     string
		expected string // 期望命中的关键词片段
	}{
		{"chest_pain_breath", "我胸痛伴呼吸困难，很难受", "胸痛 呼吸困难"},
		{"chest_pain_radiate", "胸痛放射到左臂还出汗", "胸痛 放射"},
		{"unconscious", "患者意识丧失，需要急救", "意识丧失"},
		{"coma", "病人昏迷不醒，叫不醒", "昏迷"},
		{"severe_allergy", "严重过敏反应，喉咙肿胀", "严重过敏"},
		{"anaphylaxis", "出现过敏性休克症状", "过敏性休克"},
		{"bleeding", "大出血，血流不止", "大出血"},
		{"car_accident", "出了车祸，严重外伤", "车祸"},
		{"fall", "从高空坠落，骨折了", "高空坠落"},
		{"drowning", "孩子溺水了，没有呼吸", "溺水"},
		{"poisoning", "误食农药中毒，快帮忙", "农药中毒"},
		{"overdose", "药物过量服用怎么办", "药物过量"},
		{"seizure", "突然癫痫发作，持续抽搐", "癫痫发作"},
		{"stroke", "突发偏瘫，口角歪斜，可能是脑卒中", "脑卒中"},
		{"burn", "大面积三度烧伤，非常疼痛", "三度烧伤"},
		{"pregnant_bleeding", "孕妇出血了，很多血", "孕妇 出血"},
		{"baby_fever", "新生儿发热，体温很高", "新生儿 发热"},
		{"shock", "病人休克了，血压测不到", "休克"},
		{"choke", "吃东西卡喉了，喘不上气", "异物卡喉"},
		{"head_trauma", "头部外伤后昏迷不醒", "头部外伤 昏迷"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvaluateEmergency(tt.text)
			assert.Equal(t, LevelA, result.Level, "期望 A 级")
			assert.NotEmpty(t, result.Message)
			assert.Contains(t, result.Action, "120")
			assert.NotEmpty(t, result.MatchedKeyword)
		})
	}
}

func TestEvaluateEmergency_BLevel(t *testing.T) {
		t.Parallel()
	tests := []struct {
		name string
		text string
	}{
		{"high_fever", "持续高热三天不退"},
		{"high_fever_3days", "发烧超过三天了，还是39度"},
		{"abdominal_pain", "剧烈腹痛，难以忍受"},
		{"blood_in_urine", "发现血尿，尿液带血"},
		{"vision_loss", "视力突然下降，看不清东西"},
		{"vomit", "持续呕吐，无法进食"},
		{"diarrhea", "严重腹泻，已经出现脱水症状"},
		{"headache_stiff_neck", "头痛伴有颈部僵硬"},
		{"limb_weakness", "肢体无力，半身麻木"},
		{"palpitation", "心悸胸闷，心跳过快"},
		{"blood_sugar", "血糖极高，头晕恶心"},
		{"hemoptysis", "咯血，痰中带血"},
		{"jaundice", "黄疸加重，皮肤很黄"},
		{" swollen_lymph", "淋巴结肿大，不明原因消瘦"},
		{"eye_pain", "眼痛头痛，视力模糊"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvaluateEmergency(tt.text)
			assert.Equal(t, LevelB, result.Level, "期望 B 级")
			assert.NotEmpty(t, result.Message)
			assert.NotEmpty(t, result.Action)
		})
	}
}

func TestEvaluateEmergency_None(t *testing.T) {
		t.Parallel()
	tests := []string{
		"今天天气不错，想了解一下健康饮食",
		"我最近睡眠质量不好，有什么建议",
		"帮我推荐一下补充维生素的食物",
		"想了解一下预防流感的方法",
		"", // 空输入
	}

	for _, text := range tests {
		result := EvaluateEmergency(text)
		assert.Equal(t, LevelNone, result.Level, "文本: %s", text)
		assert.Empty(t, result.Message)
		assert.Empty(t, result.Action)
	}
}

func TestEvaluateEmergency_Priority_AB(t *testing.T) {
		t.Parallel()
	// 同时包含 A 级和 B 级关键词时，应优先返回 A 级
	text := "患者胸痛伴呼吸困难并且持续高热三天"
	result := EvaluateEmergency(text)
	assert.Equal(t, LevelA, result.Level, "A 级应优先于 B 级")
}

func TestEvaluateEmergency_CaseInsensitive(t *testing.T) {
		t.Parallel()
	// 中文无大小写差异，验证混合输入仍能正确匹配
	text := "胸痛，伴呼吸困难！"
	result := EvaluateEmergency(text)
	assert.Equal(t, LevelA, result.Level, "标点不影响匹配")
}

func TestEvaluateEmergency_PartialMatch(t *testing.T) {
		t.Parallel()
	// 仅包含部分关键词不应命中
	result := EvaluateEmergency("我只是有点胸闷")
	assert.Equal(t, LevelNone, result.Level, "胸闷不应命中 A 级")

	result = EvaluateEmergency("有点发烧，但只持续了一天")
	assert.Equal(t, LevelNone, result.Level, "仅发烧不应命中 B 级")
}

// TestEvaluateEmergency_NegativeCases 验证不含急症关键词的正常文本不触发误报。
func TestEvaluateEmergency_NegativeCases(t *testing.T) {
		t.Parallel()
	tests := []struct {
		name string
		text string
	}{
		{"chest_pain_unrelated", "胸痛定是电视剧里常见的桥段"},
		{"radiation_general", "放射性物质检测在实验室进行"},
		{"chest_pain_context", "胸口痛这个词在小说里经常描写"},
		{"normal_fever", "我有点发烧，可能是感冒了"},
		{"normal_cough", "最近咳嗽有点厉害，想喝点止咳糖浆"},
		{"normal_headache", "头痛可能是没睡好"},
		{"normal_stomach", "肚子有点不舒服，可能是吃坏东西了"},
		{"normal_exercise", "运动后胸闷气短是正常的"},
		{"normal_stress", "工作压力大，胸口有点闷"},
		{"normal_fatigue", "最近比较疲劳，胸口偶尔不舒服"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvaluateEmergency(tt.text)
			assert.Equal(t, LevelNone, result.Level, "文本不应触发急症: %s", tt.text)
			assert.Empty(t, result.Message)
			assert.Empty(t, result.Action)
		})
	}
}

// TestEvaluateEmergency_BoundaryWords 验证边界词汇不触发误报。
func TestEvaluateEmergency_BoundaryWords(t *testing.T) {
		t.Parallel()
	// "胸痛" 与 "胸口痛" 是不同词汇，但 "胸口痛" 不含在关键词中
	result := EvaluateEmergency("胸口痛，但不是很严重")
	assert.Equal(t, LevelNone, result.Level, "胸口痛不应命中 A 级(胸痛)")

	// "胸闷" 单独出现不应触发 A 级
	result = EvaluateEmergency("有点胸闷，深呼吸后缓解")
	assert.Equal(t, LevelNone, result.Level, "单纯胸闷不应命中 A 级")

	// "胸痛" 单独出现（无伴随症状）不应触发 A 级
	result = EvaluateEmergency("偶尔胸痛，休息后好转")
	assert.Equal(t, LevelNone, result.Level, "单纯胸痛无伴随症状不应命中 A 级")

	// "放射" 单独出现不应触发
	result = EvaluateEmergency("放射科医生在检查")
	assert.Equal(t, LevelNone, result.Level, "放射不应命中 A 级")

	// "出汗" 单独出现不应触发
	result = EvaluateEmergency("运动后出汗很多")
	assert.Equal(t, LevelNone, result.Level, "出汗不应命中 A 级")
}

// TestEvaluateEmergency_OverlapABPriority 验证 A/B 重叠时 A 优先。
func TestEvaluateEmergency_OverlapABPriority(t *testing.T) {
		t.Parallel()
	// 同时包含 A 级(胸痛+呼吸困难) 和 B 级(心悸+胸闷) — A 应优先
	text := "胸痛伴呼吸困难，同时心悸胸闷"
	result := EvaluateEmergency(text)
	assert.Equal(t, LevelA, result.Level, "A 级应优先于 B 级")
	assert.Contains(t, result.Action, "120")

	// 同时包含 A 级(剧烈胸痛) 和 B 级(胸痛 不剧烈) — A 应优先
	text = "剧烈胸痛，虽然不剧烈时也有不适"
	result = EvaluateEmergency(text)
	assert.Equal(t, LevelA, result.Level, "剧烈胸痛(A)应优先于胸痛不剧烈(B)")

	// 同时包含 A 级(昏迷) 和 B 级(头痛+颈部僵硬) — A 应优先
	text = "患者昏迷不醒，之前头痛颈部僵硬"
	result = EvaluateEmergency(text)
	assert.Equal(t, LevelA, result.Level, "昏迷(A)应优先于头痛颈部僵硬(B)")
}

// TestEvaluateEmergency_RadiationChestPain 验证放射性胸痛保持 A 级。
func TestEvaluateEmergency_RadiationChestPain(t *testing.T) {
		t.Parallel()
	text := "放射性胸痛，向左肩放射"
	result := EvaluateEmergency(text)
	assert.Equal(t, LevelA, result.Level, "放射性胸痛应保持 A 级")
	assert.Contains(t, result.MatchedKeyword, "胸痛")
	assert.Contains(t, result.Action, "120")
}

func TestContainsAll(t *testing.T) {
		t.Parallel()
	assert.True(t, containsAll("胸痛伴呼吸困难", "胸痛 呼吸困难"))
	assert.True(t, containsAll("我胸痛并且呼吸困难", "胸痛 呼吸困难"))
	assert.False(t, containsAll("只有胸痛", "胸痛 呼吸困难"))
	assert.True(t, containsAll("Hello World", "hello world"))
	assert.True(t, containsAll("A B C", "A C"), "非连续包含")
	assert.False(t, containsAll("A B", "A C"), "缺少 C")
}
