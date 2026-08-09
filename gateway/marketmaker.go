package main

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Market maker iki hesapla calisir: biri yalniz alir, digeri yalniz satar.
// Tek hesap olsaydi kendi emriyle eslesme engeli butun islemlerini durdururdu.
func marketMakerUsernames(base string) []string {
	return []string{base, base + "-sell"}
}

// Market maker hesaplarini gateway kendisi acar ve bakiye kontrolunden muaf tutar.
// Hesaplar acilistan once var oldugu icin bu kullanici adlarini baskasi kaydedemez.
func provisionMarketMakers(ctx context.Context, db *pgxpool.Pool, base, password string) (map[string]bool, error) {
	exempt := make(map[string]bool)
	if password == "" {
		return exempt, nil
	}
	if msg := validateRegister(RegisterRequest{Username: base + "-sell", Password: password}); msg != "" {
		return nil, errors.New("market maker hesabi: " + msg)
	}

	for _, username := range marketMakerUsernames(base) {
		var id, hash string
		err := db.QueryRow(ctx, `SELECT id, password_hash FROM users WHERE username = $1`, username).Scan(&id, &hash)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			newHash, herr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if herr != nil {
				return nil, herr
			}
			err = db.QueryRow(ctx, `
				INSERT INTO users (username, password_hash, display_name) VALUES ($1, $2, 'Market Maker')
				ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash
				RETURNING id
			`, username, string(newHash)).Scan(&id)
			if err != nil {
				return nil, err
			}
		case err != nil:
			return nil, err
		case bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil:
			newHash, herr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if herr != nil {
				return nil, herr
			}
			if _, err := db.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id::text = $1`, id, string(newHash)); err != nil {
				return nil, err
			}
		}
		exempt[id] = true
	}
	return exempt, nil
}
