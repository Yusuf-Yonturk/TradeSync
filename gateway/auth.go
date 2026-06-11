package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

func registerHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gecersiz json"})
			return
		}

		req.Username = strings.TrimSpace(req.Username)
		if req.Username == "" || len(req.Password) < 4 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "kullanici adi bos olamaz, sifre en az 4 karakter olmali",
			})
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "sifre hashlenemedi"})
			return
		}

		var userID string
		err = db.QueryRow(context.Background(), `
			INSERT INTO users (username, password_hash, display_name)
			VALUES ($1, $2, $3)
			RETURNING id
		`, req.Username, string(hash), req.Name).Scan(&userID)

		if err != nil {
			if strings.Contains(err.Error(), "unique") {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "kullanici adi zaten var"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "kayit hatasi: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusCreated, AuthResponse{UserID: userID, Username: req.Username, DisplayName: req.Name})
	}
}

func loginHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gecersiz json"})
			return
		}

		var userID, hash string
		var displayName *string
		err := db.QueryRow(context.Background(), `
			SELECT id, password_hash, display_name FROM users WHERE username = $1
		`, req.Username).Scan(&userID, &hash, &displayName)

		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "kullanici adi veya sifre yanlis"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "kullanici adi veya sifre yanlis"})
			return
		}

		dn := ""
		if displayName != nil {
			dn = *displayName
		}
		writeJSON(w, http.StatusOK, AuthResponse{UserID: userID, Username: req.Username, DisplayName: dn})
	}
}
