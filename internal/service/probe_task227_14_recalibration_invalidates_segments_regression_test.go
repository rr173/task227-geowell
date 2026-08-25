package service_test

import (
	"path/filepath"
	"testing"

	"task227-geowell/internal/ingest"
	"task227-geowell/internal/model"
	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

func TestTask227Bug14RecalibrationInvalidatesStoredLayering(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug14.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	run, err := svc.CreateWellRun("RECAL-001", "recal", "WELL-RECAL", "2026-08-25", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.IngestPoints(run.ID, []ingest.RawPoint{{Seq: 0, Depth: 0, Temp: 40, Pressure: 0}, {Seq: 1, Depth: 10, Temp: 41, Pressure: 1}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Layer(run.ID); err != nil {
		t.Fatal(err)
	}
	before, err := st.ListSegments(run.ID)
	if err != nil || len(before) == 0 {
		t.Fatalf("initial segments=%d err=%v", len(before), err)
	}
	if _, err := svc.CorrectDepth(run.ID, 5); err != nil {
		t.Fatal(err)
	}
	afterRun, err := st.GetWellRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	afterSegs, err := st.ListSegments(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterRun.State != model.WellRunPendingLayering || len(afterSegs) != 0 {
		t.Fatalf("after recalibration run=%+v segments=%d, want pending_layering and no stale segments", afterRun, len(afterSegs))
	}
	if _, err := svc.Layer(run.ID); err != nil {
		t.Fatalf("run could not be layered again after recalibration: %v", err)
	}
}
