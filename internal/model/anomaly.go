package model

// 异常段状态机：候选 → 起点 / 传播中 / 误报 → 确认。
const (
	AnomCandidate     = "candidate"      // 候选
	AnomOnset         = "onset"          // 起点
	AnomPropagating   = "propagating"    // 传播中
	AnomFalsePositive = "false_positive" // 误报
	AnomConfirmed     = "confirmed"      // 确认
)

var anomTransitions = map[string][]string{
	AnomCandidate:     {AnomOnset, AnomPropagating, AnomFalsePositive},
	AnomOnset:         {AnomConfirmed, AnomPropagating, AnomFalsePositive},
	AnomPropagating:   {AnomConfirmed, AnomFalsePositive},
	AnomFalsePositive: {},
	AnomConfirmed:     {},
}

// AnomalySegment 表示在某个通道上检出的异常（淬灭）区段。
type AnomalySegment struct {
	ID           string
	ExperimentID string
	ChannelID    string
	SampleStart  int
	SampleEnd    int
	DetectedAt   float64 // 校正时间（秒）
	Type         string
	Confidence   float64
	Severity     float64
	Version      int64
}

// CanTransition 判断异常段能否迁移到目标类型。
func (a *AnomalySegment) CanTransition(to string) bool {
	for _, s := range anomTransitions[a.Type] {
		if s == to {
			return true
		}
	}
	return false
}

// Transition 执行异常段类型迁移。
func (a *AnomalySegment) Transition(to string) error {
	if !a.CanTransition(to) {
		return ErrInvalidState
	}
	a.Type = to
	return nil
}
