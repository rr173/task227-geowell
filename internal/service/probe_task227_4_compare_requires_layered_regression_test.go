package service_test

import (
	"path/filepath"
	"testing"

	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

func TestTask227Bug04CompareRejectsUnlayeredRuns(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug04.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	a, err := svc.CreateWellRun("CMP-001", "a", "WELL-CMP", "2026-08-01", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateWellRun("CMP-002", "b", "WELL-CMP", "2026-08-02", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CompareRuns(a.ID, b.ID); err == nil {
		t.Fatal("CompareRuns accepted collecting runs without layered profiles")
	}
}
