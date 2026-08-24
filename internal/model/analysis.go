package model

import "time"

// 分析包状态机：草稿 → 复核 → 发布 → 替代。
const (
	AnaDraft      = "draft"      // 草稿
	AnaReview     = "review"     // 复核
	AnaPublished  = "published"  // 发布
	AnaSuperseded = "superseded" // 替代（被新分析包取代）
)

var anaTransitions = map[string][]string{
	AnaDraft:     {AnaReview},
	AnaReview:    {AnaPublished, AnaDraft},
	AnaPublished: {AnaSuperseded},
	AnaSuperseded: {},
}

// AnalysisPackage 为一次完整分析产出的发布包。
type AnalysisPackage struct {
	ID                string
	ExperimentID      string
	Status            string
	OnsetChannelID    string
	OnsetTime         float64 // 淬灭起点校正时间（秒）
	PropagationSpeed  float64 // 传播速度（米/秒）
	AffectedSegments  []string
	ProtectionWindowMs float64 // 保护窗口（毫秒）
	Breach            bool  // 是否超出保护能力
	Summary           string
	CreatedAt         time.Time
	SupersededBy      string
	Version           int64
}

// CanTransition 判断分析包能否迁移到目标状态。
func (a *AnalysisPackage) CanTransition(to string) bool {
	for _, s := range anaTransitions[a.Status] {
		if s == to {
			return true
		}
	}
	return false
}

// Transition 执行分析包状态迁移。
func (a *AnalysisPackage) Transition(to string) error {
	if !a.CanTransition(to) {
		return ErrInvalidState
	}
	a.Status = to
	return nil
}
