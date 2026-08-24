package protection

import (
	"testing"

	"task211-quenchprop/internal/model"
)

func TestComputeWindow(t *testing.T) {
	topo := &model.CoilTopology{
		ID: "t",
		Edges: []model.TopoEdge{
			{ID: "e1", FromID: "s1", ToID: "s2", Distance: 1.0},
			{ID: "e2", FromID: "s2", ToID: "s3", Distance: 2.0},
		},
	}
	// s1→s3 距离 3 米，速度 5 m/s => 0.6s = 600ms，加 50ms 裕度 => 650ms。
	ms, breach := ComputeWindow("s1", "s3", 5.0, topo, 50, 500)
	if ms != 650 {
		t.Fatalf("window = %f ms, want 650", ms)
	}
	if !breach {
		t.Fatal("expected breach (650 > 500)")
	}
}

func TestComputeWindowUnreachable(t *testing.T) {
	topo := &model.CoilTopology{ID: "t"}
	_, breach := ComputeWindow("s1", "sX", 5.0, topo, 50, 500)
	if !breach {
		t.Fatal("expected breach when critical segment unreachable")
	}
}
