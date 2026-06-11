import { useState, useEffect, useCallback, useMemo } from 'react';
import { fetchTrades, fetchDepth, fetchKlines } from '../api';

const WS_URL = 'wss://api.yusufyonturk.com/tradesync/gateway/ws';

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

  const loadTrades = useCallback(async () => {
    try {
      const data = await fetchTrades(market);
      setTrades(data);
    } catch { /* sessiz */ }
  }, [market]);

  const loadDepth = useCallback(async () => {
    try {
      const data = await fetchDepth(market);
      setDepth(data);
    } catch { /* sessiz */ }
  }, [market]);

  const loadKlines = useCallback(async () => {
    try {
      const data = await fetchKlines(market);
      setKlines(data || []);
    } catch { /* sessiz */ }
  }, [market]);

  useEffect(() => {
    loadTrades();
    loadDepth();
    loadKlines();
  }, [loadTrades, loadDepth, loadKlines]);

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
