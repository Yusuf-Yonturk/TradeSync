#!/bin/bash
USER_ID="demo-user-123"

# BTC-USD Bids
curl -X POST http://localhost:8072/orders -H "Content-Type: application/json" -d '{"user_id":"'$USER_ID'", "symbol":"BTC-USD", "side":"BUY", "price":64000, "quantity": 10}'
curl -X POST http://localhost:8072/orders -H "Content-Type: application/json" -d '{"user_id":"'$USER_ID'", "symbol":"BTC-USD", "side":"BUY", "price":64100, "quantity": 5}'
curl -X POST http://localhost:8072/orders -H "Content-Type: application/json" -d '{"user_id":"'$USER_ID'", "symbol":"BTC-USD", "side":"BUY", "price":64500, "quantity": 2}'

# BTC-USD Asks
curl -X POST http://localhost:8072/orders -H "Content-Type: application/json" -d '{"user_id":"'$USER_ID'", "symbol":"BTC-USD", "side":"SELL", "price":65000, "quantity": 3}'
curl -X POST http://localhost:8072/orders -H "Content-Type: application/json" -d '{"user_id":"'$USER_ID'", "symbol":"BTC-USD", "side":"SELL", "price":65100, "quantity": 8}'
curl -X POST http://localhost:8072/orders -H "Content-Type: application/json" -d '{"user_id":"'$USER_ID'", "symbol":"BTC-USD", "side":"SELL", "price":65500, "quantity": 1}'

# Trades
curl -X POST http://localhost:8072/orders -H "Content-Type: application/json" -d '{"user_id":"'$USER_ID'", "symbol":"BTC-USD", "side":"BUY", "price":65000, "quantity": 1}'
sleep 1
curl -X POST http://localhost:8072/orders -H "Content-Type: application/json" -d '{"user_id":"'$USER_ID'", "symbol":"BTC-USD", "side":"SELL", "price":64500, "quantity": 1}'

# ETH-USD Bids
curl -X POST http://localhost:8072/orders -H "Content-Type: application/json" -d '{"user_id":"'$USER_ID'", "symbol":"ETH-USD", "side":"BUY", "price":3400, "quantity": 100}'
curl -X POST http://localhost:8072/orders -H "Content-Type: application/json" -d '{"user_id":"'$USER_ID'", "symbol":"ETH-USD", "side":"BUY", "price":3410, "quantity": 50}'

# ETH-USD Asks
curl -X POST http://localhost:8072/orders -H "Content-Type: application/json" -d '{"user_id":"'$USER_ID'", "symbol":"ETH-USD", "side":"SELL", "price":3450, "quantity": 20}'
curl -X POST http://localhost:8072/orders -H "Content-Type: application/json" -d '{"user_id":"'$USER_ID'", "symbol":"ETH-USD", "side":"SELL", "price":3460, "quantity": 30}'

# ETH Trades
curl -X POST http://localhost:8072/orders -H "Content-Type: application/json" -d '{"user_id":"'$USER_ID'", "symbol":"ETH-USD", "side":"BUY", "price":3450, "quantity": 5}'
