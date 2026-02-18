package ui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler(t *testing.T) {
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: status=%d", rec.Code)
	}
	if rec.Header().Get("Content-Type") == "" {
		t.Log("Content-Type may be set by FileServer")
	}
	if rec.Body.Len() == 0 {
		t.Fatal("GET /: empty body")
	}
}

func TestHandler_spaFallback(t *testing.T) {
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/teams/foo", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	// Unknown path should fall back to index.html for SPA routing
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /teams/foo (fallback): status=%d", rec.Code)
	}
}
