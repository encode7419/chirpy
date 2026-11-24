package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) apiCounter(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "Hits: %d\n", cfg.fileserverHits.Load())
}

func (cfg *apiConfig) resetCounter(w http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)
}

func main() {

	apiCfg := apiConfig{}
	appHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(appHandler))
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /metrics", apiCfg.apiCounter)
	mux.HandleFunc("POST /reset", apiCfg.resetCounter)
	server := &http.Server{Addr: ":8080", Handler: mux}
	server.ListenAndServe()

}
