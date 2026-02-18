package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ankittk/agentary/internal/otel"
)

type SSEHub struct {
	mu   sync.RWMutex
	subs map[chan []byte]struct{}
}

func NewSSEHub() *SSEHub {
	return &SSEHub{subs: make(map[chan []byte]struct{})}
}

func (h *SSEHub) Subscribe() chan []byte {
	ch := make(chan []byte, 256)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	otel.AddSSEConnection()
	return ch
}

func (h *SSEHub) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
		otel.RemoveSSEConnection()
	}
	h.mu.Unlock()
}

func (h *SSEHub) PublishJSON(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	otel.RecordSSEEvent(context.Background())
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs {
		select {
		case ch <- b:
		default:
			// Drop if subscriber is too slow; prevents global backpressure.
		}
	}
}

func (h *SSEHub) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		ch := h.Subscribe()
		defer h.Unsubscribe(ch)

		// Initial ping so clients know the stream is live.
		_, _ = fmt.Fprintf(w, "data: %s\n\n", `{"type":"connected"}`)
		flusher.Flush()

		keepalive := time.NewTicker(30 * time.Second)
		defer keepalive.Stop()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case <-keepalive.C:
				// Comment keepalive.
				_, _ = fmt.Fprint(w, ": keepalive\n\n")
				flusher.Flush()
			case msg, ok := <-ch:
				if !ok {
					return
				}
				_, _ = fmt.Fprintf(w, "data: %s\n\n", string(msg))
				flusher.Flush()
			}
		}
	}
}
