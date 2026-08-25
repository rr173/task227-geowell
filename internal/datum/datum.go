// Package datum implements depth datum correction. Raw measured depths are
// shifted onto a common reference so that points from different logging runs
// can be compared along a single depth axis.
package datum

import (
	"fmt"
	"math"

	"task227-geowell/internal/model"
)

// Correction holds the computed datum shift and the wellhead disturbance flag.
type Correction struct {
	DepthBasis  float64 // corrected datum (m) applied as an offset
	Disturbance bool    // wellhead disturbance detected near surface
	Offset      float64 // raw -> cal offset (cal = raw + offset)
}

// Compute estimates the datum offset between raw and reference depths. If a
// reference caliber (e.g. a casing shoe at a known calibrated depth) is
// supplied, the offset aligns raw depth to that reference. Without a reference
// the offset is zero and depths pass through unchanged.
func Compute(points []model.MeasurePoint, referenceDepth float64) Correction {
	c := Correction{}
	if len(points) == 0 {
		return c
	}
	// Detect wellhead disturbance: a non-monotonic or anomalous shallow
	// temperature rise within the first 5% of the profile.
	c.Disturbance = detectWellheadDisturbance(points)
	// If a calibrated reference depth is provided, align the nearest raw point.
	if referenceDepth > 0 {
		nearest := points[0]
		best := math.Abs(nearest.DepthRaw - referenceDepth)
		for _, p := range points {
			if d := math.Abs(p.DepthRaw - referenceDepth); d < best {
				best = d
				nearest = p
			}
		}
		c.Offset = referenceDepth - nearest.DepthRaw
		c.DepthBasis = referenceDepth
	}
	return c
}

// Apply returns a corrected copy of the points using the offset.
func Apply(points []model.MeasurePoint, c Correction) []model.MeasurePoint {
	out := make([]model.MeasurePoint, len(points))
	for i, p := range points {
		np := p
		np.DepthCal = p.DepthRaw + c.Offset
		// A depth-conflict point is one whose corrected depth collides with the
		// previous corrected depth (should not happen after monotonic raw input,
		// but keep the state available for downstream exclusion).
		if i > 0 && np.DepthCal <= out[i-1].DepthCal && !np.Gap {
			np.State = model.PointDepthConflict
		}
		out[i] = np
	}
	return out
}

// detectWellheadDisturbance flags a surface anomaly: a shallow temperature
// inversion (cooling then sharp warming within the top band) that indicates a
// wellhead influence rather than formation signal.
func detectWellheadDisturbance(points []model.MeasurePoint) bool {
	if len(points) < 4 {
		return false
	}
	n := len(points)
	band := int(math.Max(2, float64(n)*0.05))
	var temps []float64
	for i := 0; i < band && i < n; i++ {
		if !points[i].Gap {
			temps = append(temps, points[i].Temp)
		}
	}
	if len(temps) < 3 {
		return false
	}
	// inversion: temps[0] > temps[1] and temps[1] < temps[2] (dip then rise)
	if temps[0] > temps[1] && temps[1] < temps[2] {
		return true
	}
	return false
}

// VerifyMonotonic asserts corrected depths stay strictly increasing.
func VerifyMonotonic(points []model.MeasurePoint) error {
	prev := math.Inf(-1)
	for _, p := range points {
		if p.Gap {
			prev = p.DepthCal
			continue
		}
		if p.DepthCal <= prev {
			return fmt.Errorf("%w: corrected depth %.3f", model.ErrDepthNotMonotonic, p.DepthCal)
		}
		prev = p.DepthCal
	}
	return nil
}
