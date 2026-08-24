package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task227-geowell/internal/ingest"
	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

func TestProfileInspectionEndpointsExposePersistedState(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "httpapi.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	run, err := svc.CreateWellRun("HTTP-001", "http", "WELL-HTTP", "2026-08-25", "C", "MPa")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.IngestPoints(run.ID, []ingest.RawPoint{
		{Seq: 0, Depth: 0, Temp: 40, Pressure: 0},
		{Seq: 1, Depth: 10, Temp: 41, Pressure: 1},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CorrectDepth(run.ID, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Layer(run.ID); err != nil {
		t.Fatal(err)
	}

	server := New(svc, st, "")
	for _, path := range []string{
		"/api/wells/1/data-quality",
		"/api/wells/1/datum",
		"/api/wells/1/boundaries",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		resp := httptest.NewRecorder()
		server.Handler().ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200: %s", path, resp.Code, resp.Body.String())
		}
		var body map[string]interface{}
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("GET %s returned invalid JSON: %v", path, err)
		}
		if len(body) == 0 {
			t.Fatalf("GET %s returned empty JSON object", path)
		}
	}
}
