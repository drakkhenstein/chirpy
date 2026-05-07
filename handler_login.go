package main

import (
	"time"
	"encoding/json"
	"net/http"
	"github.com/drakkhenstein/chirpy/internal/auth"
	"github.com/drakkhenstein/chirpy/internal/database"
)

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}
	if params.Email == "" {
		respondWithError(w, http.StatusBadRequest, "Email is required", nil)
		return
	}
	if params.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Password is required", nil)
		return
	}
	user, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", nil)
		return
	}
	match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil || !match {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", nil)
		return
	}
	
	token, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating JWT", err)
		return
	}
	type response struct {
		User
		Token string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating refresh token", err)
		return
	}
	err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token: refreshToken,
		UserID: user.ID,
		ExpiresAt: time.Now().Add(60 * 24 * time.Hour), // Refresh token valid for 60 days
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error saving refresh token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
	User : User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		IsChirpyRed: user.IsChirpyRed,
	},
	Token: token,
	RefreshToken: refreshToken,
	})
}