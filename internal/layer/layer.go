// Package layer implements the core layering algorithm: it resamples the
// corrected profile onto a uniform depth grid, computes temperature and
// pressure gradients per interval, detects gradient jumps (boundary candidates)
// and classifies each segment as stable or anomalous.
package layer

import (
	"math"

	"task227-geowell/internal/model"
)

// Params controls segmentation and boundary sensitivity.
type Params struct {
	GridStep      float64 // uniform resample step (m)
	JumpThreshold float64 // |Δtemp gradient| above this marks a candidate boundary
	MinSegThick   float64 // minimum segment thickness (m) to keep
}

// DefaultParams returns a conservative default configuration.
func DefaultParams() Params {
	return Params{GridStep: 5.0, JumpThreshold: 0.05, MinSegThick: 10.0}
}

// GridPoint is a resampled, uniformly spaced profile sample.
type GridPoint struct {
	Depth float64
	Temp  float64
	Press float64
}

// Resample projects the (possibly irregular) corrected points onto a uniform
// depth grid via linear interpolation. Gaps are skipped; the surrounding valid
// samples bracket the gap so missing sections are not invented across large
// jumps.
func Resample(points []model.MeasurePoint, step float64) []GridPoint {
	valid := make([]model.MeasurePoint, 0, len(points))
	for _, p := range points {
		if !p.Gap && p.State != model.PointDepthConflict {
			valid = append(valid, p)
		}
	}
	if len(valid) < 2 {
		return nil
	}
	top := valid[0].DepthCal
	bottom := valid[len(valid)-1].DepthCal
	var out []GridPoint
	for d := top; d <= bottom+1e-9; d += step {
		t, p, ok := interp(valid, d)
		if !ok {
			continue
		}
		out = append(out, GridPoint{Depth: d, Temp: t, Press: p})
	}
	return out
}

// interp linearly interpolates temp/pressure at depth d over sorted valid pts.
func interp(pts []model.MeasurePoint, d float64) (float64, float64, bool) {
	if d < pts[0].DepthCal || d > pts[len(pts)-1].DepthCal {
		return 0, 0, false
	}
	for i := 1; i < len(pts); i++ {
		if d <= pts[i].DepthCal {
			a, b := pts[i-1], pts[i]
			if b.DepthCal == a.DepthCal {
				return b.Temp, b.Pressure, true
			}
			t := (d - a.DepthCal) / (b.DepthCal - a.DepthCal)
			return a.Temp + t*(b.Temp-a.Temp), a.Pressure + t*(b.Pressure-a.Pressure), true
		}
	}
	return 0, 0, false
}

// Gradient computes dT/dz and dP/dz between two grid points.
func Gradient(a, b GridPoint) (float64, float64) {
	dz := b.Depth - a.Depth
	if dz == 0 {
		return 0, 0
	}
	return (b.Temp - a.Temp) / dz, (b.Press - a.Press) / dz
}

// Segment is a detected well interval with its gradient signature.
type Segment struct {
	TopDepth    float64
	BottomDepth float64
	TempGrad    float64
	PressGrad   float64
	Jump        float64 // |Δtemp gradient| at the top boundary
}

// grad is a single resampled interval gradient sample.
type grad struct {
	depth, tg, pg float64
}

// Detect runs the full segmentation pipeline and returns candidate segments.
func Detect(grid []GridPoint, p Params) []Segment {
	if len(grid) < 2 {
		return nil
	}
	grads := make([]grad, 0, len(grid)-1)
	for i := 1; i < len(grid); i++ {
		tg, pg := Gradient(grid[i-1], grid[i])
		grads = append(grads, grad{depth: grid[i].Depth, tg: tg, pg: pg})
	}
	// Candidate boundaries where |Δtg| exceeds threshold.
	boundaries := []float64{grid[0].Depth}
	for i := 1; i < len(grads); i++ {
		jump := math.Abs(grads[i].tg - grads[i-1].tg)
		if jump >= p.JumpThreshold {
			boundaries = append(boundaries, grads[i].depth)
		}
	}
	boundaries = append(boundaries, grid[len(grid)-1].Depth)

	// Merge boundaries closer than MinSegThick.
	merged := []float64{boundaries[0]}
	for _, b := range boundaries[1:] {
		if b-merged[len(merged)-1] >= p.MinSegThick {
			merged = append(merged, b)
		}
	}
	if merged[len(merged)-1] != grid[len(grid)-1].Depth {
		merged = append(merged, grid[len(grid)-1].Depth)
	}

	var segs []Segment
	for i := 1; i < len(merged); i++ {
		top, bot := merged[i-1], merged[i]
		tg, pg := avgGradient(grads, top, bot)
		var jump float64
		if i > 1 {
			jump = math.Abs(gradAt(grads, top) - gradAt(grads, merged[i-2]))
		}
		segs = append(segs, Segment{
			TopDepth: top, BottomDepth: bot, TempGrad: tg, PressGrad: pg, Jump: jump,
		})
	}
	return segs
}

func avgGradient(grads []grad, top, bot float64) (float64, float64) {
	var st, sp, n float64
	for _, g := range grads {
		if g.depth > top && g.depth <= bot {
			st += g.tg
			sp += g.pg
			n++
		}
	}
	if n == 0 {
		return 0, 0
	}
	return st / n, sp / n
}

func gradAt(grads []grad, depth float64) float64 {
	for _, g := range grads {
		if math.Abs(g.depth-depth) < 1e-9 {
			return g.tg
		}
	}
	return 0
}

// Classify marks a segment as anomaly when its gradient jump is large relative
// to the run's median jump, otherwise stable.
func Classify(segs []Segment) []model.SegmentState {
	if len(segs) == 0 {
		return nil
	}
	jumps := make([]float64, len(segs))
	for i, s := range segs {
		jumps[i] = s.Jump
	}
	median := median(jumps)
	out := make([]model.SegmentState, len(segs))
	for i, s := range segs {
		if s.Jump > median*1.5 && s.Jump > 0 {
			out[i] = model.SegAnomaly
		} else {
			out[i] = model.SegStable
		}
	}
	return out
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	c := make([]float64, len(xs))
	copy(c, xs)
	// simple insertion sort for small slices
	for i := 1; i < len(c); i++ {
		v, j := c[i], i-1
		for j >= 0 && c[j] > v {
			c[j+1] = c[j]
			j--
		}
		c[j+1] = v
	}
	m := len(c) / 2
	if len(c)%2 == 1 {
		return c[m]
	}
	return (c[m-1] + c[m]) / 2
}

// ToModel converts computed segments into persistable domain entities.
func ToModel(runID int64, segs []Segment, states []model.SegmentState) []model.Segment {
	out := make([]model.Segment, len(segs))
	for i, s := range segs {
		st := model.SegCandidate
		if i < len(states) {
			st = states[i]
		}
		out[i] = model.Segment{
			WellRunID:   runID,
			Index:       i + 1,
			TopDepth:    s.TopDepth,
			BottomDepth: s.BottomDepth,
			TempGrad:    s.TempGrad,
			PressGrad:   s.PressGrad,
			GradientJump: s.Jump,
			State:       st,
			Confirmed:   false,
		}
	}
	return out
}
