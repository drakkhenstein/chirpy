package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"encoding/json"
	"time"
	"github.com/google/uuid"

	"github.com/drakkhenstein/chirpy/internal/database"
	"github.com/drakkhenstein/chirpy/internal/auth"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db 		   *database.Queries
	platform string
	jwtSecret string
	polkaKey string
}

type User struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email string `json:"email"`
	IsChirpyRed bool `json:"is_chirpy_red"`
}

type Chirp struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID uuid.UUID `json:"user_id"`
	Body string `json:"body"`
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %s", err)
	}
	defer db.Close()
	dbQueries := database.New(db)

	//reading the PLATFORM .env into apicfg for use in the handler functions
	platform := os.Getenv("PLATFORM")

	// server hits counter
	apiCfg := &apiConfig{
		fileserverHits: atomic.Int32{},
		db:             dbQueries,
		platform:       platform,
		jwtSecret:      os.Getenv("JWT_SECRET"),
		polkaKey:       os.Getenv("POLKA_KEY"),
	}
	if apiCfg.jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	if apiCfg.polkaKey == "" {
		log.Fatal("POLKA_KEY environment variable is required")
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
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics) 
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerChirpsRetrieve)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirpByID)
	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefreshToken)
	mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevokeRefreshToken)
	mux.HandleFunc("PUT /api/users", apiCfg.handlerUpdateUser)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handlerDeleteChirp)
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.handlerWebhook)
	

	

	//use listen and serve to start the server
	log.Printf("Serving files %s on port %s\n", ".", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	// build the html string with the hits count
	html := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load())
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	// write the html string to the response
	w.Write([]byte(html))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// handlerCreateChirp is a handler function that creates a new chirp
func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authorization header missing", err)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
		return
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}
	if params.Body == "" {
		respondWithError(w, http.StatusBadRequest, "Body is required", nil)
		return
	}
	cleanedBody, err := validateChirp(params.Body)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	chirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body: cleanedBody,
		UserID: userID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating chirp", err)
		return
	}
	respondWithJSON(w, http.StatusCreated, Chirp{
		ID: chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body: chirp.Body,
		UserID: chirp.UserID,
	})
}

func validateChirp(body string) (string, error) {
	const maxChirpLength = 140
	if len(body) > maxChirpLength {
		return "", fmt.Errorf("chirp is too long")
	}
	return replaceBadWords(body), nil
}

// handlerGetChirps is a handler function that retrieves all chirps
func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	dbChirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error retrieving chirps", err)
		return
	}
	respChirps := []Chirp{}
	for _, dbChirp := range dbChirps {
		respChirps = append(respChirps, Chirp{
			ID: dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			UserID: dbChirp.UserID,
			Body: dbChirp.Body,
		})
	}
	respondWithJSON(w, http.StatusOK, respChirps)
}

func authorIDFromRequest(r *http.Request) (uuid.UUID, error) {
	authorIDString := r.URL.Query().Get("author_id")
	if authorIDString == "" {
		return uuid.Nil, nil
	}
	authorID, err := uuid.Parse(authorIDString)
	if err != nil {
		return uuid.Nil, err
	}
	return authorID, nil
}

func (cfg *apiConfig) handlerChirpsRetrieve(w http.ResponseWriter, r *http.Request) {
	authorID, err := authorIDFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid author ID", err)
		return
	}

	var dbChirps []database.Chirp

	if authorID != uuid.Nil {
		dbChirps, err = cfg.db.GetChirpsByAuthor(r.Context(), authorID)
	} else {
		dbChirps, err = cfg.db.GetChirps(r.Context())
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve chirps", err)
		return
	}
	respChirps := []Chirp{}
	for _, dbChirp := range dbChirps {
		respChirps = append(respChirps, Chirp{
			ID: dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			UserID: dbChirp.UserID,
			Body: dbChirp.Body,
		})
	}
	respondWithJSON(w, http.StatusOK, respChirps)
}

// handlerGetChirpByID is a handler function that retrieves a chirp by its ID
func (cfg *apiConfig) handlerGetChirpByID(w http.ResponseWriter, r *http.Request) {
	chirpIDStr := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}
	dbChirp, err := cfg.db.GetChirpByID(r.Context(), chirpID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Chirp not found", nil)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Error retrieving chirp", err)
		return
	}
	respChirp := Chirp{
		ID: dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		UserID: dbChirp.UserID,
		Body: dbChirp.Body,
	}
	respondWithJSON(w, http.StatusOK, respChirp)
}

//a handler function that creates a new user
func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}
	if params.Email == "" {
		respondWithError(w, http.StatusBadRequest, "Email is required", nil)
		return
	}
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		w.WriteHeader(500)
		return
	}
	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email: params.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		log.Printf("Error creating user: %s", err)
		w.WriteHeader(500)
		return
	}
	
	respUser := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}
	respondWithJSON(w, http.StatusCreated, respUser)
}

func (cfg *apiConfig) handlerRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
    	respondWithError(w, http.StatusUnauthorized, "Missing or invalid token", err)
    	return
	}
	userID, err := cfg.db.GetUserFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid refresh token", nil)
		return
	}
	token, err := auth.MakeJWT(userID, cfg.jwtSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating JWT", err)
		return
	}

	type response struct {
		Token string `json:"token"`
	}

	respondWithJSON(w, http.StatusOK, response{
		Token: token,
	})
}

func (cfg *apiConfig) handlerRevokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing or invalid token", err)
		return
	}
	err = cfg.db.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error revoking refresh token", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authorization header missing", err)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
		return
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(401)
		return
	}
	if params.Email == "" {
		respondWithError(w, http.StatusBadRequest, "Email is required", nil)
		return
	}
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		w.WriteHeader(500)
		return
	}
	user, err := cfg.db.UpdateUser(r.Context(), database.UpdateUserParams{
		ID: userID,
		Email: params.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		log.Printf("Error updating user: %s", err)
		w.WriteHeader(500)
		return
	}
	respUser := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}
	respondWithJSON(w, http.StatusOK, respUser)
}

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Unauthorized User: %s", err)
		w.WriteHeader(401)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		log.Printf("Unauthorized User: %s", err)
		w.WriteHeader(401)
		return
	}
	chirpIDStr := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}
	dbChirp, err := cfg.db.GetChirpByID(r.Context(), chirpID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Chirp not found", nil)
			return
		}
		log.Printf("Chirp not found: %s", err)
		w.WriteHeader(404)
		return
	}
	if dbChirp.UserID != userID {
		w.WriteHeader(403)
		return
	}
	err = cfg.db.DeleteChirps(r.Context(), chirpID)
	if err != nil {
		log.Printf("Error deleting chirp: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(204)
}

// handlerWebhook is a handler function that handles webhooks from Polka and upgrades users to Chirpy Red when they upgrade on Polka
func (cfg *apiConfig) handlerWebhook(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Event string `json:"event"`
		Data struct {
			UserID uuid.UUID `json:"user_id"`
		}
	}

	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	if apiKey != cfg.polkaKey {
		w.WriteHeader(401)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}
	
	if params.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	_, err = cfg.db.UpgradeToChirpyRed(r.Context(), params.Data.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Couldn't find user", nil)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't update user", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
