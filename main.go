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
	mux.Handle("/", http.FileServer(http.Dir(".")))

	//use listen and serve to start the server
	log.Fatal(srv.ListenAndServe())
}