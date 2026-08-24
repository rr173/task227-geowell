package service

import (
	"path/filepath"
	"testing"

	"task227-geowell/internal/ingest"
	"task227-geowell/internal/model"
	"task227-geowell/internal/store"
)

func TestTask227Bug10ConfirmedSegmentCannotRegress(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug10.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st)
	run, err := svc.CreateWellRun("SEG-001", "segments", "WELL-SEG", "2026-08-25", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	var pts []ingest.RawPoint
	for i := 0; i <= 20; i++ {
		pts = append(pts, ingest.RawPoint{Seq: i, Depth: float64(i * 5), Temp: 40 + float64(i), Pressure: float64(i)})
	}
	if err := svc.IngestPoints(run.ID, pts); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CorrectDepth(run.ID, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Layer(run.ID); err != nil {
		t.Fatal(err)
	}
	segments, err := st.ListSegments(run.ID)
	if err != nil || len(segments) == 0 {
		t.Fatalf("segments=%+v, err=%v", segments, err)
	}
	if err := svc.ConfirmSegment(segments[0].ID, model.SegConfirmed); err != nil {
		t.Fatalf("confirm segment: %v", err)
	}
	if err := svc.ConfirmSegment(segments[0].ID, model.SegStable); err == nil {
		t.Fatal("confirmed segment was allowed to regress to stable")
	}
	segments, err = st.ListSegments(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if segments[0].State != model.SegConfirmed || !segments[0].Confirmed {
		t.Fatalf("segment after rejected regression = %+v, want confirmed", segments[0])
	}
}
