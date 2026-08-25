package service_test

import (
	"errors"
	"path/filepath"
	"testing"

	"task227-geowell/internal/ingest"
	"task227-geowell/internal/model"
	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

func TestTask227Bug15RetryUnderReviewSnapshotAfterReloadReusesArtifact(t *testing.T) {
	db := filepath.Join(t.TempDir(), "bug15.db")
	st, err := store.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	svc := service.New(st)
	makeRun := func(code string) int64 {
		run, err := svc.CreateWellRun(code, code, "WELL-REVIEW", "2026-08-25", "C", "MPa")
		if err != nil {
			t.Fatal(err)
		}
		if err := svc.IngestPoints(run.ID, []ingest.RawPoint{{Seq: 0, Depth: 0, Temp: 40, Pressure: 0}, {Seq: 1, Depth: 10, Temp: 41, Pressure: 1}}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Layer(run.ID); err != nil {
			t.Fatal(err)
		}
		return run.ID
	}
	base, target := makeRun("REVIEW-BASE"), makeRun("REVIEW-TARGET")
	if err := st.SetWellRunMeta(base, 0, true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PublishComparison(base, target); !errors.Is(err, model.ErrInvalidSnapshot) {
		t.Fatalf("first publish error=%v, want disturbance rejection", err)
	}
	open, err := st.ListSnapshotsByWell("WELL-REVIEW")
	if err != nil || len(open) != 1 || open[0].State != model.SnapUnderReview {
		t.Fatalf("after first publish snapshots=%+v err=%v", open, err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = store.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.SetWellRunMeta(base, 0, false); err != nil {
		t.Fatal(err)
	}
	if _, err := service.New(st).PublishComparison(base, target); err != nil {
		t.Fatalf("retry after reload failed: %v", err)
	}
	all, err := st.ListSnapshotsByWell("WELL-REVIEW")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].State != model.SnapPublished {
		t.Fatalf("snapshots after retry=%+v, want one published artifact", all)
	}
}
