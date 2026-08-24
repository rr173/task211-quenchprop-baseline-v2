// Package ingest 负责波形的幂等摄入与基础校验。
package ingest

import (
	"task211-quenchprop/internal/model"
	"task211-quenchprop/internal/store"
)

// Result 波形摄入结果。
type Result struct {
	WaveformID string
	Duplicated bool // 是否命中既有指纹（幂等）
}

// monotonic 校验时间轴严格单调递增（允许极小容差）。
func monotonic(points []model.Sample) bool {
	for i := 1; i < len(points); i++ {
		if points[i].RawT <= points[i-1].RawT {
			return false
		}
	}
	return true
}

// Ingest 幂等摄入一条波形：先算指纹，命中既有记录则直接返回；
// 否则校验时间轴单调后落库。rateMismatch 用于采样率一致性（与试验基准比对）。
func Ingest(s *store.WaveformStore, w *model.Waveform, expectedRate float64) (*Result, error) {
	if len(w.Points) == 0 {
		return nil, model.ErrInsufficientData
	}
	if !monotonic(w.Points) {
		return nil, model.ErrTimeAxisBroken
	}
	if expectedRate > 0 && w.SampleRate > 0 && abs(w.SampleRate-expectedRate) > 1e-6 {
		return nil, model.ErrRateMismatch
	}
	fp := model.FingerprintOf(w.ExperimentID, w.ChannelID, w.SampleRate, w.Points)
	w.Fingerprint = fp
	exists, err := s.ExistsByFingerprint(fp)
	if err != nil {
		return nil, err
	}
	if exists {
		existing, err := s.GetByFingerprint(fp)
		if err != nil {
			return nil, err
		}
		return &Result{WaveformID: existing.ID, Duplicated: true}, nil
	}
	w.ID = model.NewID("wf")
	if err := s.Save(w); err != nil {
		return nil, err
	}
	return &Result{WaveformID: w.ID, Duplicated: false}, nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
