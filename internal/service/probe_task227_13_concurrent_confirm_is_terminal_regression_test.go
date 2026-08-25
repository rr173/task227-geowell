package service_test

import (
	"path/filepath"
	"sync"
	"testing"

	"task227-geowell/internal/ingest"
	"task227-geowell/internal/model"
	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

func TestTask227Bug13ConcurrentConfirmedSegmentRemainsTerminal(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug13.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	run, err := svc.CreateWellRun("CONFIRM-001", "confirm", "WELL-CONFIRM", "2026-08-25", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.IngestPoints(run.ID, []ingest.RawPoint{{Seq: 0, Depth: 0, Temp: 40, Pressure: 0}, {Seq: 1, Depth: 10, Temp: 41, Pressure: 1}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Layer(run.ID); err != nil {
		t.Fatal(err)
	}
	segs, err := st.ListSegments(run.ID)
	if err != nil || len(segs) == 0 {
		t.Fatalf("segments=%d err=%v", len(segs), err)
	}
	segID := segs[0].ID
	if err := svc.ConfirmSegment(segID, model.SegConfirmed); err != nil {
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
			if err := service.New(st).ConfirmSegment(segID, model.SegStable); err == nil {
				errs <- nil
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err == nil {
			t.Error("a concurrent request moved a confirmed segment backward")
		}
	}
	segs, err = st.ListSegments(run.ID)
	if err != nil || len(segs) != 1 || segs[0].State != model.SegConfirmed || !segs[0].Confirmed {
		t.Fatalf("final segment=%+v err=%v, want confirmed", segs, err)
	}
}
