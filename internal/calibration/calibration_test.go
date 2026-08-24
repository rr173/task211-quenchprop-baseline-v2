package calibration

import (
	"testing"

	"task211-quenchprop/internal/model"
)

func spikeWaveform(channelID string, delayNs float64) *model.Waveform {
	n := 1000
	pts := make([]model.Sample, n)
	spikeIdx := 1 // 1ms 处校准脉冲
	for i := 0; i < n; i++ {
		trueT := float64(i) / 1000
		rawT := trueT + delayNs/1e9
		v := 0.0
		if i == spikeIdx {
			v = 0.2
		}
		pts[i] = model.Sample{RawT: rawT, V: v}
	}
	return &model.Waveform{ChannelID: channelID, Points: pts}
}

func TestCalibrateDelays(t *testing.T) {
	ref := spikeWaveform("ref", 0)
	b := spikeWaveform("b", 5e6)
	c := spikeWaveform("c", 10e6)
	delays, err := CalibrateDelays([]*model.Waveform{ref, b, c}, "ref")
	if err != nil {
		t.Fatal(err)
	}
	if delays["ref"] != 0 {
		t.Fatalf("ref delay = %f, want 0", delays["ref"])
	}
	if delays["b"] < 4.9e6 || delays["b"] > 5.1e6 {
		t.Fatalf("b delay = %f, want ~5e6", delays["b"])
	}
	if delays["c"] < 9.9e6 || delays["c"] > 10.1e6 {
		t.Fatalf("c delay = %f, want ~10e6", delays["c"])
	}
}

func TestCalibrateMissingReference(t *testing.T) {
	if _, err := CalibrateDelays([]*model.Waveform{spikeWaveform("b", 0)}, "nope"); err == nil {
		t.Fatal("expected error for missing reference channel")
	}
}
