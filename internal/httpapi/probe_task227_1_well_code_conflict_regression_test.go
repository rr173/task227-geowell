package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"task227-geowell/internal/service"
	"task227-geowell/internal/store"
)

func TestTask227Bug01WellCodeConflictKeepsOriginalRun(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "bug01.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	srv := New(service.New(st), st, "")
	create := func(code, well, temp, press string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/wells", strings.NewReader(`{"code":"`+code+`","name":"run","well":"`+well+`","log_date":"2026-08-25","unit_temp":"`+temp+`","unit_press":"`+press+`"}`))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		srv.Handler().ServeHTTP(resp, req)
		return resp
	}
	first := create("DUP-001", "WELL-A", "C", "MPa")
	if first.Code != http.StatusCreated {
		t.Fatalf("first create status=%d body=%s", first.Code, first.Body.String())
	}
	second := create("DUP-001", "WELL-B", "K", "bar")
	if second.Code != http.StatusConflict {
		t.Fatalf("conflicting create status=%d, want 409; body=%s", second.Code, second.Body.String())
	}
	var runs []map[string]interface{}
	req := httptest.NewRequest(http.MethodGet, "/api/wells", nil)
	resp := httptest.NewRecorder()
	srv.Handler().ServeHTTP(resp, req)
	if err := json.Unmarshal(resp.Body.Bytes(), &runs); err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0]["well"] != "WELL-A" {
		t.Fatalf("stored runs=%v, want only original WELL-A", runs)
	}
}
