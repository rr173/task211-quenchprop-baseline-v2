package store

import (
	"testing"

	"task211-quenchprop/internal/model"
)

func TestBug09_PersistedProtectionBreachKeepsItsMeaning(t *testing.T) {
	db, err := Open(t.TempDir() + "/analysis.db")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	s := NewAnalysisStore(db)
	want := &model.AnalysisPackage{ID: "analysis-breach", ExperimentID: "exp", Status: model.AnaDraft, OnsetChannelID: "ch", Breach: true, AffectedSegments: []string{"s1"}}
	if err := s.Create(want); err != nil { t.Fatal(err) }
	got, err := s.Get(want.ID)
	if err != nil { t.Fatal(err) }
	if !got.Breach {
		t.Fatalf("persisted breach = %v, want true", got.Breach)
	}
}
