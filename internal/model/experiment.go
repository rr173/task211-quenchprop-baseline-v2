package model

import "time"

// 放电试验状态机：准备 → 采集 → 分析中 → 已确认 → 封存（终态）。
const (
	ExpStatusPrepared   = "prepared"   // 准备
	ExpStatusCollecting = "collecting" // 采集
	ExpStatusAnalyzing  = "analyzing"  // 分析中
	ExpStatusConfirmed  = "confirmed"  // 已确认
	ExpStatusSealed     = "sealed"     // 封存（终态，拒写）
)

var expTransitions = map[string][]string{
	ExpStatusPrepared:   {ExpStatusCollecting},
	ExpStatusCollecting: {ExpStatusAnalyzing, ExpStatusPrepared},
	ExpStatusAnalyzing:  {ExpStatusConfirmed, ExpStatusCollecting},
	ExpStatusConfirmed:  {ExpStatusSealed},
	ExpStatusSealed:     {},
}

// DischargeExperiment 表示一次低温超导磁体放电（淬灭）试验。
type DischargeExperiment struct {
	ID         string
	Name       string
	MagnetSN   string
	TopologyID string
	Operator   string
	SampleRate float64 // Hz
	Status     string
	CreatedAt  time.Time
	SealedAt   *time.Time
	Version    int64
}

// CanTransition 判断能否迁移到目标状态。
func (e *DischargeExperiment) CanTransition(to string) bool {
	for _, s := range expTransitions[e.Status] {
		if s == to {
			return true
		}
	}
	return false
}

// Transition 执行状态迁移；封存后不可再迁移。
func (e *DischargeExperiment) Transition(to string) error {
	if e.Status == ExpStatusSealed {
		return ErrSealed
	}
	if !e.CanTransition(to) {
		return ErrInvalidState
	}
	e.Status = to
	if to == ExpStatusSealed {
		now := time.Now()
		e.SealedAt = &now
	}
	return nil
}

// IsSealed 试验是否已封存。
func (e *DischargeExperiment) IsSealed() bool { return e.Status == ExpStatusSealed }

// IsWritable 封存后拒写。
func (e *DischargeExperiment) IsWritable() bool { return e.Status != ExpStatusSealed }
