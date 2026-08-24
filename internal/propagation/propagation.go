// Package propagation 实现淬灭起点的检测与沿线圈拓扑的传播分析。
package propagation

import (
	"sort"

	"task211-quenchprop/internal/model"
)

// OnsetResult 单通道的淬灭起点检测结果。
type OnsetResult struct {
	ChannelID  string
	Time       float64 // 校正后的起点时间（秒）
	Idx        int
	Confidence float64
	Slope      float64
}

// DetectOnset 在波形（应用当前延迟校正后）上检测淬灭起点：
// 返回「最早」量测值穿越阈值且上升斜率超过最小斜率的采样点，置信度随斜率增大。
func DetectOnset(w *model.Waveform, threshold, minSlope float64) (OnsetResult, bool) {
	pts := w.CorrectedPoints()
	for i := 1; i < len(pts); i++ {
		if pts[i].V < threshold {
			continue
		}
		dt := pts[i].RawT - pts[i-1].RawT
		if dt <= 0 {
			continue
		}
		slope := (pts[i].V - pts[i-1].V) / dt
		if slope < minSlope {
			continue
		}
		conf := model.Clamp(slope/(minSlope*5), 0, 1)
		return OnsetResult{ChannelID: w.ChannelID, Time: pts[i].RawT, Idx: i, Confidence: conf, Slope: slope}, true
	}
	return OnsetResult{}, false
}

// SegmentSpeed 相邻分段间的传播速度估计。
type SegmentSpeed struct {
	FromSeg   string
	ToSeg     string
	Distance  float64
	TimeDelta float64
	Speed     float64 // 米/秒
}

// PropagationResult 沿拓扑的传播分析结果。
type PropagationResult struct {
	OnsetChannelID   string
	OnsetTime        float64
	ArrivalTimes     map[string]float64 // 通道 -> 到达（起点）时间（秒）
	AffectedSegments []string
	Speeds           []SegmentSpeed
	PropagationSpeed float64 // 米/秒（取各段速度中位数）
}

// arrivalForSegment 返回某拓扑分段内最早检出起点的通道时间；无则返回 -1。
func arrivalForSegment(segID string, chanSeg map[string]string, onsets map[string]float64) float64 {
	best := -1.0
	for ch, seg := range chanSeg {
		if seg != segID {
			continue
		}
		t, ok := onsets[ch]
		if !ok {
			continue
		}
		if best < 0 || t < best {
			best = t
		}
	}
	return best
}

// Propagate 依据各通道起点时间、通道-分段映射与拓扑，计算传播分析。
// onsets：channelID -> 校正起点时间；channels：参与分析的通道；
// topo：线圈拓扑。结果含起点通道、受影响分段、各段传播速度与整体速度。
func Propagate(onsets map[string]float64, channels []*model.Channel, topo *model.CoilTopology) PropagationResult {
	res := PropagationResult{ArrivalTimes: map[string]float64{}}
	chanSeg := make(map[string]string, len(channels))
	for _, c := range channels {
		chanSeg[c.ID] = c.SegmentID
		for ch, t := range onsets {
			if ch == c.ID {
				res.ArrivalTimes[ch] = t
			}
		}
	}
	// 起点通道：最早起点
	onsetCh := ""
	onsetT := 0.0
	for ch, t := range onsets {
		if onsetCh == "" || t < onsetT {
			onsetCh = ch
			onsetT = t
		}
	}
	res.OnsetChannelID = onsetCh
	res.OnsetTime = onsetT

	segSet := map[string]bool{}
	for ch := range onsets {
		if seg, ok := chanSeg[ch]; ok {
			segSet[seg] = true
		}
	}
	for seg := range segSet {
		res.AffectedSegments = append(res.AffectedSegments, seg)
	}
	sort.Strings(res.AffectedSegments)

	// 从起点分段 BFS，沿拓扑计算相邻分段到达时间差 -> 速度
	onsetSeg := chanSeg[onsetCh]
	speeds := []SegmentSpeed{}
	visited := map[string]bool{onsetSeg: true}
	queue := []string{onsetSeg}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		curArr := arrivalForSegment(cur, chanSeg, onsets)
		if curArr < 0 {
			continue
		}
		for _, e := range topo.Neighbors(cur) {
			toArr := arrivalForSegment(e.ToID, chanSeg, onsets)
			if toArr < 0 {
				continue
			}
			dt := toArr - curArr
			if dt > 0 {
				speeds = append(speeds, SegmentSpeed{
					FromSeg: cur, ToSeg: e.ToID, Distance: e.Distance, TimeDelta: dt,
					Speed: e.Distance / dt,
				})
			}
			if !visited[e.ToID] {
				visited[e.ToID] = true
				queue = append(queue, e.ToID)
			}
		}
	}
	if len(speeds) > 0 {
		vals := make([]float64, len(speeds))
		for i, s := range speeds {
			vals[i] = s.Speed
		}
		res.PropagationSpeed = model.Median(vals)
		res.Speeds = speeds
	}
	return res
}
