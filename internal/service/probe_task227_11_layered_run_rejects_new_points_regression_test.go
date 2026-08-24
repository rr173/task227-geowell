package service

import (
	"path/filepath"
	"testing"

	"task227-geowell/internal/ingest"
	"task227-geowell/internal/model"
	"task227-geowell/internal/store"
)

func TestTask227Bug11LayeredRunRejectsNewPoints(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug11.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st)
	run, err := svc.CreateWellRun("LAYERED-001", "layered", "WELL-LAYERED", "2026-08-25", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	initial := []ingest.RawPoint{
		{Seq: 0, Depth: 0, Temp: 40, Pressure: 0},
		{Seq: 1, Depth: 10, Temp: 41, Pressure: 1},
	}
	if err := svc.IngestPoints(run.ID, initial); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CorrectDepth(run.ID, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Layer(run.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.IngestPoints(run.ID, []ingest.RawPoint{{Seq: 2, Depth: 20, Temp: 42, Pressure: 2}}); err == nil {
		t.Fatal("layered run accepted new measurement points")
	}
	run, err = st.GetWellRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if run.State != model.WellRunLayered {
		t.Fatalf("run state after rejected ingest=%s, want layered", run.State)
	}
	points, err := st.ListPoints(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != len(initial) {
		t.Fatalf("stored points=%d, want %d", len(points), len(initial))
	}
}
