package main

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
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
	Token       string `json:"token"`
}

const (
	minPasswordLen = 8
	// bcrypt 72 bayttan sonrasini islemez
	maxPasswordLen = 72
	maxNameLen     = 64
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,32}$`)

func validateRegister(req RegisterRequest) string {
	if !usernameRe.MatchString(req.Username) {
		return "kullanici adi 3-32 karakter olmali (harf, rakam, . _ -)"
	}
	if len(req.Password) < minPasswordLen || len(req.Password) > maxPasswordLen {
		return "sifre 8-72 karakter olmali"
	}
	if utf8.RuneCountInString(req.Name) > maxNameLen {
		return "isim en fazla 64 karakter olabilir"
	}
	return ""
}

func registerHandler(db *pgxpool.Pool, signer *tokenSigner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gecersiz json"})
			return
		}

		req.Username = strings.TrimSpace(req.Username)
		req.Name = strings.TrimSpace(req.Name)
		if msg := validateRegister(req); msg != "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "sifre hashlenemedi"})
			return
		}

		var userID string
		err = db.QueryRow(r.Context(), `
			INSERT INTO users (username, password_hash, display_name)
			VALUES ($1, $2, $3)
			RETURNING id
		`, req.Username, string(hash), req.Name).Scan(&userID)

		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "kullanici adi zaten var"})
				return
			}
			log.Printf("kayit hatasi: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "kayit hatasi"})
			return
		}

		writeJSON(w, http.StatusCreated, AuthResponse{
			UserID:      userID,
			Username:    req.Username,
			DisplayName: req.Name,
			Token:       signer.Issue(userID, time.Now()),
		})
	}
}

func loginHandler(db *pgxpool.Pool, signer *tokenSigner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gecersiz json"})
			return
		}

		var userID, hash string
		var displayName *string
		req.Username = strings.TrimSpace(req.Username)
		if req.Username == "" || len(req.Password) == 0 || len(req.Password) > maxPasswordLen {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "kullanici adi veya sifre yanlis"})
			return
		}

		err := db.QueryRow(r.Context(), `
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
		writeJSON(w, http.StatusOK, AuthResponse{
			UserID:      userID,
			Username:    req.Username,
			DisplayName: dn,
			Token:       signer.Issue(userID, time.Now()),
		})
	}
}
