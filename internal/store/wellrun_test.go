package store

import (
	"path/filepath"
	"testing"

	"task227-geowell/internal/model"
)

// newTestStore opens a fresh store on a temp database and returns it with a
// cleanup closer.
func newTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return st, func() { _ = st.Close() }
}

// TestDisturbanceFlagPersistsAcrossReads guards the regression where
// scanWellRun scanned the disturbance column into a local variable but never
// copied it back onto the entity. GetWellRun must round-trip whatever
// SetWellRunMeta persisted, otherwise PublishComparison would treat a
// disturbed run as publishable.
func TestDisturbanceFlagPersistsAcrossReads(t *testing.T) {
	st, cleanup := newTestStore(t)
	defer cleanup()

	r := &model.WellRun{
		Code:      "T-D-001",
		Name:      "disturbed",
		Well:      "W-D",
		LogDate:   "2026-08-01",
		State:     model.WellRunCollecting,
		UnitTemp:  model.TempCelsius,
		UnitPress: model.PressMPa,
	}
	if err := st.CreateWellRun(r); err != nil {
		t.Fatalf("create run: %v", err)
	}

	// Persist a disturbance flag and a non-zero depth basis.
	if err := st.SetWellRunMeta(r.ID, 123.4, true); err != nil {
		t.Fatalf("set meta: %v", err)
	}

	got, err := st.GetWellRun(r.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if !got.Disturbance {
		t.Fatalf("Disturbance = false, want true (flag must survive a read)")
	}
	if got.DepthBasis != 123.4 {
		t.Fatalf("DepthBasis = %v, want 123.4", got.DepthBasis)
	}

	// And confirm a clean run still reads back clean.
	if err := st.SetWellRunMeta(r.ID, 0, false); err != nil {
		t.Fatalf("clear meta: %v", err)
	}
	got, err = st.GetWellRun(r.ID)
	if err != nil {
		t.Fatalf("get run after clear: %v", err)
	}
	if got.Disturbance {
		t.Fatalf("Disturbance = true, want false after clearing")
	}
}
