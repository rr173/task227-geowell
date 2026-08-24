// Package metrics computes aggregate statistics over a well run's points and
// segments: mean gradients, depth coverage, anomaly ratio and data completeness.
package metrics

import (
	"math"

	"task227-geowell/internal/model"
)

// RunMetrics summarizes a run's profile numerically.
type RunMetrics struct {
	PointCount    int
	SegmentCount  int
	AnomalyCount  int
	MeanTempGrad  float64
	MeanPressGrad float64
	MaxJump       float64
	DepthTop      float64
	DepthBottom   float64
	Coverage      float64 // (bottom-top) span in metres
	Completeness  float64 // 1 - gaps/total points
}

// Compute derives metrics from the points and segments of one run.
func Compute(pts []model.MeasurePoint, segs []model.Segment) RunMetrics {
	m := RunMetrics{PointCount: len(pts), SegmentCount: len(segs)}
	if len(pts) == 0 {
		return m
	}
	var gaps int
	top, bottom := math.MaxFloat64, -math.MaxFloat64
	for _, p := range pts {
		if p.Gap {
			gaps++
		}
		if p.DepthCal < top {
			top = p.DepthCal
		}
		if p.DepthCal > bottom {
			bottom = p.DepthCal
		}
	}
	m.DepthTop = top
	m.DepthBottom = bottom
	m.Coverage = bottom - top
	m.Completeness = 1 - float64(gaps)/float64(len(pts))

	var st, sp, maxJ float64
	for _, s := range segs {
		st += s.TempGrad
		sp += s.PressGrad
		if s.GradientJump > maxJ {
			maxJ = s.GradientJump
		}
		if s.State == model.SegAnomaly {
			m.AnomalyCount++
		}
	}
	if len(segs) > 0 {
		m.MeanTempGrad = st / float64(len(segs))
		m.MeanPressGrad = sp / float64(len(segs))
	}
	m.MaxJump = maxJ
	return m
}
