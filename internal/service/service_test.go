package service_test

import (
	"path/filepath"
	"testing"

	"task227-geowell/internal/ingest"
	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

func newTestService(t *testing.T) (*service.Service, func()) {
	t.Helper()
	db := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(db)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return service.New(st), func() { _ = st.Close() }
}

func ingestBatch(t *testing.T, svc *service.Service, runID int64, boundary float64) {
	t.Helper()
	var pts []ingest.RawPoint
	depth, temp, seq := 0.0, 40.0, 0
	for depth <= 1500.0 {
		pts = append(pts, ingest.RawPoint{
			Seq:      seq,
			Depth:    depth,
			Temp:     temp,
			Pressure: 0.1 * depth,
		})
		seq++
		if depth < boundary {
			temp += 0.1
		} else {
			temp += 1.0
		}
		depth += 5.0
	}
	if err := svc.IngestPoints(runID, pts); err != nil {
		t.Fatalf("ingest: %v", err)
	}
}

func TestFullLoop(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	baseline, err := svc.CreateWellRun("T-A-001", "base", "W-A", "2026-08-01", "C", "MPa")
	if err != nil {
		t.Fatalf("create baseline: %v", err)
	}
	target, err := svc.CreateWellRun("T-A-002", "tgt", "W-A", "2026-08-15", "C", "MPa")
	if err != nil {
		t.Fatalf("create target: %v", err)
	}

	ingestBatch(t, svc, baseline.ID, 800.0)
	ingestBatch(t, svc, target.ID, 825.0)

	if _, err := svc.CorrectDepth(baseline.ID, 0); err != nil {
		t.Fatalf("correct baseline: %v", err)
	}
	if _, err := svc.CorrectDepth(target.ID, 0); err != nil {
		t.Fatalf("correct target: %v", err)
	}
	bs, err := svc.Layer(baseline.ID)
	if err != nil {
		t.Fatalf("layer baseline: %v", err)
	}
	if len(bs) == 0 {
		t.Fatal("expected segments from baseline layering")
	}
	if _, err := svc.Layer(target.ID); err != nil {
		t.Fatalf("layer target: %v", err)
	}

	snap, err := svc.PublishComparison(baseline.ID, target.ID)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if snap.State != "published" {
		t.Fatalf("snapshot state = %s, want published", snap.State)
	}
}

func TestArchivedMutationRejected(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	run, err := svc.CreateWellRun("T-B-001", "r", "W-B", "2026-08-01", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	ingestBatch(t, svc, run.ID, 800.0)
	if _, err := svc.CorrectDepth(run.ID, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Layer(run.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.ArchiveRun(run.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.IngestPoints(run.ID, nil); err == nil {
		t.Fatal("expected archived mutation to be rejected")
	}
}

func TestDepthNotMonotonicRejected(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	run, err := svc.CreateWellRun("T-C-001", "r", "W-C", "2026-08-01", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	pts := []ingest.RawPoint{
		{Seq: 0, Depth: 100, Temp: 50, Pressure: 10},
		{Seq: 1, Depth: 90, Temp: 51, Pressure: 11}, // depth倒序
	}
	if err := svc.IngestPoints(run.ID, pts); err == nil {
		t.Fatal("expected depth-not-monotonic error")
	}
}
