package entity

// OnboardingStatus 记录安装向导的完成状态，供前后端交换状态信息。
type OnboardingStatus struct {
	Completed   bool `json:"completed"`    // 向导是否已完成
	CurrentStep int  `json:"current_step"` // 当前所在步骤（1-3）
	Skipped     bool `json:"skipped"`      // 用户是否主动跳过向导
}
