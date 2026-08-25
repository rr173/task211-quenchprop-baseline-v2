package propagation

import (
	"testing"

	"task211-quenchprop/internal/model"
)

func TestBug08_PropagationFollowsReverseTopologyEdges(t *testing.T) {
	topo := &model.CoilTopology{ID: "reverse", Edges: []model.TopoEdge{{ID: "e", FromID: "s2", ToID: "s1", Distance: 1}}}
	channels := []*model.Channel{{ID: "origin", SegmentID: "s1", Status: model.ChanOnline}, {ID: "next", SegmentID: "s2", Status: model.ChanOnline}}
	res := Propagate(map[string]float64{"origin": 0.1, "next": 0.3}, channels, topo)
	if res.PropagationSpeed <= 0 {
		t.Fatalf("reverse-edge propagation speed = %v, want positive", res.PropagationSpeed)
	}
}
