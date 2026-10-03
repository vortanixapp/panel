package httpprom

import (
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"
	"time"
)

func StartPprof() {
	addr := strings.TrimSpace(os.Getenv("PPROF_ADDR"))
	if addr == "" {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("pprof listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("pprof: %v", err)
		}
	}()
}
