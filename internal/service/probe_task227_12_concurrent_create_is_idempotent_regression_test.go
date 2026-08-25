package service_test

import (
	"path/filepath"
	"sync"
	"testing"

	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

func TestTask227Bug12ConcurrentIdenticalCreatesRemainIdempotent(t *testing.T) {
	db := filepath.Join(t.TempDir(), "bug12.db")
	st, err := store.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			svc := service.New(st)
			if _, err := svc.CreateWellRun("CREATE-001", "same", "WELL-CREATE", "2026-08-25", "C", "MPa"); err != nil {
				errs <- err
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("identical concurrent create failed: %v", err)
	}
	runs, err := st.ListWellRuns("")
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("stored runs=%d, want exactly one", len(runs))
	}
}
