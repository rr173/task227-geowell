package compare

import (
	"math"
	"testing"
)

func TestCompareFlagsBoundaryMovementBeyondTolerance(t *testing.T) {
	moves := Compare(
		[]Boundary{{Depth: 100, Jump: 0.2}},
		[]Boundary{{Depth: 140, Jump: 0.2}},
		Tolerance{DepthDeltaMax: 25},
	)
	if len(moves) != 1 || !moves[0].Anomaly || moves[0].DeltaDepth != 40 {
		t.Fatalf("Compare() = %+v, want one anomalous +40m movement", moves)
	}
}

func TestCompareReportsVanishedBoundary(t *testing.T) {
	moves := Compare([]Boundary{{Depth: 100}}, nil, DefaultTolerance())
	if len(moves) != 1 || !moves[0].Anomaly || !math.IsNaN(moves[0].DepthTarget) {
		t.Fatalf("vanished boundary = %+v, want anomalous NaN target", moves)
	}
}
