package service

import (
	"path/filepath"
	"sync"
	"testing"

	"task227-geowell/internal/ingest"
	"task227-geowell/internal/store"
)

func TestTask227Bug09ConcurrentSameComparisonPublishesOnce(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug09.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st)
	base, err := svc.CreateWellRun("PUB-BASE", "base", "WELL-PUB", "2026-08-01", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	target, err := svc.CreateWellRun("PUB-TARGET", "target", "WELL-PUB", "2026-08-15", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	ingestForPublishProbe(t, svc, base.ID, 800)
	ingestForPublishProbe(t, svc, target.ID, 820)
	if _, err := svc.CorrectDepth(base.ID, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CorrectDepth(target.ID, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Layer(base.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Layer(target.ID); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := svc.PublishComparison(base.ID, target.ID); err != nil {
				errs <- err
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent publish failed: %v", err)
	}
	snaps, err := st.ListSnapshotsByWell("WELL-PUB")
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 1 || snaps[0].State != "published" {
		t.Fatalf("snapshots=%+v, want one published snapshot", snaps)
	}
}

func ingestForPublishProbe(t *testing.T, svc *Service, runID int64, boundary float64) {
	t.Helper()
	var pts []ingest.RawPoint
	depth, temp, seq := 0.0, 40.0, 0
	for depth <= 1500 {
		pts = append(pts, ingest.RawPoint{Seq: seq, Depth: depth, Temp: temp, Pressure: 0.1 * depth})
		seq++
		if depth < boundary {
			temp += 0.1
		} else {
			temp += 1.0
		}
		depth += 5
	}
	if err := svc.IngestPoints(runID, pts); err != nil {
		t.Fatal(err)
	}
}
