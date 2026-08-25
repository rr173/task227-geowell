// Package calibrate provides depth calibration strategies beyond the simple
// datum offset. It supports a reference-marker tie point and a two-point
// stretch correction used when the logging tool reports a depth bias that
// grows with depth.
package calibrate

import (
	"task227-geowell/internal/model"
)

// Strategy selects the calibration approach.
type Strategy string

const (
	Offset     Strategy = "offset"     // single datum shift
	TiePoint   Strategy = "tie_point"  // align one known marker depth
	Stretch    Strategy = "stretch"    // linear depth scaling from two markers
)

// Marker is a known calibrated depth tied to a raw measurement.
type Marker struct {
	Raw    float64
	Cal    float64
}

// Plan describes the resulting depth transform.
type Plan struct {
	Strategy Strategy
	Offset   float64
	Scale    float64 // applied as cal = raw*scale + offset
}

// Build computes the transform for the chosen strategy.
func Build(s Strategy, markers []Marker) (Plan, error) {
	switch s {
	case Offset:
		if len(markers) == 0 {
			return Plan{Strategy: Offset}, nil
		}
		return Plan{Strategy: Offset, Offset: markers[0].Cal - markers[0].Raw}, nil
	case TiePoint:
		if len(markers) < 1 {
			return Plan{}, mathRangeErr("tie_point requires >=1 marker")
		}
		m := markers[0]
		return Plan{Strategy: TiePoint, Offset: m.Cal - m.Raw, Scale: 1}, nil
	case Stretch:
		if len(markers) < 2 {
			return Plan{}, mathRangeErr("stretch requires >=2 markers")
		}
		a, b := markers[0], markers[1]
		if a.Raw == b.Raw {
			return Plan{}, mathRangeErr("markers share raw depth")
		}
		scale := (b.Cal - a.Cal) / (b.Raw - a.Raw)
		offset := a.Cal - scale*a.Raw
		return Plan{Strategy: Stretch, Offset: offset, Scale: scale}, nil
	default:
		return Plan{}, mathRangeErr("unknown strategy")
	}
}

// Apply transforms a raw depth into a calibrated depth.
func (p Plan) Apply(raw float64) float64 {
	return raw*p.Scale + p.Offset
}

// ApplyPoints returns corrected copies of the points using the plan.
func ApplyPoints(pts []model.MeasurePoint, p Plan) []model.MeasurePoint {
	out := make([]model.MeasurePoint, len(pts))
	for i, pt := range pts {
		np := pt
		np.DepthCal = p.Apply(pt.DepthRaw)
		out[i] = np
	}
	return out
}

func mathRangeErr(msg string) error {
	return &calibrateError{msg: msg}
}

type calibrateError struct{ msg string }

func (e *calibrateError) Error() string { return e.msg }
