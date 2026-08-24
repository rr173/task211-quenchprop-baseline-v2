package service

import (
	"testing"

	"task211-quenchprop/internal/model"
	"task211-quenchprop/internal/store"
)

func TestBug06_AnalysisFailureLeavesExperimentCollecting(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/rollback.db")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	app := NewApp(db, DefaultConfig())
	if err := app.AddTopology(&model.CoilTopology{ID: "topo-rollback"}); err != nil { t.Fatal(err) }
	exp, err := app.CreateExperiment(ExperimentInput{Name: "rollback", MagnetSN: "mag", Operator: "op", TopologyID: "topo-rollback", SampleRate: 1000})
	if err != nil { t.Fatal(err) }
	ch := &model.Channel{Index: 1, SegmentID: "s1", Status: model.ChanOnline}
	if err := app.AddChannel(exp.ID, ch); err != nil { t.Fatal(err) }
	if _, err := app.IngestWaveform(exp.ID, ch.ID, 1000, []model.Sample{{RawT: 0, V: 0}, {RawT: 0.001, V: 0}}); err != nil { t.Fatal(err) }
	if _, err := app.Analyze(exp.ID, AnalyzeOptions{}); err == nil { t.Fatal("expected analysis failure for waveform without onset") }
	got, err := app.GetExperiment(exp.ID)
	if err != nil { t.Fatal(err) }
	if got.Status != model.ExpStatusCollecting {
		t.Fatalf("status after failed analysis = %s, want %s", got.Status, model.ExpStatusCollecting)
	}
}
