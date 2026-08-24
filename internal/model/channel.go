package model

// 通道状态。
const (
	ChanOnline   = "online"   // 在线（正常可用）
	ChanBad      = "bad"      // 坏道（仍可贡献证据，仅降置信）
	ChanDisabled = "disabled" // 停用（不参与分析）
)

// Channel 表示磁体上一个测量通道（传感器），挂载于某拓扑分段。
type Channel struct {
	ID           string
	ExperimentID string
	Index        int
	Label        string
	Kind         string  // temperature | voltage | resistance
	SegmentID    string  // 所属拓扑分段 ID
	Status       string
	Position     float64 // 沿磁体位置（米）
	DelayNs      float64 // 采集延迟（纳秒），用于延迟校准
	Version      int64
}

// IsUsable 通道是否参与分析（在线）。
func (c *Channel) IsUsable() bool { return c.Status == ChanOnline }

// IsFaulty 是否为坏道：坏道只降置信，不丢证据。
func (c *Channel) IsFaulty() bool { return c.Status == ChanBad }

// IsParticipating 是否参与传播分析（在线或坏道均参与，停用不参与）。
func (c *Channel) IsParticipating() bool {
	return c.Status == ChanOnline || c.Status == ChanBad
}
