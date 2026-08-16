import { useState, useEffect } from 'react';
import { formatTime, formatPrice, formatQty } from '../../utils/format';
import { fetchOrders, fetchMyTrades, cancelOrder, AuthError } from '../../api';
import './TradeList.css';

// Emirler asenkron işlendiği için kişisel listeler sekme açıkken bu aralıkla yenilenir
const REFRESH_MS = 3000;

const EMPTY = { key: "", rows: [] };

export default function TradeListPanel({ trades, user, market, onSessionExpired }) {
  const [tab, setTab] = useState("ALL");
  // key, satırların hangi sekme ve piyasa için yüklendiğini tutar
  const [orders, setOrders] = useState(EMPTY);
  const [myTrades, setMyTrades] = useState(EMPTY);

  const token = user?.token;
  const viewKey = `${tab}:${market}`;

  // "İşlemlerim" ve "Açık Emirler" kullanıcıya özeldir, sunucudan token ile çekilir
  useEffect(() => {
    if (tab === "ALL" || !token) return;
    const key = `${tab}:${market}`;
    let active = true;

    const load = () => {
      const request = tab === "ORDERS" ? fetchOrders(token, market) : fetchMyTrades(token, market);
      request
        .then((data) => {
          if (!active) return;
          const result = { key, rows: Array.isArray(data) ? data : [] };
          if (tab === "ORDERS") setOrders(result);
          else setMyTrades(result);
        })
        .catch((err) => {
          // arka plan yenilemesi başarısız olursa eldeki listeyi koru
          if (active && err instanceof AuthError) onSessionExpired?.();
        });
    };

    load();
    const timer = setInterval(load, REFRESH_MS);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [tab, market, token, onSessionExpired]);

  const handleCancel = async (orderId) => {
    try {
      await cancelOrder(orderId, user.token);
      setOrders(prev => ({ ...prev, rows: prev.rows.filter(o => o.id !== orderId) }));
    } catch (err) {
      alert("İptal edilemedi: " + err.message);
      if (err instanceof AuthError) onSessionExpired?.();
    }
  };

  const loadingOrders = orders.key !== viewKey;
  const loadingMine = tab === "MY" && myTrades.key !== viewKey;
  const displayTrades = tab === "MY" ? (loadingMine ? [] : myTrades.rows) : trades;

  return (
    <div className="panel trade-list">
      {/* Tab Başlıkları */}
      <div className="panel-header tl-tabs">
        <span
          className={`tl-tab ${tab === "ALL" ? "active" : ""}`}
          onClick={() => setTab("ALL")}
        >
          Son İşlemler
        </span>
        <span
          className={`tl-tab ${tab === "MY" ? "active" : ""}`}
          onClick={() => setTab("MY")}
        >
          İşlemlerim
        </span>
        <span
          className={`tl-tab ${tab === "ORDERS" ? "active" : ""}`}
          onClick={() => setTab("ORDERS")}
        >
          Açık Emirler
        </span>
      </div>

      {/* ---- Tamamlanmış İşlemler Görünümü ---- */}
      {tab !== "ORDERS" && (
        <>
          <div className="ob-cols trades-cols">
            <span>Zaman</span>
            <span>Fiyat</span>
            <span>Adet</span>
          </div>
          <div className="ob-list">
            {displayTrades.length === 0 ? (
              <div className="tl-empty">
                {loadingMine ? "Yükleniyor..." : tab === "MY" ? "Henüz tamamlanmış işleminiz yok." : "İşlem bulunamadı."}
              </div>
            ) : displayTrades.map((t, idx) => {
              // herkese açık listede yön bilgisi yoktur
              const colorClass = t.side === "SELL" ? "red" : "green";
              return (
                <div key={t.id || idx} className="trade-row">
                  <span className="muted">{formatTime(t.traded_at || t.timestamp)}</span>
                  <span className={colorClass}>{formatPrice(t.price)}</span>
                  <span>{formatQty(t.quantity)}</span>
                </div>
              );
            })}
          </div>
        </>
      )}

      {/* ---- Açık Emirler Görünümü ---- */}
      {tab === "ORDERS" && (
        <>
          <div className="ob-cols orders-cols">
            <span>Yön</span>
            <span>Fiyat</span>
            <span>Miktar</span>
            <span>İptal</span>
          </div>
          <div className="ob-list">
            {loadingOrders ? (
              <div className="tl-empty">Yükleniyor...</div>
            ) : orders.rows.length === 0 ? (
              <div className="tl-empty">Açık emir bulunmuyor.</div>
            ) : orders.rows.map((o, idx) => (
              <div key={o.id || idx} className="order-row">
                <span className={o.side === "BUY" ? "green" : "red"}>
                  {o.side === "BUY" ? "AL" : "SAT"}
                </span>
                <span>{formatPrice(o.price)}</span>
                <span>{formatQty(o.remaining ?? o.quantity)}</span>
                <button
                  className="cancel-btn"
                  onClick={() => handleCancel(o.id)}
                  title="Emiri İptal Et"
                >
                  ✕
                </button>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
