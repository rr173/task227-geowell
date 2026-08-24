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

func TestTask227Bug07RetryWithChangedMeasurementDoesNotOverwrite(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug07.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	run, err := svc.CreateWellRun("POINT-001", "points", "WELL-POINT", "2026-08-25", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	first := []ingest.RawPoint{{Seq: 0, Depth: 10, Temp: 40, Pressure: 1}}
	if err := svc.IngestPoints(run.ID, first); err != nil {
		t.Fatal(err)
	}
	if err := svc.IngestPoints(run.ID, []ingest.RawPoint{{Seq: 0, Depth: 10, Temp: 99, Pressure: 1}}); !errors.Is(err, model.ErrDuplicatePoint) {
		t.Fatalf("changed retry error = %v, want duplicate-point conflict", err)
	}
	points, err := st.ListPoints(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 || points[0].Temp != 40 {
		t.Fatalf("stored points after changed retry = %+v, want original temperature 40", points)
	}
}
