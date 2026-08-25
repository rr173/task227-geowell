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

func TestTask227Bug16ArchiveInvalidatesOpenComparison(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug16.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	makeRun := func(code string) int64 {
		run, err := svc.CreateWellRun(code, code, "WELL-ARCHIVE-SNAP", "2026-08-25", "C", "MPa")
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
	base, target := makeRun("ARCHIVE-BASE"), makeRun("ARCHIVE-TARGET")
	if err := st.SetWellRunMeta(base, 0, true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PublishComparison(base, target); !errors.Is(err, model.ErrInvalidSnapshot) {
		t.Fatalf("publish error=%v, want disturbance rejection", err)
	}
	if err := svc.ArchiveRun(base); err != nil {
		t.Fatal(err)
	}
	snaps, err := st.ListSnapshotsByWell("WELL-ARCHIVE-SNAP")
	if err != nil || len(snaps) != 1 || snaps[0].State != model.SnapSuperseded {
		t.Fatalf("snapshots after archive=%+v err=%v, want superseded", snaps, err)
	}
}
