import { useState, useEffect, useCallback } from 'react';
import { formatTime, formatPrice, formatQty } from '../../utils/format';
import { fetchOrders, cancelOrder } from '../../api';
import './TradeList.css';

export default function TradeListPanel({ trades, user, market }) {
  const [tab, setTab] = useState("ALL");
  const [orders, setOrders] = useState([]);
  const [loadingOrders, setLoadingOrders] = useState(false);

  // Açık emirleri çek — sadece "EMİRLERİM" sekmesindeyken
  const loadOrders = useCallback(async () => {
    if (!user?.user_id) return;
    setLoadingOrders(true);
    try {
      const data = await fetchOrders(user.user_id, market);
      // API array döndürürse kullan, yoksa boş dizi
      setOrders(Array.isArray(data) ? data : []);
    } catch {
      setOrders([]);
    } finally {
      setLoadingOrders(false);
    }
  }, [user, market]);

  useEffect(() => {
    if (tab === "ORDERS") loadOrders();
  }, [tab, loadOrders]);

  const handleCancel = async (orderId) => {
    try {
      await cancelOrder(orderId);
      setOrders(prev => prev.filter(o => o.id !== orderId));
    } catch (err) {
      alert("İptal edilemedi: " + err.message);
    }
  };

  // Tüm işlemler vs sadece benimkiler (tamamlanmış trade'ler)
  const displayTrades = tab === "ALL"
    ? trades
    : tab === "MY"
      ? trades.filter(t => t.buyer_id === user?.user_id || t.seller_id === user?.user_id)
      : null; // "ORDERS" sekmesi için null

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
                {tab === "MY" ? "Henüz tamamlanmış işleminiz yok." : "İşlem bulunamadı."}
              </div>
            ) : displayTrades.map((t, idx) => {
              const isMyBuy  = t.buyer_id  === user?.user_id;
              const isMySell = t.seller_id === user?.user_id;
              let colorClass = "green";
              if (tab === "MY") {
                if (isMySell && !isMyBuy) colorClass = "red";
              }
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
            ) : orders.length === 0 ? (
              <div className="tl-empty">Açık emir bulunmuyor.</div>
            ) : orders.map((o, idx) => (
              <div key={o.id || idx} className="order-row">
                <span className={o.side === "BUY" ? "green" : "red"}>
                  {o.side === "BUY" ? "AL" : "SAT"}
                </span>
                <span>{formatPrice(o.price)}</span>
                <span>{formatQty(o.quantity)}</span>
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
