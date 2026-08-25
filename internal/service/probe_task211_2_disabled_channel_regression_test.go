package service

import (
	"testing"

	"task211-quenchprop/internal/model"
	"task211-quenchprop/internal/store"
)

func TestBug02_DisabledChannelIsExcludedFromAnalysis(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/disabled.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := NewApp(db, DefaultConfig())
	if err := app.AddTopology(&model.CoilTopology{ID: "topo-disabled", Segments: []model.TopoSegment{{ID: "seg-1"}}}); err != nil {
		t.Fatal(err)
	}
	exp, err := app.CreateExperiment(ExperimentInput{Name: "disabled", MagnetSN: "mag", Operator: "op", TopologyID: "topo-disabled", SampleRate: 1000})
	if err != nil {
		t.Fatal(err)
	}
	ch := &model.Channel{Index: 1, SegmentID: "seg-1", Status: model.ChanDisabled}
	if err := app.AddChannel(exp.ID, ch); err != nil {
		t.Fatal(err)
	}
	if _, err := app.IngestWaveform(exp.ID, ch.ID, 1000, []model.Sample{{RawT: 0, V: 0}, {RawT: 0.001, V: 1}}); err != model.ErrUnknownChannel {
		t.Fatalf("disabled waveform ingest error = %v, want %v", err, model.ErrUnknownChannel)
	}
}
