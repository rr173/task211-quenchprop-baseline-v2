package service

import (
	"testing"

	"task211-quenchprop/internal/model"
	"task211-quenchprop/internal/store"
)

func TestBug01_SealedExperimentRemainsImmutable(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/sealed.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	app := NewApp(db, DefaultConfig())
	if err := app.AddTopology(&model.CoilTopology{ID: "topo-sealed", Name: "topology"}); err != nil {
		t.Fatal(err)
	}
	exp, err := app.CreateExperiment(ExperimentInput{
		Name: "sealed", MagnetSN: "mag-1", Operator: "op", TopologyID: "topo-sealed", SampleRate: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, next := range []string{model.ExpStatusCollecting, model.ExpStatusAnalyzing, model.ExpStatusConfirmed, model.ExpStatusSealed} {
		if _, err := app.TransitionExperiment(exp.ID, next); err != nil {
			t.Fatalf("transition to %s: %v", next, err)
		}
	}
	sealed, err := app.GetExperiment(exp.ID)
	if err != nil {
		t.Fatal(err)
	}
	sealed.TopologyID = "topo-other"
	if err := app.UpdateExperimentTopology(sealed); err != model.ErrSealed {
		t.Fatalf("sealed topology update error = %v, want %v", err, model.ErrSealed)
	}
}
