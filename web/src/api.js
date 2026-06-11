const BASE = "https://api.yusufyonturk.com/tradesync/gateway";

export async function fetchTrades(symbol = "") {
  const url = symbol
    ? `${BASE}/trades?symbol=${symbol}`
    : `${BASE}/trades`;
  const res = await fetch(url);
  if (!res.ok) throw new Error("trades alinamadi");
  return res.json();
}

export async function fetchKlines(symbol) {
  const res = await fetch(`${BASE}/klines?symbol=${symbol}`);
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
  const res = await fetch(`${BASE}/depth?symbol=${symbol}`);
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

export async function submitOrder(order) {
  const res = await fetch(`${BASE}/orders`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(order),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "order gonderilemedi");
  return data;
}
