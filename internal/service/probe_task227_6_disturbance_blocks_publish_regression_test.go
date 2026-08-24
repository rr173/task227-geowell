package service_test

import (
	"path/filepath"
	"testing"

	"task227-geowell/internal/ingest"
	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

func TestTask227Bug06DisturbanceSurvivesReloadAndBlocksPublish(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug06.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	makeRun := func(code string) int64 {
		run, err := svc.CreateWellRun(code, code, "WELL-DIST", "2026-08-25", "C", "MPa")
		if err != nil {
			t.Fatal(err)
		}
		if err := svc.IngestPoints(run.ID, []ingest.RawPoint{{Seq: 0, Depth: 0, Temp: 40, Pressure: 0}, {Seq: 1, Depth: 10, Temp: 41, Pressure: 1}}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CorrectDepth(run.ID, 0); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Layer(run.ID); err != nil {
			t.Fatal(err)
		}
		return run.ID
	}
	base, target := makeRun("DIST-001"), makeRun("DIST-002")
	if err := st.SetWellRunMeta(base, 0, true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PublishComparison(base, target); err == nil {
		t.Fatal("PublishComparison published a comparison containing a disturbed run")
	}
}
