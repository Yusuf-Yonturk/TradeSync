package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const tokenTTL = 24 * time.Hour

type ctxKey int

const userIDKey ctxKey = 0

// HMAC imzali, durumsuz oturum token'i: base64(userID.exp).base64(imza)
type tokenSigner struct {
	secret []byte
	ttl    time.Duration
}

func newTokenSigner(secret string) *tokenSigner {
	key := []byte(secret)
	if len(key) == 0 {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			log.Fatalf("token anahtari uretilemedi: %v", err)
		}
		log.Println("AUTH_SECRET tanimli degil, gecici anahtar uretildi; yeniden baslatmada oturumlar duser")
	}
	return &tokenSigner{secret: key, ttl: tokenTTL}
}

func (s *tokenSigner) sign(payload string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *tokenSigner) Issue(userID string, now time.Time) string {
	raw := userID + "." + strconv.FormatInt(now.Add(s.ttl).Unix(), 10)
	payload := base64.RawURLEncoding.EncodeToString([]byte(raw))
	return payload + "." + s.sign(payload)
}

func (s *tokenSigner) Verify(token string, now time.Time) (string, bool) {
	payload, sig, ok := strings.Cut(token, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(s.sign(payload))) {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return "", false
	}
	i := strings.LastIndexByte(string(raw), '.')
	if i <= 0 {
		return "", false
	}
	exp, err := strconv.ParseInt(string(raw[i+1:]), 10, 64)
	if err != nil || now.Unix() >= exp {
		return "", false
	}
	return string(raw[:i]), true
}

// Authorization: Bearer <token> ister, kullanici kimligini context'e koyar.
func (s *tokenSigner) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "giris gerekli"})
			return
		}
		userID, ok := s.Verify(strings.TrimSpace(token), time.Now())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "oturum gecersiz veya suresi dolmus"})
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userIDKey, userID)))
	}
}

func userIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}
