package main

import (
	"context"
	"errors"
	"math/big"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tradesync/shared/money"
	"tradesync/shared/schema"
)

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

// Bir islemin bakiyelere etkisi.
type tradeAmounts struct {
	base, quote string
	// alicinin emri icin ayrilmis tutardan dusulecek kisim (limit fiyat * miktar)
	buyerReserved *big.Int
	// islem limitten ucuza gerceklestiyse aliciya geri donen fark
	buyerRefund *big.Int
	// alicinin aldigi, saticinin verdigi varlik
	baseAmount *big.Int
	// saticinin aldigi tutar (islem fiyati * miktar)
	quoteAmount *big.Int
}

func amountsFor(t TradeEvent) (tradeAmounts, bool) {
	base, quote, ok := money.SplitSymbol(t.Symbol)
	if !ok || t.Price <= 0 || t.Quantity <= 0 {
		return tradeAmounts{}, false
	}
	limit := t.BuyLimit
	// eski kayitlarda limit yok; islem fiyati limitin ustunde de olamaz
	if limit < t.Price {
		limit = t.Price
	}
	paid := money.Notional(t.Price, t.Quantity)
	reserved := money.Notional(limit, t.Quantity)
	return tradeAmounts{
		base:          base,
		quote:         quote,
		buyerReserved: reserved,
		buyerRefund:   new(big.Int).Sub(reserved, paid),
		baseAmount:    money.Base(t.Quantity),
		quoteAmount:   paid,
	}, true
}

// Iptal edilen emrin kalan kismi icin serbest birakilacak tutar.
func releaseAmount(u OrderUpdate) (*big.Int, bool) {
	if u.Status != "CANCELLED" {
		return nil, false
	}
	_, amount, ok := money.Reserve(u.Symbol, u.Side, u.Price, u.Quantity-u.FilledQty)
	return amount, ok
}

// Islemi kaydeder ve bakiyeleri tek transaction'da gunceller.
// Ayni trade_id daha once yazildiysa hicbir sey yapmaz ve false doner.
func settleTrade(ctx context.Context, db *pgxpool.Pool, t TradeEvent) (bool, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		INSERT INTO trades (trade_id, buy_order_id, sell_order_id, buyer_id, seller_id, symbol, price, quantity, traded_at)
		VALUES (NULLIF($1, ''), $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (trade_id) DO NOTHING
	`, t.ID, t.BuyOrderID, t.SellOrderID, t.BuyerID, t.SellerID, t.Symbol, t.Price, t.Quantity, t.Timestamp)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}

	if a, ok := amountsFor(t); ok {
		if err := consumeLock(ctx, tx, t.BuyOrderID, a.buyerReserved, a.buyerRefund); err != nil {
			return false, err
		}
		if err := credit(ctx, tx, t.BuyerID, a.base, a.baseAmount); err != nil {
			return false, err
		}
		if err := consumeLock(ctx, tx, t.SellOrderID, a.baseAmount, new(big.Int)); err != nil {
			return false, err
		}
		if err := credit(ctx, tx, t.SellerID, a.quote, a.quoteAmount); err != nil {
			return false, err
		}
	}
	return true, tx.Commit(ctx)
}

// Emrin ayrilmis tutarindan amount kadarini harcar; refund kadarini sahibine geri verir.
// Emir icin ayrilmis tutar yoksa (bakiye kontrolunden muaf ya da eski emir) dokunmaz.
func consumeLock(ctx context.Context, tx pgx.Tx, orderID string, amount, refund *big.Int) error {
	var userID, asset string
	var empty bool
	err := tx.QueryRow(ctx, `
		UPDATE order_locks SET remaining = remaining - $2::text::numeric
		WHERE order_id = $1 AND remaining >= $2::text::numeric
		RETURNING user_id, asset, remaining = 0
	`, orderID, amount.String()).Scan(&userID, &asset, &empty)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE balances SET locked = locked - $3::text::numeric, available = available + $4::text::numeric
		WHERE user_id = $1 AND asset = $2
	`, userID, asset, amount.String(), refund.String()); err != nil {
		return err
	}
	return dropLockIfEmpty(ctx, tx, orderID, empty)
}

func dropLockIfEmpty(ctx context.Context, tx pgx.Tx, orderID string, empty bool) error {
	if !empty {
		return nil
	}
	_, err := tx.Exec(ctx, `DELETE FROM order_locks WHERE order_id = $1`, orderID)
	return err
}

func credit(ctx context.Context, tx pgx.Tx, userID, asset string, amount *big.Int) error {
	if userID == "" || amount.Sign() <= 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO balances (user_id, asset, available) VALUES ($1, $2, $3::text::numeric)
		ON CONFLICT (user_id, asset) DO UPDATE SET available = balances.available + EXCLUDED.available
	`, userID, asset, amount.String())
	return err
}

// Iptal edilen emrin kalan tutarini kullaniciya geri verir. released isareti
// sayesinde ayni iptal ikinci kez gelirse bir sey yapmaz. Emrin daha once
// gerceklesmis ama henuz islenmemis islemleri icin ayrilan kisim satirda kalir.
func releaseOrder(ctx context.Context, db *pgxpool.Pool, orderID string, amount *big.Int) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var userID, asset, freed string
	var empty bool
	err = tx.QueryRow(ctx, `
		WITH old AS (
			SELECT order_id, user_id, asset, remaining FROM order_locks
			WHERE order_id = $1 AND NOT released
			FOR UPDATE
		)
		UPDATE order_locks l
		SET remaining = l.remaining - LEAST(old.remaining, $2::text::numeric), released = TRUE
		FROM old WHERE l.order_id = old.order_id
		RETURNING old.user_id, old.asset, LEAST(old.remaining, $2::text::numeric)::text, l.remaining = 0
	`, orderID, amount.String()).Scan(&userID, &asset, &freed, &empty)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE balances SET locked = locked - $3::text::numeric, available = available + $3::text::numeric
		WHERE user_id = $1 AND asset = $2
	`, userID, asset, freed); err != nil {
		return err
	}
	if err := dropLockIfEmpty(ctx, tx, orderID, empty); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
