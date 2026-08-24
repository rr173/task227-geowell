// Package compare implements cross-run comparison: it aligns gradient
// boundaries from a baseline run and a target run, computes how far each
// boundary moved, and flags movements beyond a tolerance as anomalies.
package compare

import (
	"encoding/json"
	"math"

	"task227-geowell/internal/model"
)

// Tolerance controls anomaly flagging for boundary movement.
type Tolerance struct {
	DepthDeltaMax float64 // m; movement beyond this is anomalous
}

// DefaultTolerance returns a conservative default.
func DefaultTolerance() Tolerance { return Tolerance{DepthDeltaMax: 25.0} }

// Boundary is a single gradient boundary depth extracted from a run.
type Boundary struct {
	Depth float64
	Jump  float64
}

// BoundariesFromSegments extracts the top depth of each segment as a boundary.
func BoundariesFromSegments(segs []model.Segment) []Boundary {
	out := make([]Boundary, 0, len(segs))
	for _, s := range segs {
		out = append(out, Boundary{Depth: s.TopDepth, Jump: s.GradientJump})
	}
	return out
}

// Movement describes one matched boundary pair.
type Movement struct {
	DepthBaseline float64 `json:"depth_baseline"`
	DepthTarget   float64 `json:"depth_target"`
	DeltaDepth    float64 `json:"delta_depth"`
	Anomaly       bool    `json:"anomaly"`
	Note          string  `json:"note"`
}

// Compare matches baseline boundaries to the nearest target boundary and
// reports the movement. A baseline boundary with no close target counterpart
// is reported as a vanished boundary.
func Compare(baseline, target []Boundary, tol Tolerance) []Movement {
	var moves []Movement
	used := make([]bool, len(target))
	for _, b := range baseline {
		bestIdx := -1
		bestD := math.MaxFloat64
		for i, t := range target {
			if used[i] {
				continue
			}
			if d := math.Abs(t.Depth - b.Depth); d < bestD {
				bestD = d
				bestIdx = i
			}
		}
		if bestIdx < 0 || bestD > tol.DepthDeltaMax*4 {
			moves = append(moves, Movement{
				DepthBaseline: b.Depth,
				DepthTarget:   math.NaN(),
				DeltaDepth:    math.NaN(),
				Anomaly:       true,
				Note:          "baseline boundary vanished in target run",
			})
			continue
		}
		used[bestIdx] = true
		t := target[bestIdx]
		delta := t.Depth - b.Depth
		mv := Movement{
			DepthBaseline: b.Depth,
			DepthTarget:   t.Depth,
			DeltaDepth:    delta,
			Anomaly:       math.Abs(delta) > tol.DepthDeltaMax,
			Note:          "",
		}
		if mv.Anomaly {
			mv.Note = "boundary moved beyond tolerance"
		}
		moves = append(moves, mv)
	}
	return moves
}

// BuildDetail serializes movements to the snapshot detail JSON field.
func BuildDetail(moves []Movement) (string, error) {
	b, err := json.Marshal(moves)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ParseDetail reads movements back from a snapshot detail JSON field.
func ParseDetail(detail string) ([]Movement, error) {
	if detail == "" || detail == "[]" {
		return []Movement{}, nil
	}
	var moves []Movement
	if err := json.Unmarshal([]byte(detail), &moves); err != nil {
		return nil, err
	}
	return moves, nil
}

// Summary returns counts of total/anomalous movements.
func Summary(moves []Movement) (total, anomalous int) {
	for _, m := range moves {
		total++
		if m.Anomaly {
			anomalous++
		}
	}
	return total, anomalous
}
