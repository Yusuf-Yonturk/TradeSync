package main

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Bundan eski mumlar silinir; aksi halde hash sonsuza kadar buyur.
const klineRetention = 7 * 24 * time.Hour

type Kline struct {
	Time   int64 `json:"time"`
	Open   int64 `json:"open"`
	High   int64 `json:"high"`
	Low    int64 `json:"low"`
	Close  int64 `json:"close"`
	Volume int64 `json:"volume"`
}

func klineMinute(t time.Time) int64 {
	return t.Truncate(time.Minute).Unix()
}

func newKline(minute, price, quantity int64) Kline {
	return Kline{Time: minute, Open: price, High: price, Low: price, Close: price, Volume: quantity}
}

func (k Kline) with(price, quantity int64) Kline {
	if price > k.High {
		k.High = price
	}
	if price < k.Low {
		k.Low = price
	}
	k.Close = price
	k.Volume += quantity
	return k
}

// cutoff'tan eski dakikalara ait alanlari doner; sayi olmayan alanlar da temizlenir.
func staleFields(fields []string, cutoff int64) []string {
	var stale []string
	for _, f := range fields {
		minute, err := strconv.ParseInt(f, 10, 64)
		if err != nil || minute < cutoff {
			stale = append(stale, f)
		}
	}
	return stale
}

func updateKline(ctx context.Context, rdb *redis.Client, t TradeEvent) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	minute := klineMinute(t.Timestamp)
	key := "klines:" + t.Symbol
	field := strconv.FormatInt(minute, 10)

	val, err := rdb.HGet(ctx, key, field).Result()
	var k Kline
	switch {
	case err == redis.Nil:
		k = newKline(minute, t.Price, t.Quantity)
		// yeni mum dakikada bir acilir; eskileri o sirada temizle
		trimKlines(ctx, rdb, key, minute-int64(klineRetention/time.Second))
	case err != nil:
		log.Printf("kline okuma hatasi: %v", err)
		return
	default:
		if err := json.Unmarshal([]byte(val), &k); err != nil {
			log.Printf("kline parse hatasi: %v", err)
			return
		}
		k = k.with(t.Price, t.Quantity)
	}

	b, err := json.Marshal(k)
	if err != nil {
		log.Printf("kline marshal hatasi: %v", err)
		return
	}
	if err := rdb.HSet(ctx, key, field, b).Err(); err != nil {
		log.Printf("kline yazma hatasi: %v", err)
	}
}

func trimKlines(ctx context.Context, rdb *redis.Client, key string, cutoff int64) {
	fields, err := rdb.HKeys(ctx, key).Result()
	if err != nil {
		log.Printf("kline temizlik hatasi: %v", err)
		return
	}
	if stale := staleFields(fields, cutoff); len(stale) > 0 {
		if err := rdb.HDel(ctx, key, stale...).Err(); err != nil {
			log.Printf("kline temizlik hatasi: %v", err)
		}
	}
}
