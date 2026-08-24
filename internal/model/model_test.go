package model

import "testing"

func TestMedian(t *testing.T) {
	if Median([]float64{3, 1, 2}) != 2 {
		t.Fatal("median of {3,1,2} should be 2")
	}
	if Median([]float64{1, 2, 3, 4}) != 2.5 {
		t.Fatal("median of {1,2,3,4} should be 2.5")
	}
	if Median(nil) != 0 {
		t.Fatal("median of empty should be 0")
	}
}

func TestClamp(t *testing.T) {
	if Clamp(5, 0, 1) != 1 {
		t.Fatal("clamp upper bound failed")
	}
	if Clamp(-1, 0, 1) != 0 {
		t.Fatal("clamp lower bound failed")
	}
	if Clamp(0.5, 0, 1) != 0.5 {
		t.Fatal("clamp in-range failed")
	}
}

func TestPathDistance(t *testing.T) {
	topo := &CoilTopology{
		ID: "t",
		Edges: []TopoEdge{
			{ID: "e1", FromID: "s1", ToID: "s2", Distance: 1.0},
			{ID: "e2", FromID: "s2", ToID: "s3", Distance: 2.0},
		},
	}
	if d := topo.PathDistance("s1", "s3"); d != 3.0 {
		t.Fatalf("distance = %f, want 3.0", d)
	}
	if d := topo.PathDistance("s3", "s1"); d != 3.0 {
		t.Fatalf("reverse distance = %f, want 3.0", d)
	}
	if d := topo.PathDistance("s1", "sX"); d != -1 {
		t.Fatalf("unreachable distance = %f, want -1", d)
	}
}
