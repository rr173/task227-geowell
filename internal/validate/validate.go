// Package validate holds cross-field validation rules that span multiple
// entities: unit consistency between well run and points, disturbance handling,
// and layering-readiness checks.
package validate

import (
	"fmt"

	"task227-geowell/internal/model"
)

// RunReadiness reports whether a well run has enough valid data to be layered.
func RunReadiness(r *model.WellRun, pointCount, segmentCount int) error {
	if r.Archived {
		return model.ErrArchivedMutation
	}
	if r.State != model.WellRunPendingLayering && r.State != model.WellRunLayered {
		return fmt.Errorf("%w: state=%s", model.ErrBadTransition, r.State)
	}
	if pointCount < 2 {
		return fmt.Errorf("need at least 2 valid points, have %d", pointCount)
	}
	return nil
}

// UnitConsistent asserts a point's declared units match the run's units.
// Points themselves carry no unit (they inherit the run's), but this guards
// against mismatched run configurations when comparing two runs.
func UnitConsistent(a, b *model.WellRun) error {
	if a.UnitTemp != b.UnitTemp {
		return fmt.Errorf("temperature unit mismatch: %s vs %s", a.UnitTemp, b.UnitTemp)
	}
	if a.UnitPress != b.UnitPress {
		return fmt.Errorf("pressure unit mismatch: %s vs %s", a.UnitPress, b.UnitPress)
	}
	return nil
}

// DisturbanceResolved returns an error if either run still carries a wellhead
// disturbance that must be excluded before the comparison is trustworthy.
func DisturbanceResolved(runs ...*model.WellRun) error {
	for _, r := range runs {
		if r.Disturbance {
			return fmt.Errorf("%w: run %d still flagged", model.ErrInvalidSnapshot, r.ID)
		}
	}
	return nil
}

// SegmentConfirmable checks a segment can move to confirmed (must not already
// be confirmed and must be a terminal-class state ready for sign-off).
func SegmentConfirmable(s *model.Segment) error {
	if s.Confirmed {
		return fmt.Errorf("segment %d already confirmed", s.ID)
	}
	if s.State != model.SegStable && s.State != model.SegAnomaly && s.State != model.SegCandidate {
		return fmt.Errorf("%w: segment state=%s", model.ErrBadTransition, s.State)
	}
	return nil
}
