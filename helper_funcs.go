package main

import (
	"strings"
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

// responds with clean body code from bad words and a success message
func respondWithJSON(w http.ResponseWriter, statusCode int, payload map[string]string) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to serialize JSON", nil)
		return
	}
	w.Write(jsonBytes)
}

// bad word list replacement function
func replaceBadWords(message string) string {
	// Implementation for replacing bad words
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	// splits the message string down by words
	words := strings.Split(message, " ")
	for i, word := range words {
		for _, badWord := range badWords {
			if strings.ToLower(word) == badWord {
				words[i] = "****"
			}
		}
	}
	return strings.Join(words, " ")
}