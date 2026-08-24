package store

import (
	"testing"
	"time"

	"task211-quenchprop/internal/model"
)

func TestExperimentStorePersistsAcrossReopenAndRejectsStaleVersion(t *testing.T) {
	dbPath := t.TempDir() + "/experiments.db"

	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	store := NewExperimentStore(db)
	wantCreated := time.Date(2026, 8, 24, 1, 2, 3, 0, time.UTC)
	exp := &model.DischargeExperiment{
		ID:         "exp-persist",
		Name:       "重启恢复试验",
		MagnetSN:   "MAG-PERSIST",
		TopologyID: "topo-persist",
		Operator:   "engineer",
		SampleRate: 1000,
		Status:     model.ExpStatusPrepared,
		CreatedAt:  wantCreated,
	}
	if err := store.Create(exp); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store = NewExperimentStore(db)
	got, err := store.Get(exp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != exp.Name || got.Status != model.ExpStatusPrepared || got.Version != 1 {
		t.Fatalf("reopened experiment = %+v, want persisted prepared version 1", got)
	}
	if !got.CreatedAt.Equal(wantCreated) {
		t.Fatalf("created_at = %s, want %s", got.CreatedAt, wantCreated)
	}

	got.Status = model.ExpStatusCollecting
	if err := store.Update(got); err != nil {
		t.Fatal(err)
	}
	stale := *got
	stale.Version = 1
	stale.Status = model.ExpStatusPrepared
	if err := store.Update(&stale); err != model.ErrConflict {
		t.Fatalf("stale update error = %v, want %v", err, model.ErrConflict)
	}
}
