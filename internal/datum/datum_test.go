package datum

import (
	"errors"
	"testing"

	"task227-geowell/internal/model"
)

func TestComputeAndApplyAlignReferenceDepth(t *testing.T) {
	pts := []model.MeasurePoint{
		{DepthRaw: 100, DepthCal: 100, Temp: 40},
		{DepthRaw: 110, DepthCal: 110, Temp: 41},
	}
	c := Compute(pts, 105)
	if c.Offset != 5 || c.DepthBasis != 105 {
		t.Fatalf("correction = %+v, want offset=5 basis=105", c)
	}
	cal := Apply(pts, c)
	if cal[0].DepthCal != 105 || cal[1].DepthCal != 115 {
		t.Fatalf("calibrated depths = %v, want [105 115]", []float64{cal[0].DepthCal, cal[1].DepthCal})
	}
}

func TestVerifyMonotonicRejectsCorrectedCollision(t *testing.T) {
	pts := []model.MeasurePoint{
		{DepthCal: 10},
		{DepthCal: 10},
	}
	if err := VerifyMonotonic(pts); !errors.Is(err, model.ErrDepthNotMonotonic) {
		t.Fatalf("VerifyMonotonic() error = %v, want depth error", err)
	}
}
