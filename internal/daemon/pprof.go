package daemon

import (
	"log/slog"
	"net/http"

	_ "net/http/pprof"
)

func startPprof(addr string) {
	if addr == "" {
		return
	}
	go func() {
		// Uses DefaultServeMux, which has pprof handlers registered via blank import.
		if err := http.ListenAndServe(addr, nil); err != nil {
			slog.Info("pprof server stopped", "addr", addr, "err", err)
		}
	}()
}
