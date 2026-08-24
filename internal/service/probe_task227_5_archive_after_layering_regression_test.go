package service_test

import (
	"path/filepath"
	"testing"

	"task227-geowell/internal/model"
	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

func TestTask227Bug05CollectingRunCannotBeArchived(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug05.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	run, err := svc.CreateWellRun("ARCH-001", "archive", "WELL-ARCH", "2026-08-25", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ArchiveRun(run.ID); err == nil {
		t.Fatal("ArchiveRun archived a collecting run")
	}
	stored, err := st.GetWellRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != model.WellRunCollecting || stored.Archived {
		t.Fatalf("run after rejected archive = %+v", stored)
	}
}
