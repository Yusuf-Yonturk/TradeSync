// Package money emir tutarlarini hesaplar. Fiyat ve miktar 10^4 olcekli
// tamsayidir; bakiyeler 10^8 olcekli tutulur ki fiyat*miktar tam kalsin.
package money

import (
	"math/big"
	"strings"
)

// Fiyat ve miktarin olcegi.
const Scale = 10000

// Notional, fiyat*miktar tutarini bakiye olceginde verir.
func Notional(price, quantity int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(price), big.NewInt(quantity))
}

// Base, miktari bakiye olcegine cevirir.
func Base(quantity int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(quantity), big.NewInt(Scale))
}

// FromUnits, tam birimi (ornegin 5 BTC) bakiye olcegine cevirir.
func FromUnits(units int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(units), big.NewInt(Scale*Scale))
}

// SplitSymbol "BTC-USD" icin ("BTC", "USD") doner.
func SplitSymbol(symbol string) (base, quote string, ok bool) {
	base, quote, ok = strings.Cut(symbol, "-")
	if !ok || base == "" || quote == "" || strings.Contains(quote, "-") {
		return "", "", false
	}
	return base, quote, true
}

// Reserve, bir emrin kalan miktari icin ayrilmasi gereken varligi ve tutari verir:
// alis icin karsi para biriminden fiyat*miktar, satis icin satilan varliktan miktar.
func Reserve(symbol, side string, price, quantity int64) (asset string, amount *big.Int, ok bool) {
	base, quote, ok := SplitSymbol(symbol)
	if !ok || price <= 0 || quantity <= 0 {
		return "", nil, false
	}
	switch side {
	case "BUY":
		return quote, Notional(price, quantity), true
	case "SELL":
		return base, Base(quantity), true
	}
	return "", nil, false
}
