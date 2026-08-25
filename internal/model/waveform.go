package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Sample 为单通道的一个采样点。RawT 为原始记录时间，校准时通过 DelayNs 校正。
type Sample struct {
	RawT float64 // 原始记录时间（秒，相对试验起点）
	V    float64 // 量测值
}

// Corrected 返回应用某采集延迟后的校正时间（秒）。
func (s Sample) Corrected(delayNs float64) float64 { return s.RawT - delayNs/1e9 }

// Waveform 为单通道的时序波形。
type Waveform struct {
	ID           string
	ExperimentID string
	ChannelID    string
	SampleRate   float64 // Hz
	Fingerprint  string  // 幂等指纹
	Calibrated   bool
	DelayNs      float64 // 实际采用的校准延迟
	Points       []Sample
	CreatedAt    time.Time
	Version      int64
}

// FingerprintOf 依据（试验、通道、采样率、样本数、首尾值）计算幂等指纹。
// 用于波形层面的幂等：相同来源的重复上传直接命中既有记录。
func FingerprintOf(expID, chanID string, rate float64, points []Sample) string {
	first := 0.0
	last := 0.0
	if len(points) > 0 {
		first = points[0].V
		last = points[len(points)-1].V
	}
	raw := fmt.Sprintf("%s|%s|%.6f|%d|%.9f|%.9f", expID, chanID, rate, len(points), first, last)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// CorrectedPoints 返回应用当前 DelayNs 后的校正采样点。
func (w *Waveform) CorrectedPoints() []Sample {
	out := make([]Sample, len(w.Points))
	for i, p := range w.Points {
		cp := p
		cp.RawT = p.Corrected(w.DelayNs)
		out[i] = cp
	}
	return out
}
