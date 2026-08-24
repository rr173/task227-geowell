// Package snapshot implements comparison snapshot publication: it builds a
// versioned, immutable comparison artifact and manages its state machine
// (draft -> under_review -> published -> superseded).
package snapshot

import (
	"fmt"

	"task227-geowell/internal/model"
)

// Publisher builds and versions snapshots for a well.
type Publisher struct {
	well    string
	version int
}

// NewPublisher creates a publisher seeded with the latest stored version.
func NewPublisher(well string, latestVersion int) *Publisher {
	return &Publisher{well: well, version: latestVersion}
}

// NextCode returns the next versioned code for this well.
func (p *Publisher) NextCode() (string, int) {
	p.version++
	return fmt.Sprintf("%s-cmp-v%d", p.well, p.version), p.version
}

// Build constructs a snapshot entity in draft state from a detail JSON string.
func (p *Publisher) Build(code string, version int, baselineID, targetID int64, detail string) *model.ComparisonSnapshot {
	return &model.ComparisonSnapshot{
		Code:         code,
		Well:         p.well,
		Version:      version,
		State:        model.SnapDraft,
		BaselineRunID: baselineID,
		TargetRunID:   targetID,
		Detail:       detail,
	}
}

// CanPublish asserts the snapshot is in a publishable state and that the
// disturbance on either run does not taint the comparison.
func CanPublish(snap *model.ComparisonSnapshot, baselineDisturbance, targetDisturbance bool) error {
	if snap.State != model.SnapUnderReview {
		return fmt.Errorf("%w: state=%s", model.ErrInvalidSnapshot, snap.State)
	}
	if baselineDisturbance || targetDisturbance {
		return fmt.Errorf("%w: wellhead disturbance present, exclude before publishing", model.ErrInvalidSnapshot)
	}
	return nil
}
