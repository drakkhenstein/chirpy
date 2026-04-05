package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	// server hits counter
	apiCfg := &apiConfig{
		fileserverHits: atomic.Int32{},
	}

	// create a new http.ServeMux
	mux := http.NewServeMux()

	// create a new http.Server struct
	srv := &http.Server{
		Addr: ":8080",
		Handler:mux,
	}

	// register the handler for the root path
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("/healthz", handlerReadiness)
	mux.HandleFunc("/metrics", apiCfg.handlerMetrics) 
	mux.HandleFunc("/reset", apiCfg.handlerReset)

	//use listen and serve to start the server
	log.Printf("Serving files %s on port %s\n", ".", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}