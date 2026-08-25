// Package report builds human-readable summaries of a well run's layering
// result and cross-run comparison, used by engineers reviewing anomalies.
package report

import (
	"fmt"
	"strings"

	"task227-geowell/internal/model"
)

// RunProfile summarizes one run's segments.
type RunProfile struct {
	RunID    int64
	Well     string
	Segments []model.Segment
	Anomaly  int
}

// BuildProfile constructs a profile from a run and its segments.
func BuildProfile(run *model.WellRun, segs []model.Segment) RunProfile {
	p := RunProfile{RunID: run.ID, Well: run.Well, Segments: segs}
	for _, s := range segs {
		if s.State == model.SegAnomaly {
			p.Anomaly++
		}
	}
	return p
}

// Text renders the profile as a plain-text report.
func (p RunProfile) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "井次 #%d (%s) 分段报告\n", p.RunID, p.Well)
	fmt.Fprintf(&b, "共 %d 段，异常 %d 段\n", len(p.Segments), p.Anomaly)
	for _, s := range p.Segments {
		fmt.Fprintf(&b, "  [%d] %.1f-%.1f m  温梯 %.4f  压梯 %.4f  跳变 %.4f  [%s]\n",
			s.Index, s.TopDepth, s.BottomDepth, s.TempGrad, s.PressGrad, s.GradientJump, s.State)
	}
	return b.String()
}

// Comparison summarizes a cross-run movement set.
type Comparison struct {
	Well         string
	BaselineRun  int64
	TargetRun    int64
	Movements    []MovementLine
	AnomalyCount int
}

// MovementLine is one boundary movement row in the report.
type MovementLine struct {
	Baseline float64
	Target   float64
	Delta    float64
	Anomaly  bool
}

// Text renders the comparison as a plain-text report.
func (c Comparison) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "井 %s 跨次对比 (#%d -> #%d)\n", c.Well, c.BaselineRun, c.TargetRun)
	fmt.Fprintf(&b, "边界 %d 条，异常 %d 条\n", len(c.Movements), c.AnomalyCount)
	for _, m := range c.Movements {
		flag := ""
		if m.Anomaly {
			flag = "  <== 异常"
		}
		fmt.Fprintf(&b, "  基线 %.1f m -> 目标 %.1f m (Δ %.1f)%s\n", m.Baseline, m.Target, m.Delta, flag)
	}
	return b.String()
}
