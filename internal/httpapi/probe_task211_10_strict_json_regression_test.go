package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"

	"task211-quenchprop/internal/service"
	"task211-quenchprop/internal/store"
)

func TestBug10_HTTPRejectsTrailingJSON(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/http.db")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	h := NewRouter(service.NewApp(db, service.DefaultConfig()))
	req := httptest.NewRequest("POST", "/api/topologies", strings.NewReader(`{"id":"one"}{"id":"two"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("trailing JSON status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}
