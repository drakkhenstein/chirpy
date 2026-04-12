package main

import (
	"encoding/json"
	"net/http"
)

func respondWithError(w http.ResponseWriter, statusCode int, message string, err error) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	jsonBytes, err := json.Marshal(map[string]string{"error": message})
	if err != nil {
		w.Write([]byte(`{"error": "Failed to serialize JSON"}`))
		return
	}
	w.Write(jsonBytes)
	return
}

func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to serialize JSON", nil)
		return
	}
	w.Write(jsonBytes)
}