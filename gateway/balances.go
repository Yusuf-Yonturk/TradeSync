package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tradesync/shared/money"
	"tradesync/shared/schema"
)

// Gercek para yatirma olmadigi icin her hesaba bir kez verilen deneme bakiyesi.
const defaultDemoFunds = "USD:100000,BTC:10,ETH:100,SOL:500"

var assetRe = regexp.MustCompile(`^[A-Z0-9]{2,10}$`)

var errInsufficient = errors.New("yetersiz bakiye")

type demoFund struct {
	asset string
	units int64
}

// "USD:100000,BTC:10" bicimini cozer.
func parseDemoFunds(s string) ([]demoFund, error) {
	var funds []demoFund
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		asset, raw, ok := strings.Cut(part, ":")
		if !ok || !assetRe.MatchString(asset) {
			return nil, fmt.Errorf("gecersiz bakiye tanimi: %q", part)
		}
		units, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || units <= 0 || units > 1e12 {
			return nil, fmt.Errorf("gecersiz tutar: %q", part)
		}
		funds = append(funds, demoFund{asset: asset, units: units})
	}
	return funds, nil
}

type ledger struct {
	db    *pgxpool.Pool
	funds []demoFund
	// bakiye kontrolunden muaf hesaplar (market maker)
	exempt map[string]bool
}

func applySchema(ctx context.Context, db *pgxpool.Pool) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", schema.LockKey); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, schema.SQL); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Deneme bakiyesini hesaba bir kez yukler. Yukleme bu cagrida yapildiysa true doner.
func (l *ledger) ensureFunds(ctx context.Context, userID string) (bool, error) {
	tx, err := l.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `UPDATE users SET funded_at = NOW() WHERE id::text = $1 AND funded_at IS NULL`, userID)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	for _, f := range l.funds {
		if _, err := tx.Exec(ctx, `
			INSERT INTO balances (user_id, asset, available) VALUES ($1, $2, $3::text::numeric)
			ON CONFLICT (user_id, asset) DO UPDATE SET available = balances.available + EXCLUDED.available
		`, userID, f.asset, money.FromUnits(f.units).String()); err != nil {
			return false, err
		}
	}
	return true, tx.Commit(ctx)
}

// Emir icin gereken tutari kullanilabilir bakiyeden ayirir. Bakiye yetmezse errInsufficient doner.
func (l *ledger) reserve(ctx context.Context, orderID, userID, asset string, amount *big.Int) error {
	if l.exempt[userID] {
		return nil
	}
	err := l.tryReserve(ctx, orderID, userID, asset, amount)
	if !errors.Is(err, errInsufficient) {
		return err
	}
	// Bakiye sisteminden once acilmis hesaplar ilk emirde deneme bakiyesini alir.
	granted, ferr := l.ensureFunds(ctx, userID)
	if ferr != nil {
		return ferr
	}
	if !granted {
		return errInsufficient
	}
	return l.tryReserve(ctx, orderID, userID, asset, amount)
}

func (l *ledger) tryReserve(ctx context.Context, orderID, userID, asset string, amount *big.Int) error {
	tx, err := l.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE balances SET available = available - $3::text::numeric, locked = locked + $3::text::numeric
		WHERE user_id = $1 AND asset = $2 AND available >= $3::text::numeric
	`, userID, asset, amount.String())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errInsufficient
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_locks (order_id, user_id, asset, remaining) VALUES ($1, $2, $3, $4::text::numeric)
	`, orderID, userID, asset, amount.String()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Emir engine'e iletilemediyse ayrilan tutari geri verir.
func (l *ledger) unreserve(ctx context.Context, orderID string) error {
	tx, err := l.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var userID, asset, remaining string
	err = tx.QueryRow(ctx, `
		DELETE FROM order_locks WHERE order_id = $1 RETURNING user_id, asset, remaining::text
	`, orderID).Scan(&userID, &asset, &remaining)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE balances SET locked = locked - $3::text::numeric, available = available + $3::text::numeric
		WHERE user_id = $1 AND asset = $2
	`, userID, asset, remaining); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Degerler API'nin geri kalani gibi 10^4 olceklidir.
type BalanceRow struct {
	Asset     string      `json:"asset"`
	Available json.Number `json:"available"`
	Locked    json.Number `json:"locked"`
}

// GET /balances : giris yapan kullanicinin bakiyeleri
func (l *ledger) balancesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := userIDFrom(r.Context())
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if !l.exempt[userID] {
			if _, err := l.ensureFunds(ctx, userID); err != nil {
				log.Printf("deneme bakiyesi hatasi: %v", err)
			}
		}

		rows, err := l.db.Query(ctx, `
			SELECT asset, trunc(available / $2)::text, trunc(locked / $2)::text
			FROM balances WHERE user_id = $1 ORDER BY asset
		`, userID, money.Scale)
		if err != nil {
			log.Printf("bakiye sorgu hatasi: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "bakiye okunamadi"})
			return
		}
		defer rows.Close()

		out := []BalanceRow{}
		for rows.Next() {
			var b BalanceRow
			var available, locked string
			if err := rows.Scan(&b.Asset, &available, &locked); err != nil {
				log.Printf("bakiye okuma hatasi: %v", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "bakiye okunamadi"})
				return
			}
			b.Available, b.Locked = json.Number(available), json.Number(locked)
			out = append(out, b)
		}
		writeJSON(w, http.StatusOK, out)
	}
}
