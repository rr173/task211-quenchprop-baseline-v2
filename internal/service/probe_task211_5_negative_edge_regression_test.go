package service

import (
	"testing"

	"task211-quenchprop/internal/model"
	"task211-quenchprop/internal/store"
)

func TestBug05_NegativeTopologyDistanceRejected(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/topology.db")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	app := NewApp(db, DefaultConfig())
	err = app.AddTopology(&model.CoilTopology{ID: "topo-negative", Segments: []model.TopoSegment{{ID: "s1"}, {ID: "s2"}}, Edges: []model.TopoEdge{{ID: "e", FromID: "s1", ToID: "s2", Distance: -1}}})
	if err != model.ErrInvalidArgument {
		t.Fatalf("negative edge error = %v, want %v", err, model.ErrInvalidArgument)
	}
}
