package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	c := New("http://localhost:3548", "")
	if c.BaseURL != "http://localhost:3548" || c.APIKey != "" {
		t.Errorf("New: %+v", c)
	}
	c2 := New("http://localhost:3548", "secret")
	if c2.APIKey != "secret" {
		t.Errorf("New with key: %+v", c2)
	}
}

func TestHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	ctx := context.Background()
	ok, err := c.Health(ctx)
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if !ok {
		t.Fatal("Health: expected ok true")
	}
}

func TestHealth_error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"down"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	ctx := context.Background()
	_, err := c.Health(ctx)
	if err == nil {
		t.Fatal("expected error from 503")
	}
}

func TestClient_setsAPIKeyHeader(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-API-Key")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "mykey")
	ctx := context.Background()
	_, _ = c.Health(ctx)
	if gotKey != "mykey" {
		t.Errorf("X-API-Key: got %q", gotKey)
	}
}
