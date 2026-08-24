// Package ingest implements the well-logging ingestion module: it receives
// raw measurement points, validates depth monotonicity, detects duplicates and
// computes a content fingerprint for idempotent upload.
package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"task227-geowell/internal/model"
)

// RawPoint is the external input for one sampled depth reading.
type RawPoint struct {
	Seq      int     `json:"seq"`
	Depth    float64 `json:"depth"`
	Temp     float64 `json:"temp"`
	Pressure float64 `json:"pressure"`
	Missing  bool    `json:"missing,omitempty"`
}

// Fingerprint builds a stable hash over the sorted (seq,depth,temp,pressure)
// tuples so identical uploads can be detected as idempotent.
func Fingerprint(pts []RawPoint) string {
	cp := make([]RawPoint, len(pts))
	copy(cp, pts)
	sort.Slice(cp, func(i, j int) bool { return cp[i].Seq < cp[j].Seq })
	var b strings.Builder
	for _, p := range cp {
		b.WriteString(fmt.Sprintf("%d:%.4f:%.4f:%.4f:%v;", p.Seq, p.Depth, p.Temp, p.Pressure, p.Missing))
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// Validate enforces domain rules on an incoming batch:
//   - depth must be strictly increasing (no depth倒序)
//   - unit must be declared (caller passes via WellRun, not here)
//   - no duplicate seq
func Validate(pts []RawPoint) error {
	seen := make(map[int]bool, len(pts))
	prev := -1.0
	for _, p := range pts {
		if p.Seq < 0 {
			return fmt.Errorf("negative seq %d", p.Seq)
		}
		if seen[p.Seq] {
			return fmt.Errorf("%w: seq=%d", model.ErrDuplicatePoint, p.Seq)
		}
		seen[p.Seq] = true
		if p.Missing {
			prev = p.Depth
			continue
		}
		if p.Depth <= prev {
			return fmt.Errorf("%w: depth %.3f not greater than %.3f at seq=%d",
				model.ErrDepthNotMonotonic, p.Depth, prev, p.Seq)
		}
		prev = p.Depth
	}
	return nil
}

// ToMeasurePoints converts validated raw points into domain entities. Depth
// correction is applied by the datum module afterwards; here depth_cal mirrors
// depth_raw until correction runs.
func ToMeasurePoints(runID int64, pts []RawPoint) []model.MeasurePoint {
	out := make([]model.MeasurePoint, 0, len(pts))
	for _, p := range pts {
		st := model.PointValid
		if p.Missing {
			st = model.PointMissing
		}
		out = append(out, model.MeasurePoint{
			WellRunID: runID,
			Seq:       p.Seq,
			DepthRaw:  p.Depth,
			DepthCal:  p.Depth,
			Temp:      p.Temp,
			Pressure:  p.Pressure,
			State:     st,
			Gap:       p.Missing,
		})
	}
	return out
}

// EncodeUnit is a small helper to assert a declared unit string is known.
func EncodeUnit(tu, pu string) (model.TemperatureUnit, model.PressureUnit, error) {
	var t model.TemperatureUnit
	switch model.TemperatureUnit(tu) {
	case model.TempCelsius, model.TempKelvin:
		t = model.TemperatureUnit(tu)
	default:
		return "", "", model.ErrUnitNotDeclared
	}
	var p model.PressureUnit
	switch model.PressureUnit(pu) {
	case model.PressMPa, model.PressBar, model.PressKPa:
		p = model.PressureUnit(pu)
	default:
		return "", "", model.ErrUnitNotDeclared
	}
	return t, p, nil
}

// FormatDepth is a presentation helper used by reports.
func FormatDepth(d float64) string {
	return strconv.FormatFloat(d, 'f', 2, 64) + " m"
}
