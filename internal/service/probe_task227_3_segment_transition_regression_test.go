package service_test

import (
	"path/filepath"
	"testing"

	"task227-geowell/internal/ingest"
	"task227-geowell/internal/model"
	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

func TestTask227Bug03ConfirmedSegmentCannotMoveBack(t *testing.T) {
	if model.SegConfirmed.CanTransition(model.SegStable) {
		t.Fatal("confirmed segment unexpectedly permits moving back to stable")
	}
	st, err := store.Open(filepath.Join(t.TempDir(), "bug03.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	run, err := svc.CreateWellRun("SEG-001", "segment", "WELL-SEG", "2026-08-25", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.IngestPoints(run.ID, []ingest.RawPoint{{Seq: 0, Depth: 0, Temp: 40, Pressure: 0}, {Seq: 1, Depth: 10, Temp: 41, Pressure: 1}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CorrectDepth(run.ID, 0); err != nil {
		t.Fatal(err)
	}
	segs, err := svc.Layer(run.ID)
	if err != nil || len(segs) == 0 {
		t.Fatalf("Layer() = %v, segments=%d", err, len(segs))
	}
	stored, err := st.ListSegments(run.ID)
	if err != nil || len(stored) == 0 {
		t.Fatalf("ListSegments() = %v, segments=%d", err, len(stored))
	}
	if err := svc.ConfirmSegment(stored[0].ID, model.SegConfirmed); err != nil {
		t.Fatal(err)
	}
	if err := svc.ConfirmSegment(stored[0].ID, model.SegStable); err == nil {
		t.Fatal("confirmed segment was allowed to move back to stable")
	}
}
