/**
 * Zaman damgasını HH:MM:SS:mmm formatına çevirir
 */
export function formatTime(t) {
  const d = new Date(t);
  const hh = d.getHours().toString().padStart(2, '0');
  const mm = d.getMinutes().toString().padStart(2, '0');
  const ss = d.getSeconds().toString().padStart(2, '0');
  const ms = d.getMilliseconds().toString().padStart(3, '0');
  return `${hh}:${mm}:${ss}:${ms}`;
}

/**
 * Ham integer fiyatı (x10000) okunabilir Türkçe formatına çevirir
 * Örn: 150000000 → "15.000,00"
 */
export const formatPrice = (p) =>
  Number(p / 10000).toLocaleString('tr-TR', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });

/**
 * Ham integer miktarı (x10000) okunabilir formata çevirir
 * Örn: 10000 → "1"
 */
export const formatQty = (q) =>
  Number(q / 10000).toLocaleString('tr-TR', { maximumFractionDigits: 4 });
