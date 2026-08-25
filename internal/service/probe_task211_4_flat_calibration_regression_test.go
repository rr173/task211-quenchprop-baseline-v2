package service

import (
	"testing"

	"task211-quenchprop/internal/model"
	"task211-quenchprop/internal/store"
)

func TestBug04_FlatWaveformCannotBeCalibrated(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/calibration.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := NewApp(db, DefaultConfig())
	if err := app.AddTopology(&model.CoilTopology{ID: "topo-flat"}); err != nil {
		t.Fatal(err)
	}
	exp, err := app.CreateExperiment(ExperimentInput{Name: "flat", MagnetSN: "mag", Operator: "op", TopologyID: "topo-flat", SampleRate: 1000})
	if err != nil {
		t.Fatal(err)
	}
	a := &model.Channel{Index: 1, SegmentID: "s1", Status: model.ChanOnline}
	b := &model.Channel{Index: 2, SegmentID: "s2", Status: model.ChanOnline}
	if err := app.AddChannel(exp.ID, a); err != nil { t.Fatal(err) }
	if err := app.AddChannel(exp.ID, b); err != nil { t.Fatal(err) }
	flat := []model.Sample{{RawT: 0, V: 0}, {RawT: 0.001, V: 0}, {RawT: 0.002, V: 0}}
	if _, err := app.IngestWaveform(exp.ID, a.ID, 1000, flat); err != nil { t.Fatal(err) }
	if _, err := app.IngestWaveform(exp.ID, b.ID, 1000, flat); err != nil { t.Fatal(err) }
	if _, err := app.Calibrate(exp.ID, a.ID); err != model.ErrInsufficientData {
		t.Fatalf("flat calibration error = %v, want %v", err, model.ErrInsufficientData)
	}
}
