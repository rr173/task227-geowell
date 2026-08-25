package layer

import (
	"testing"

	"task227-geowell/internal/model"
)

func TestResampleInterpolatesUniformGrid(t *testing.T) {
	pts := []model.MeasurePoint{
		{DepthCal: 0, Temp: 10, Pressure: 1},
		{DepthCal: 10, Temp: 20, Pressure: 3},
	}
	grid := Resample(pts, 5)
	if len(grid) != 3 || grid[1].Depth != 5 || grid[1].Temp != 15 || grid[1].Press != 2 {
		t.Fatalf("Resample() = %+v, want midpoint at depth 5", grid)
	}
}

func TestDetectSeparatesGradientJump(t *testing.T) {
	grid := []GridPoint{
		{Depth: 0, Temp: 0, Press: 0},
		{Depth: 5, Temp: 1, Press: 1},
		{Depth: 10, Temp: 2, Press: 2},
		{Depth: 15, Temp: 7, Press: 3},
		{Depth: 20, Temp: 12, Press: 4},
	}
	segs := Detect(grid, Params{GridStep: 5, JumpThreshold: 0.5, MinSegThick: 5})
	if len(segs) < 2 {
		t.Fatalf("Detect() returned %d segments, want a boundary", len(segs))
	}
	if segs[1].Jump <= 0 {
		t.Fatalf("second segment jump = %v, want positive", segs[1].Jump)
	}
}
