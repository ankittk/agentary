package httpapi

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSSEHub_Subscribe_Publish_Unsubscribe(t *testing.T) {
	hub := NewSSEHub()
	ch := hub.Subscribe()
	hub.PublishJSON(map[string]string{"type": "test"})
	msg := <-ch
	if !strings.Contains(string(msg), "test") {
		t.Errorf("PublishJSON: got %s", msg)
	}
	hub.Unsubscribe(ch)
	// After unsubscribe, channel is closed
	_, ok := <-ch
	if ok {
		t.Error("expected channel closed after Unsubscribe")
	}
}

func TestSSEHub_Handler(t *testing.T) {
	hub := NewSSEHub()
	handler := hub.Handler()
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/stream", nil)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		handler(rec, req)
		close(done)
	}()
	// Wait for handler to send "connected" then stop (avoid reading rec.Body while handler writes - race).
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done
	// Read response body only after handler has finished writing.
	sc := bufio.NewScanner(rec.Body)
	var found bool
	for sc.Scan() {
		if strings.Contains(sc.Text(), "connected") {
			found = true
			break
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !found {
		t.Error("expected response to contain \"connected\"")
	}
}
