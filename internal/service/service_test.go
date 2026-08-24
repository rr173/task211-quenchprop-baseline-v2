package service

import (
	"testing"

	"task211-quenchprop/internal/model"
	"task211-quenchprop/internal/store"
)

func TestSealedExperimentRejectsFurtherWrites(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/service.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	app := NewApp(db, DefaultConfig())
	if err := app.AddTopology(&model.CoilTopology{ID: "topo-sealed", Name: "sealed topology"}); err != nil {
		t.Fatal(err)
	}
	exp, err := app.CreateExperiment(ExperimentInput{
		Name:       "封存写入边界",
		MagnetSN:   "MAG-SEALED",
		Operator:   "engineer",
		TopologyID: "topo-sealed",
		SampleRate: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	channel := &model.Channel{Index: 1, SegmentID: "seg-1", Status: model.ChanOnline}
	if err := app.AddChannel(exp.ID, channel); err != nil {
		t.Fatal(err)
	}

	for _, next := range []string{
		model.ExpStatusCollecting,
		model.ExpStatusAnalyzing,
		model.ExpStatusConfirmed,
		model.ExpStatusSealed,
	} {
		if _, err := app.TransitionExperiment(exp.ID, next); err != nil {
			t.Fatalf("transition to %s: %v", next, err)
		}
	}
	if _, err := app.IngestWaveform(exp.ID, channel.ID, 1000, nil); err != model.ErrSealed {
		t.Fatalf("sealed ingest error = %v, want %v", err, model.ErrSealed)
	}
	if err := app.AddChannel(exp.ID, &model.Channel{Index: 2, SegmentID: "seg-2"}); err != model.ErrSealed {
		t.Fatalf("sealed channel error = %v, want %v", err, model.ErrSealed)
	}
}
