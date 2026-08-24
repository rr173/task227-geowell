// Package httpapi exposes the REST surface for the geothermal well profile
// layering service. All routes are prefixed with /api.
package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"task227-geowell/internal/model"
	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

// Server holds the HTTP dependencies.
type Server struct {
	svc    *service.Service
	store  *store.Store
	addr   string
}

// New builds an HTTP server bound to addr.
func New(svc *service.Service, st *store.Store, addr string) *Server {
	return &Server{svc: svc, store: st, addr: addr}
}

// Handler returns the configured mux.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/wells", s.handleWells)            // GET list, POST create
	mux.HandleFunc("/api/wells/", s.handleWellByID)        // GET one, POST points, DELETE
	mux.HandleFunc("/api/wells/{id}/points", s.handlePoints)       // GET points
	mux.HandleFunc("/api/wells/{id}/correct", s.handleCorrect)     // POST depth correction
	mux.HandleFunc("/api/wells/{id}/layer", s.handleLayer)         // POST layering
	mux.HandleFunc("/api/wells/{id}/segments", s.handleSegments)   // GET segments
	mux.HandleFunc("/api/wells/{id}/report", s.handleReport)        // GET text report
	mux.HandleFunc("/api/wells/{id}/metrics", s.handleMetrics)      // GET metrics
	mux.HandleFunc("/api/wells/{id}/export", s.handleExport)        // GET csv export
	mux.HandleFunc("/api/wells/{id}/calibrate", s.handleCalibrate)  // POST calibration plan
	mux.HandleFunc("/api/segments", s.handleAllSegments)            // GET all segments
	mux.HandleFunc("/api/segments/{id}/confirm", s.handleConfirmSegment) // POST confirm
	mux.HandleFunc("/api/compare", s.handleCompare)         // POST cross-run compare
	mux.HandleFunc("/api/snapshots", s.handleSnapshots)     // GET list
	mux.HandleFunc("/api/snapshots/", s.handleSnapshotByID) // GET one, POST publish
	mux.HandleFunc("/api/stats", s.handleStats)             // GET statistics
	return mux
}

// ListenAndServe starts the long-running server.
func (s *Server) ListenAndServe() error {
	return http.ListenAndServe(s.addr, s.Handler())
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func parseID(path string) (int64, error) {
	// path like /api/wells/12/points -> extract first numeric segment
	for i := 0; i < len(path); i++ {
		if path[i] >= '0' && path[i] <= '9' {
			// find the contiguous digit run starting at i
			j := i
			for j < len(path) && path[j] >= '0' && path[j] <= '9' {
				j++
			}
			n, err := strconv.ParseInt(strings.TrimSpace(path[i:j]), 10, 64)
			if err != nil {
				return 0, errBadID
			}
			return n, nil
		}
	}
	return 0, errBadID
}

var errBadID = &model.StateError{From: "id", To: "numeric"}
