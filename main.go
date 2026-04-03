package main

import (
	"log"
	"net/http"
)

func main() {
	// create a new http.ServeMux
	mux := http.NewServeMux()

	// create a new http.Server struct
	srv := &http.Server{
		Addr: ":8080",
		Handler:mux,
	}

	// register the handler for the root path
	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	mux.HandleFunc("/healthz", handlerReadiness)

	//use listen and serve to start the server
	log.Printf("Serving files %s on port %s\n", ".", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}