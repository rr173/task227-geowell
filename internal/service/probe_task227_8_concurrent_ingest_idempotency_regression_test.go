package service_test

import (
	"path/filepath"
	"sync"
	"testing"

	"task227-geowell/internal/ingest"
	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

func TestTask227Bug08ConcurrentIdenticalIngestsRemainIdempotent(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug08.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	run, err := svc.CreateWellRun("CONC-001", "concurrent", "WELL-CONC", "2026-08-25", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	batch := []ingest.RawPoint{{Seq: 0, Depth: 10, Temp: 40, Pressure: 1}}
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := svc.IngestPoints(run.ID, batch); err != nil {
				errs <- err
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent identical ingest failed: %v", err)
	}
	points, err := st.ListPoints(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 {
		t.Fatalf("stored points=%d, want exactly one", len(points))
	}
}
