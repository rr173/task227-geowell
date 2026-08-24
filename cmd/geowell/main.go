// Command geowell is the entry point for the geothermal well temperature/pressure
// profile anomaly layering service. It supports a long-running HTTP server and a
// --smoke-test mode that exercises the full data loop and verifies persistence
// and restart recovery.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"task227-geowell/internal/httpapi"
	"task227-geowell/internal/ingest"
	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "geowell.db", "SQLite database path")
	smoke := flag.Bool("smoke-test", false, "run self-test loop and exit")
	flag.Parse()

	if *smoke {
		if err := runSmokeTest(*dbPath); err != nil {
			log.Fatalf("smoke-test failed: %v", err)
		}
		fmt.Println("smoke-test: OK")
		os.Exit(0)
	}

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()
	svc := service.New(st)
	srv := httpapi.New(svc, st, *addr)
	log.Printf("geowell listening on %s (db=%s)", *addr, *dbPath)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// runSmokeTest builds a well run, ingests points, corrects depth, layers,
// compares two runs and publishes a snapshot, then closes and reopens the
// database to prove persistence and restart recovery.
func runSmokeTest(dbPath string) error {
	_ = os.Remove(dbPath)
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")

	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	svc := service.New(st)

	// 1. create two well runs on the same well.
	baseline, err := svc.CreateWellRun("WELL-A-2026-001", "Geothermal profile A", "WELL-A", "2026-08-01", "C", "MPa")
	if err != nil {
		return fmt.Errorf("create baseline: %w", err)
	}
	target, err := svc.CreateWellRun("WELL-A-2026-002", "Geothermal profile B", "WELL-A", "2026-08-15", "C", "MPa")
	if err != nil {
		return fmt.Errorf("create target: %w", err)
	}

	// 2. ingest points with a gradient boundary near 800 m for both runs.
	if err := ingestBatch(svc, baseline.ID, 800.0, false); err != nil {
		return fmt.Errorf("ingest baseline: %w", err)
	}
	if err := ingestBatch(svc, target.ID, 825.0, false); err != nil {
		return fmt.Errorf("ingest target: %w", err)
	}

	// 3. correct depth (reference shoe at 0 m -> offset 0).
	if _, err := svc.CorrectDepth(baseline.ID, 0); err != nil {
		return fmt.Errorf("correct baseline: %w", err)
	}
	if _, err := svc.CorrectDepth(target.ID, 0); err != nil {
		return fmt.Errorf("correct target: %w", err)
	}

	// 4. layer both runs.
	if _, err := svc.Layer(baseline.ID); err != nil {
		return fmt.Errorf("layer baseline: %w", err)
	}
	if _, err := svc.Layer(target.ID); err != nil {
		return fmt.Errorf("layer target: %w", err)
	}

	// 5. compare and publish a snapshot.
	snap, err := svc.PublishComparison(baseline.ID, target.ID)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	if snap.State != "published" {
		return fmt.Errorf("snapshot not published: state=%s", snap.State)
	}

	// 6. close and reopen to verify persistence / restart recovery.
	if err := st.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	st2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("reopen: %w", err)
	}
	defer st2.Close()
	svc2 := service.New(st2)
	runs, err := st2.ListWellRuns("WELL-A")
	if err != nil {
		return fmt.Errorf("list after reopen: %w", err)
	}
	if len(runs) != 2 {
		return fmt.Errorf("expected 2 runs after reopen, got %d", len(runs))
	}
	segs, err := st2.ListSegments(baseline.ID)
	if err != nil {
		return fmt.Errorf("segments after reopen: %w", err)
	}
	if len(segs) == 0 {
		return fmt.Errorf("no segments persisted after reopen")
	}
	// re-run compare to prove the loop is reproducible after restart.
	if _, err := svc2.CompareRuns(baseline.ID, target.ID); err != nil {
		return fmt.Errorf("compare after reopen: %w", err)
	}
	return nil
}

// ingestBatch creates a synthetic profile with a steep gradient change near
// boundaryDepth to exercise boundary detection.
func ingestBatch(svc *service.Service, runID int64, boundaryDepth float64, disturb bool) error {
	var pts []ingest.RawPoint
	depth := 0.0
	seq := 0
	temp := 40.0
	for depth <= 1500.0 {
		p := ingest.RawPoint{
			Seq:      seq,
			Depth:    depth,
			Temp:     temp,
			Pressure: 0.1 * depth,
		}
		if disturb && depth < 50 {
			p.Missing = true // simulate a shallow wellhead gap
		}
		pts = append(pts, p)
		seq++
		if depth < boundaryDepth {
			temp += 0.02 * 5.0 // gentle gradient
		} else {
			temp += 0.20 * 5.0 // steep gradient after boundary
		}
		depth += 5.0
	}
	return svc.IngestPoints(runID, pts)
}
