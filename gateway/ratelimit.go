package main

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Sabit pencereli sayac: bir anahtar pencere icinde en fazla limit kez sayilabilir.
type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]*rateEntry
	swept   time.Time
	now     func() time.Time
}

type rateEntry struct {
	count int
	reset time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, entries: make(map[string]*rateEntry), now: time.Now}
}

// Anahtar sinira ulastiysa ne kadar beklemesi gerektigini doner; ulasmadiysa sifir.
func (l *rateLimiter) blockedFor(key string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	e := l.entries[key]
	if e == nil || !now.Before(e.reset) || e.count < l.limit {
		return 0
	}
	return e.reset.Sub(now)
}

func (l *rateLimiter) add(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	e := l.entries[key]
	if e == nil || !now.Before(e.reset) {
		e = &rateEntry{reset: now.Add(l.window)}
		l.entries[key] = e
	}
	e.count++
}

// Suresi dolan kayitlari ara sira temizler; harita sinirsiz buyumesin.
func (l *rateLimiter) sweep(now time.Time) {
	if now.Sub(l.swept) < l.window {
		return
	}
	l.swept = now
	for k, e := range l.entries {
		if !now.Before(e.reset) {
			delete(l.entries, k)
		}
	}
}

// Istemcinin adresi. Gateway bir vekil sunucunun arkasindaysa gercek adres
// CLIENT_IP_HEADER ile belirtilen basliktan (ornegin X-Real-IP) okunur;
// baslik yalniz vekil tarafindan yazildigindan emin olunmalidir.
func clientIP(r *http.Request, header string) string {
	if header != "" {
		if v := r.Header.Get(header); v != "" {
			return v
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func tooManyRequests(w http.ResponseWriter, wait time.Duration) {
	w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
	writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "cok fazla deneme, biraz sonra tekrar deneyin"})
}
