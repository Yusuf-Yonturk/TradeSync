const BASE_URL = "https://715b27a221be60.lhr.life";

export async function fetchTrades(symbol = "") {
  const url = symbol ? `${BASE_URL}/trades?symbol=${symbol}` : `${BASE_URL}/trades`;
  const res = await fetch(url);
  if (!res.ok) throw new Error("Trades fetch failed");
  return res.json();
}

export async function fetchDepth(symbol) {
  const res = await fetch(`${BASE_URL}/depth?symbol=${symbol}`);
  if (!res.ok) throw new Error("Depth fetch failed");
  return res.json();
}

export async function submitOrder(order) {
  const res = await fetch(`${BASE_URL}/orders`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
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
