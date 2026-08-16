import { useState, useEffect, useMemo } from 'react';
import { fetchTrades, fetchDepth, fetchKlines, WS_URL } from '../api';

/**
 * Market verilerini (trade, depth, klines) ve WebSocket bağlantısını yöneten hook.
 *
 * @param {string} market - Seçili piyasa (örn. "BTC-USD")
 * @param {object|null} user  - Giriş yapmış kullanıcı (null ise WS kurulmaz)
 * @returns {{ trades, depth, klines, isConnected, tickerPrices }}
 */
export function useMarketData(market, user) {
  const [trades, setTrades]           = useState([]);
  const [depth, setDepth]             = useState({ bids: [], asks: [] });
  const [klines, setKlines]           = useState([]);
  const [isConnected, setIsConnected] = useState(false);
  const [tickerPrices, setTickerPrices] = useState({});

  // ---------- REST Çağrıları ----------

  useEffect(() => {
    // piyasa değiştikten sonra gelen eski yanıtlar yok sayılır
    let active = true;
    const apply = (setter) => (data) => { if (active) setter(data); };

    fetchTrades(market).then(apply(setTrades)).catch(() => { /* sessiz */ });
    fetchDepth(market).then(apply(setDepth)).catch(() => { /* sessiz */ });
    fetchKlines(market).then(apply((data) => setKlines(data || []))).catch(() => { /* sessiz */ });

    return () => { active = false; };
  }, [market]);

  // ---------- WebSocket ----------

  useEffect(() => {
    if (!user) return;

    const ws = new WebSocket(WS_URL);

    ws.onopen  = () => setIsConnected(true);
    ws.onclose = () => setIsConnected(false);

    ws.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        const sym  = data.Symbol || data.symbol;

        // Ticker fiyatını güncelle (tüm piyasalar için)
        if (data.price !== undefined && data.quantity !== undefined) {
          setTickerPrices((prev) => ({ ...prev, [sym]: data.price }));
        }

        if (sym !== market) return;

        const hasDepth = data.Bids !== undefined || data.Asks !== undefined
                      || data.bids !== undefined || data.asks !== undefined;

        if (hasDepth) {
          setDepth({
            bids: data.Bids || data.bids || [],
            asks: data.Asks || data.asks || [],
          });
        } else if (data.price !== undefined && data.quantity !== undefined) {
          // Yeni trade geldi
          setTrades((prev) => [data, ...prev].slice(0, 100));
        }
      } catch { /* geçersiz mesaj */ }
    };

    return () => ws.close();
  }, [user, market]);

  // Seçili piyasaya ait trade'leri filtrele
  const filteredTrades = useMemo(
    () => trades.filter((t) => t.symbol === market),
    [trades, market]
  );

  return { trades: filteredTrades, depth, klines, isConnected, tickerPrices };
}
