import { useState } from 'react';
import { formatTime, formatPrice, formatQty } from '../../utils/format';
import './TradeList.css';

export default function TradeListPanel({ trades, user }) {
  const [tab, setTab] = useState("ALL");

  const displayTrades = tab === "ALL" 
    ? trades 
    : trades.filter(t => t.buyer_id === user?.user_id || t.seller_id === user?.user_id);

  return (
    <div className="panel trade-list">
      <div className="panel-header" style={{ display: 'flex', gap: '16px' }}>
        <span 
          style={{ cursor: 'pointer', color: tab === "ALL" ? '#fff' : '#848e9c' }} 
          onClick={() => setTab("ALL")}
        >
          Son İşlemler
        </span>
        <span 
          style={{ cursor: 'pointer', color: tab === "MY" ? '#fff' : '#848e9c' }} 
          onClick={() => setTab("MY")}
        >
          İşlemlerim
        </span>
      </div>
      <div className="ob-cols trades">
        <span>Zaman</span>
        <span>Fiyat</span>
        <span>Adet</span>
      </div>
      <div className="ob-list">
        {displayTrades.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '20px', color: '#848e9c', fontSize: '12px' }}>
            {tab === "MY" ? "Henüz işleminiz yok." : "İşlem bulunamadı."}
          </div>
        ) : displayTrades.map((t, idx) => {
          // Eğer bu işlem bana aitse, alıcı mıyım satıcı mıyım ona göre renk ver
          let isMyBuy = t.buyer_id === user?.user_id;
          let isMySell = t.seller_id === user?.user_id;
          
          let colorClass = "green"; // Default
          if (tab === "MY") {
             if (isMySell && !isMyBuy) colorClass = "red";
             else if (isMyBuy && !isMySell) colorClass = "green";
          } else {
             // Genel market için (rastgele veya Taker Side'a göre yapabilirdik, şu an basitçe)
             colorClass = "green";
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
    </div>
  );
}
