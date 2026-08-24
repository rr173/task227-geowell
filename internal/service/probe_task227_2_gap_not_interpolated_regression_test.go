package service_test

import (
	"errors"
	"path/filepath"
	"testing"

	"task227-geowell/internal/ingest"
	"task227-geowell/internal/model"
	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
	"task227-geowell/internal/layer"
)

func TestTask227Bug02GapDoesNotCreateSyntheticProfile(t *testing.T) {
	points := []model.MeasurePoint{
		{DepthCal: 0, Temp: 40, Pressure: 0, State: model.PointValid},
		{DepthCal: 10, Temp: 200, Pressure: 10, State: model.PointMissing, Gap: true},
		{DepthCal: 20, Temp: 42, Pressure: 2, State: model.PointValid},
	}
	grid := layer.Resample(points, 5)
	if len(grid) != 0 {
		t.Fatalf("Resample bridged a missing interval and produced %d samples: %+v", len(grid), grid)
	}
	st, err := store.Open(filepath.Join(t.TempDir(), "bug02.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	run, err := svc.CreateWellRun("GAP-001", "gap", "WELL-GAP", "2026-08-25", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.IngestPoints(run.ID, []ingest.RawPoint{{Seq: 0, Depth: 0, Missing: true}, {Seq: 1, Depth: 5, Missing: true}}); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Layer(run.ID)
	if !errors.Is(err, model.ErrBadTransition) {
		t.Fatalf("Layer(all missing) error = %v, want readiness error", err)
	}
}
