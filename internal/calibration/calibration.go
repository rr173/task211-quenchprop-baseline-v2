// Package calibration 实现多通道波形延迟校正（最陡边对齐）。
package calibration

import (
	"fmt"

	"task211-quenchprop/internal/model"
)

// EdgeInfo 描述检测到的最陡上升边。
type EdgeInfo struct {
	Time  float64 // 边时间（秒，应用当前延迟后的校正时间）
	Idx   int
	Slope float64
	Value float64
}

// SteepestRisingEdge 在波形（应用当前延迟校正）上寻找最陡上升边。
func SteepestRisingEdge(w *model.Waveform) EdgeInfo {
	pts := w.CorrectedPoints()
	best := EdgeInfo{Slope: 0}
	for i := 1; i < len(pts); i++ {
		dt := pts[i].RawT - pts[i-1].RawT
		if dt <= 0 {
			continue
		}
		slope := (pts[i].V - pts[i-1].V) / dt
		if slope > best.Slope {
			best = EdgeInfo{Time: pts[i].RawT, Idx: i, Slope: slope, Value: pts[i].V}
		}
	}
	return best
}

// CalibrateDelays 以 refChannelID 为基准，估计各通道相对基准的采集延迟，
// 并就地更新各波形的 DelayNs（把最陡边对齐）。返回 channelID -> 新延迟（纳秒）。
func CalibrateDelays(waveforms []*model.Waveform, refChannelID string) (map[string]float64, error) {
	edges := map[string]EdgeInfo{}
	for _, w := range waveforms {
		edge := SteepestRisingEdge(w)
		// 有效上升沿要求斜率严格为正：全平波形（Slope==0）与仅有下降沿的波形
		// 均缺少可用边，必须显式拒绝，避免用零斜率对应的「幽灵边」生成看似有效
		// 实则无意义的延迟并继续写回校准结果。
		if edge.Slope <= 0 {
			return nil, model.ErrInsufficientData
		}
		edges[w.ChannelID] = edge
	}
	ref, ok := edges[refChannelID]
	if !ok {
		return nil, fmt.Errorf("reference channel %s not found among waveforms", refChannelID)
	}
	out := map[string]float64{}
	for _, w := range waveforms {
		e := edges[w.ChannelID]
		offsetNs := (e.Time - ref.Time) * 1e9
		newDelay := w.DelayNs + offsetNs
		w.DelayNs = newDelay
		w.Calibrated = true
		out[w.ChannelID] = newDelay
	}
	return out, nil
}
