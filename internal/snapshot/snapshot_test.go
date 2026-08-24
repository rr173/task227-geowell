package snapshot

import (
	"testing"

	"task227-geowell/internal/model"
)

func TestPublisherAdvancesVersionAndBuildsDraft(t *testing.T) {
	p := NewPublisher("WELL-A", 3)
	code, version := p.NextCode()
	if code != "WELL-A-cmp-v4" || version != 4 {
		t.Fatalf("NextCode() = %q, %d; want WELL-A-cmp-v4, 4", code, version)
	}
	snap := p.Build(code, version, 11, 12, "[]")
	if snap.State != model.SnapDraft || snap.BaselineRunID != 11 || snap.TargetRunID != 12 {
		t.Fatalf("Build() = %+v, want draft with run IDs", snap)
	}
}

func TestCanPublishBlocksDisturbedRun(t *testing.T) {
	snap := &model.ComparisonSnapshot{State: model.SnapUnderReview}
	if err := CanPublish(snap, true, false); err == nil {
		t.Fatal("CanPublish() accepted a disturbed baseline")
	}
	if err := CanPublish(snap, false, false); err != nil {
		t.Fatalf("CanPublish() rejected clean runs: %v", err)
	}
}
