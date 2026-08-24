package service

import (
	"testing"

	"task211-quenchprop/internal/model"
	"task211-quenchprop/internal/store"
)

func TestBug03_WaveformIdentityIncludesChannel(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/fingerprint.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := NewApp(db, DefaultConfig())
	if err := app.AddTopology(&model.CoilTopology{ID: "topo-fp"}); err != nil {
		t.Fatal(err)
	}
	exp, err := app.CreateExperiment(ExperimentInput{Name: "fp", MagnetSN: "mag", Operator: "op", TopologyID: "topo-fp", SampleRate: 1000})
	if err != nil {
		t.Fatal(err)
	}
	first := &model.Channel{Index: 1, SegmentID: "s1", Status: model.ChanOnline}
	second := &model.Channel{Index: 2, SegmentID: "s2", Status: model.ChanOnline}
	if err := app.AddChannel(exp.ID, first); err != nil {
		t.Fatal(err)
	}
	if err := app.AddChannel(exp.ID, second); err != nil {
		t.Fatal(err)
	}
	points := []model.Sample{{RawT: 0, V: 0}, {RawT: 0.001, V: 1}}
	a, err := app.IngestWaveform(exp.ID, first.ID, 1000, points)
	if err != nil || a.Duplicated {
		t.Fatalf("first waveform = %+v, err=%v", a, err)
	}
	b, err := app.IngestWaveform(exp.ID, second.ID, 1000, points)
	if err != nil {
		t.Fatal(err)
	}
	if b.Duplicated || b.WaveformID == a.WaveformID {
		t.Fatalf("second channel incorrectly deduplicated: first=%+v second=%+v", a, b)
	}
}
