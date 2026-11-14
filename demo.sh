#!/bin/bash
# Ornek emirler gonderir. Gateway token istedigi icin once demo hesaplariyla giris yapar.
set -euo pipefail

GATEWAY="${GATEWAY_URL:-http://localhost:8072}"
DEMO_USER="${DEMO_USER:-demo_user}"
DEMO_PASS="${DEMO_PASS:-demo-pass-123}"

# Hesap yoksa olusturur, giris yapip token'i yazar.
login() {
  local creds='{"name":"Demo","username":"'$1'","password":"'$DEMO_PASS'"}'
  # hesap zaten varsa 409 doner, sorun degil
  curl -s -o /dev/null -X POST "$GATEWAY/auth/register" -H "Content-Type: application/json" -d "$creds"
  curl -s -X POST "$GATEWAY/auth/login" -H "Content-Type: application/json" -d "$creds" \
    | sed -n 's/.*"token":"\([^"]*\)".*/\1/p'
}

# Kullanici kendi emriyle eslesemedigi icin alislar ve satislar ayri hesaplardan gider.
BUYER=$(login "$DEMO_USER")
SELLER=$(login "${DEMO_USER}_2")
if [ -z "$BUYER" ] || [ -z "$SELLER" ]; then
  echo "giris yapilamadi" >&2
  exit 1
fi

order() {
  local token=$BUYER
  if [ "$2" = "SELL" ]; then token=$SELLER; fi
  curl -s -X POST "$GATEWAY/orders" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $token" \
    -d '{"symbol":"'$1'", "side":"'$2'", "price":'$3', "quantity":'$4'}'
  echo
}

# BTC-USD Bids
order BTC-USD BUY 64000 10
order BTC-USD BUY 64100 5
order BTC-USD BUY 64500 2

# BTC-USD Asks
order BTC-USD SELL 65000 3
order BTC-USD SELL 65100 8
order BTC-USD SELL 65500 1

# Trades
order BTC-USD BUY 65000 1
sleep 1
order BTC-USD SELL 64500 1

# ETH-USD Bids
order ETH-USD BUY 3400 100
order ETH-USD BUY 3410 50

# ETH-USD Asks
order ETH-USD SELL 3450 20
order ETH-USD SELL 3460 30

# ETH Trades
order ETH-USD BUY 3450 5
