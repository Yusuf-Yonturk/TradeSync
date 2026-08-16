// Yerelde calisirken web/.env.local icine VITE_API_BASE=http://localhost:8072 yazilabilir.
const BASE = (import.meta.env.VITE_API_BASE || "https://api.yusufyonturk.com/tradesync/gateway").replace(/\/$/, "");

export const WS_URL = BASE.replace(/^http/, "ws") + "/ws";

// Oturum gecersizse cagiran taraf kullaniciyi cikisa yonlendirebilsin.
export class AuthError extends Error {}

export async function fetchTrades(symbol = "") {
  const url = symbol
    ? `${BASE}/trades?symbol=${encodeURIComponent(symbol)}`
    : `${BASE}/trades`;
  const res = await fetch(url);
  if (!res.ok) throw new Error("trades alinamadi");
  return res.json();
}

export async function fetchKlines(symbol) {
  const res = await fetch(`${BASE}/klines?symbol=${encodeURIComponent(symbol)}`);
  if (!res.ok) throw new Error("Failed to fetch klines");
  const raw = await res.json();
  return raw.map(k => ({
    ...k,
    open: k.open / 10000,
    high: k.high / 10000,
    low: k.low / 10000,
    close: k.close / 10000
  }));
}

export async function fetchDepth(symbol) {
  const res = await fetch(`${BASE}/depth?symbol=${encodeURIComponent(symbol)}`);
  if (!res.ok) throw new Error("depth fetch failed");
  return res.json();
}

export async function register(name, username, password) {
  const res = await fetch(`${BASE}/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, username, password }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Kayıt olunamadı");
  return data;
}

export async function login(username, password) {
  const res = await fetch(`${BASE}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Giriş yapılamadı");
  return data;
}

export async function submitOrder(order, token) {
  const res = await fetch(`${BASE}/orders`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(order),
  });
  const data = await res.json();
  if (res.status === 401) throw new AuthError(data.error || "Oturum süresi doldu");
  if (!res.ok) throw new Error(data.error || "order gonderilemedi");
  return data;
}

/**
 * Giriş yapan kullanıcının açık (bekleyen) emirlerini getirir.
 */
export async function fetchOrders(token, symbol = "") {
  const params = new URLSearchParams();
  if (symbol) params.append("symbol", symbol);
  const res = await fetch(`${BASE}/orders?${params}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (res.status === 401) throw new AuthError("Oturum süresi doldu");
  if (!res.ok) throw new Error("emirler alinamadi");
  return res.json();
}

/**
 * Belirli bir emiri iptal eder. İptal asenkrondur; emir bu arada dolmuş olabilir.
 */
export async function cancelOrder(orderId, token) {
  const res = await fetch(`${BASE}/orders/${encodeURIComponent(orderId)}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });
  const data = await res.json().catch(() => ({}));
  if (res.status === 401) throw new AuthError(data.error || "Oturum süresi doldu");
  if (!res.ok) throw new Error(data.error || "emir iptal edilemedi");
  return data;
}

/**
 * Giriş yapan kullanıcının bakiyeleri. Değerler diğer uçlar gibi 10^4 ölçeklidir.
 */
export async function fetchBalances(token) {
  const res = await fetch(`${BASE}/balances`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (res.status === 401) throw new AuthError("Oturum süresi doldu");
  if (!res.ok) throw new Error("bakiye alinamadi");
  return res.json();
}

/**
 * Giriş yapan kullanıcının gerçekleşmiş işlemleri; her kayıtta yön (side) bulunur.
 */
export async function fetchMyTrades(token, symbol = "") {
  const params = new URLSearchParams();
  if (symbol) params.append("symbol", symbol);
  const res = await fetch(`${BASE}/trades/mine?${params}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (res.status === 401) throw new AuthError("Oturum süresi doldu");
  if (!res.ok) throw new Error("islemler alinamadi");
  return res.json();
}
