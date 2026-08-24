package propagation

import (
	"testing"

	"task211-quenchprop/internal/model"
)

func ramp(onsetT float64, n int, rate float64) []model.Sample {
	pts := make([]model.Sample, n)
	for i := 0; i < n; i++ {
		t := float64(i) / rate
		v := 0.0
		if t >= onsetT {
			v = (t - onsetT) * 50
			if v > 5 {
				v = 5
			}
		}
		pts[i] = model.Sample{RawT: t, V: v}
	}
	return pts
}

func TestDetectOnset(t *testing.T) {
	w := &model.Waveform{ChannelID: "ch1", Points: ramp(0.10, 1000, 1000)}
	on, ok := DetectOnset(w, 0.5, 0.05)
	if !ok {
		t.Fatal("expected onset detected")
	}
	if on.Time < 0.10 || on.Time > 0.12 {
		t.Fatalf("onset time = %f, want ~0.11", on.Time)
	}
	if on.Confidence <= 0 {
		t.Fatalf("confidence should be positive, got %f", on.Confidence)
	}
}

func TestDetectOnsetBelowThreshold(t *testing.T) {
	w := &model.Waveform{ChannelID: "ch1", Points: ramp(0.10, 1000, 1000)}
	if _, ok := DetectOnset(w, 100.0, 0.05); ok {
		t.Fatal("expected no onset when threshold too high")
	}
}

func TestPropagate(t *testing.T) {
	topo := &model.CoilTopology{
		ID: "t",
		Edges: []model.TopoEdge{
			{ID: "e1", FromID: "s1", ToID: "s2", Distance: 1.0},
			{ID: "e2", FromID: "s2", ToID: "s3", Distance: 2.0},
		},
	}
	channels := []*model.Channel{
		{ID: "a", SegmentID: "s1"},
		{ID: "b", SegmentID: "s2"},
		{ID: "c", SegmentID: "s3"},
	}
	onsets := map[string]float64{"a": 0.1, "b": 0.3, "c": 0.9} // s1→s2: 5 m/s；s2→s3: 3.33 m/s
	res := Propagate(onsets, channels, topo)
	if res.OnsetChannelID != "a" {
		t.Fatalf("onset channel = %s, want a", res.OnsetChannelID)
	}
	if len(res.AffectedSegments) != 3 {
		t.Fatalf("affected segments = %d, want 3", len(res.AffectedSegments))
	}
	if res.PropagationSpeed <= 0 {
		t.Fatalf("propagation speed should be positive")
	}
}
