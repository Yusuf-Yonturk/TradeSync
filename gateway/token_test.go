package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTokenRoundTrip(t *testing.T) {
	s := newTokenSigner("test-secret")
	now := time.Unix(1_700_000_000, 0)

	tok := s.Issue("user-1", now)
	id, ok := s.Verify(tok, now.Add(time.Hour))
	if !ok || id != "user-1" {
		t.Fatalf("beklenen user-1, gelen %q ok=%v", id, ok)
	}
}

func TestTokenExpired(t *testing.T) {
	s := newTokenSigner("test-secret")
	now := time.Unix(1_700_000_000, 0)

	tok := s.Issue("user-1", now)
	if _, ok := s.Verify(tok, now.Add(tokenTTL)); ok {
		t.Fatal("suresi dolmus token kabul edildi")
	}
}

func TestTokenTampered(t *testing.T) {
	s := newTokenSigner("test-secret")
	other := newTokenSigner("baska-secret")
	now := time.Unix(1_700_000_000, 0)

	tok := s.Issue("user-1", now)
	payload, sig, _ := strings.Cut(tok, ".")
	forged, _, _ := strings.Cut(s.Issue("user-2", now), ".")

	bad := []string{
		"",
		"abc",
		payload,
		payload + ".",
		forged + "." + sig,
		other.Issue("user-1", now),
	}
	for _, b := range bad {
		if _, ok := s.Verify(b, now); ok {
			t.Errorf("gecersiz token kabul edildi: %q", b)
		}
	}
}

func TestRequireAuth(t *testing.T) {
	s := newTokenSigner("test-secret")
	var seen string
	h := s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		seen = userIDFrom(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	call := func(header string) int {
		req := httptest.NewRequest(http.MethodPost, "/orders", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rec := httptest.NewRecorder()
		h(rec, req)
		return rec.Code
	}

	if code := call(""); code != http.StatusUnauthorized {
		t.Errorf("token yokken beklenen 401, gelen %d", code)
	}
	if code := call("Bearer sahte.token"); code != http.StatusUnauthorized {
		t.Errorf("sahte token icin beklenen 401, gelen %d", code)
	}
	if code := call("Bearer " + s.Issue("user-7", time.Now())); code != http.StatusOK {
		t.Errorf("gecerli token icin beklenen 200, gelen %d", code)
	}
	if seen != "user-7" {
		t.Errorf("context'te beklenen user-7, gelen %q", seen)
	}
}
