// Sunucu adresi EXPO_PUBLIC_API_BASE ile degistirilebilir; yerelde calisirken
// telefonun erisebilecegi adres verilmelidir (ornegin http://192.168.1.10:8072).
const BASE_URL = (process.env.EXPO_PUBLIC_API_BASE || "https://api.yusufyonturk.com/tradesync/gateway").replace(/\/$/, "");

export const WS_URL = BASE_URL.replace(/^http/, "ws") + "/ws";

// Websocket'ten iki tur mesaj gelir: derinlik (bids/asks) ve islem (price/quantity).
export function parseMessage(raw) {
  let msg;
  try {
    msg = JSON.parse(raw);
  } catch {
    return null;
  }
  if (!msg || typeof msg.symbol !== "string") return null;
  if (msg.bids !== undefined || msg.asks !== undefined) {
    return { type: "depth", symbol: msg.symbol, bids: msg.bids || [], asks: msg.asks || [] };
  }
  if (msg.price !== undefined && msg.quantity !== undefined) {
    return { type: "trade", symbol: msg.symbol, price: msg.price, quantity: msg.quantity, timestamp: msg.timestamp };
  }
  return null;
}

export async function fetchTrades(symbol = "") {
  const url = symbol ? `${BASE_URL}/trades?symbol=${encodeURIComponent(symbol)}` : `${BASE_URL}/trades`;
  const res = await fetch(url);
  if (!res.ok) throw new Error("Trades fetch failed");
  return res.json();
}

export async function fetchDepth(symbol) {
  const res = await fetch(`${BASE_URL}/depth?symbol=${encodeURIComponent(symbol)}`);
  if (!res.ok) throw new Error("Depth fetch failed");
  return res.json();
}

// Emir gondermek icin login() ile alinan token gerekir.
export async function submitOrder(order, token) {
  const res = await fetch(`${BASE_URL}/orders`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify(order),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Order submission failed");
  return data;
}

export async function login(username, password) {
  const res = await fetch(`${BASE_URL}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Login failed");
  return data;
}
