package service_test

import (
	"path/filepath"
	"testing"

	"task227-geowell/internal/model"
	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

// newTestStore opens a fresh store on a temp database and returns it with a
// cleanup closer, so service tests can also reach store methods that the
// service keeps private (used here to seed the persisted disturbance flag).
func newTestStore(t *testing.T) (*store.Store, func()) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return st, func() { _ = st.Close() }
}

// TestPublishBlocksWhenDisturbancePersisted reproduces the regression end to
// end: once CorrectDepth has marked a wellhead disturbance on a participating
// run, PublishComparison must refuse to publish the cross-run snapshot. Before
// the scanWellRun fix, the disturbance flag was dropped on read, so the
// service saw a clean run and published anyway.
//
// The disturbance is persisted directly via the store (exactly what
// CorrectDepth/SetWellRunMeta writes) so the assertion targets the publish
// read-path and is not coupled to unrelated gradient/detection behavior.
func TestPublishBlocksWhenDisturbancePersisted(t *testing.T) {
	s, cleanup := newTestStore(t)
	defer cleanup()
	svc := service.New(s)

	baseline, err := svc.CreateWellRun("T-E-001", "base", "W-E", "2026-08-01", "C", "MPa")
	if err != nil {
		t.Fatalf("create baseline: %v", err)
	}
	target, err := svc.CreateWellRun("T-E-002", "tgt", "W-E", "2026-08-15", "C", "MPa")
	if err != nil {
		t.Fatalf("create target: %v", err)
	}

	ingestBatch(t, svc, baseline.ID, 800.0)
	ingestBatch(t, svc, target.ID, 800.0)

	if _, err := svc.CorrectDepth(baseline.ID, 0); err != nil {
		t.Fatalf("correct baseline: %v", err)
	}
	if _, err := svc.CorrectDepth(target.ID, 0); err != nil {
		t.Fatalf("correct target: %v", err)
	}

	// Persist a wellhead disturbance on the baseline, as CorrectDepth would
	// when datum detection flags one. This is the state the bug dropped on read.
	if err := s.SetWellRunMeta(baseline.ID, 0, true); err != nil {
		t.Fatalf("persist disturbance: %v", err)
	}

	if _, err := svc.Layer(baseline.ID); err != nil {
		t.Fatalf("layer baseline: %v", err)
	}
	if _, err := svc.Layer(target.ID); err != nil {
		t.Fatalf("layer target: %v", err)
	}

	snap, err := svc.PublishComparison(baseline.ID, target.ID)
	if err == nil {
		t.Fatalf("PublishComparison succeeded; expected disturbance to block publication, state=%s", snap.State)
	}
	if snap == nil {
		t.Fatalf("expected snapshot left in under_review with error, got nil snap; err=%v", err)
	}
	if snap.State != model.SnapUnderReview {
		t.Fatalf("snapshot state = %s, want under_review (disturbed run must not publish)", snap.State)
	}
}
