package httpapi

import (
	"net/http"
	"strconv"

	"task227-geowell/internal/calibrate"
	"task227-geowell/internal/export"
	"task227-geowell/internal/metrics"
	"task227-geowell/internal/model"
	"task227-geowell/internal/report"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "geowell"})
}

func (s *Server) handleWells(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		well := r.URL.Query().Get("well")
		runs, err := s.store.ListWellRuns(well)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, runs)
	case http.MethodPost:
		var body struct {
			Code      string `json:"code"`
			Name      string `json:"name"`
			Well      string `json:"well"`
			LogDate   string `json:"log_date"`
			UnitTemp  string `json:"unit_temp"`
			UnitPress string `json:"unit_press"`
		}
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		run, err := s.svc.CreateWellRun(body.Code, body.Name, body.Well, body.LogDate, body.UnitTemp, body.UnitPress)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, run)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleWellByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid well id"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		run, err := s.store.GetWellRun(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, run)
	case http.MethodDelete:
		run, err := s.store.GetWellRun(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		if run.Archived {
			writeJSON(w, http.StatusConflict, map[string]string{"error": model.ErrArchivedMutation.Error()})
			return
		}
		// soft delete: archive the run instead of dropping data
		if err := s.svc.ArchiveRun(id); err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "archived"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handlePoints(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid well id"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	pts, err := s.store.ListPoints(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, pts)
}

func (s *Server) handleCorrect(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid well id"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var body struct {
		ReferenceDepth float64 `json:"reference_depth"`
	}
	_ = readJSON(r, &body)
	c, err := s.svc.CorrectDepth(id, body.ReferenceDepth)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleLayer(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid well id"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	segs, err := s.svc.Layer(id)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"segments": segs, "count": len(segs)})
}

func (s *Server) handleSegments(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid well id"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	segs, err := s.store.ListSegments(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, segs)
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid well id"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	run, err := s.store.GetWellRun(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	segs, err := s.store.ListSegments(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	prof := report.BuildProfile(run, segs)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(prof.Text()))
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid well id"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	pts, err := s.store.ListPoints(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	segs, err := s.store.ListSegments(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	m := metrics.Compute(pts, segs)
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid well id"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	segs, err := s.store.ListSegments(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(export.SegmentsCSV(segs)))
}

func (s *Server) handleCalibrate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid well id"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var body struct {
		Strategy string `json:"strategy"`
		Markers  []struct {
			Raw float64 `json:"raw"`
			Cal float64 `json:"cal"`
		} `json:"markers"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	markers := make([]calibrate.Marker, 0, len(body.Markers))
	for _, m := range body.Markers {
		markers = append(markers, calibrate.Marker{Raw: m.Raw, Cal: m.Cal})
	}
	plan, err := s.svc.CorrectDepthPlan(id, calibrate.Strategy(body.Strategy), markers)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) handleAllSegments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	// List segments for a specific run when ?run_id is given, else all runs.
	runIDStr := r.URL.Query().Get("run_id")
	if runIDStr != "" {
		runID, err := strconv.ParseInt(runIDStr, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid run_id"})
			return
		}
		segs, err := s.store.ListSegments(runID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, segs)
		return
	}
	runs, err := s.store.ListWellRuns("")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var all []model.Segment
	for _, run := range runs {
		segs, err := s.store.ListSegments(run.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		all = append(all, segs...)
	}
	writeJSON(w, http.StatusOK, all)
}

func (s *Server) handleConfirmSegment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid segment id"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var body struct {
		State string `json:"state"`
	}
	_ = readJSON(r, &body)
	st := model.SegmentState(body.State)
	// Only candidate/stable/anomaly may be confirmed; other states are illegal.
	valid := st == model.SegStable || st == model.SegAnomaly || st == model.SegConfirmed
	if !valid {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": model.ErrBadTransition.Error()})
		return
	}
	if err := s.svc.ConfirmSegment(id, st); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "segment_id": strconv.FormatInt(id, 10)})
}

func (s *Server) handleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var body struct {
		BaselineID int64 `json:"baseline_id"`
		TargetID   int64 `json:"target_id"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	moves, err := s.svc.CompareRuns(body.BaselineID, body.TargetID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"movements": moves, "count": len(moves)})
}

func (s *Server) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	well := r.URL.Query().Get("well")
	if well == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "well query required"})
		return
	}
	snaps, err := s.store.ListSnapshotsByWell(well)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snaps)
}

func (s *Server) handleSnapshotByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid snapshot id"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		snap, err := s.store.GetSnapshot(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, snap)
	case http.MethodPost:
		// publish a comparison (baseline/target provided in body)
		var body struct {
			BaselineID int64 `json:"baseline_id"`
			TargetID   int64 `json:"target_id"`
		}
		_ = readJSON(r, &body)
		snap, err := s.svc.PublishComparison(body.BaselineID, body.TargetID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, snap)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	runs, _ := s.store.ListWellRuns("")
	byState := map[string]int{}
	for _, run := range runs {
		byState[string(run.State)]++
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"well_runs": len(runs),
		"by_state":  byState,
		"api_version": "1.0",
	})
}
