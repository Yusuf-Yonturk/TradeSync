import { formatPrice, formatQty } from '../../utils/format';
import './OrderBook.css';

export default function OrderBookPanel({ depth, lastPrice }) {
  const asks = depth.asks || [];
  const bids = depth.bids || [];

  const maxAsk = Math.max(...asks.map(a => a.quantity), 1);
  const maxBid = Math.max(...bids.map(b => b.quantity), 1);

  return (
    <div className="panel">
      <div className="panel-header">Emir Defteri</div>
      <div className="ob-cols orderbook">
        <span>Fiyat</span>
        <span>Adet</span>
      </div>
      
      <div className="ob-list asks">
        {[...asks].reverse().map((a, i) => (
          <div key={`ask-${i}`} className="ob-row ask">
            <div className="depth-bar" style={{ width: `${(a.quantity / maxAsk) * 100}%` }}></div>
            <span className="red">{formatPrice(a.price)}</span>
            <span>{formatQty(a.quantity)}</span>
          </div>
        ))}
      </div>
      
      <div className="spread-row">
        <span className="spread-price green">{lastPrice ? formatPrice(lastPrice) : "---"}</span>
        <span className="spread-val">Spread: {asks.length > 0 && bids.length > 0 ? formatPrice(Math.abs(asks[0].price - bids[0].price)) : "---"}</span>
      </div>
      
      <div className="ob-list bids">
        {bids.map((b, i) => (
          <div key={`bid-${i}`} className="ob-row bid">
            <div className="depth-bar" style={{ width: `${(b.quantity / maxBid) * 100}%` }}></div>
            <span className="green">{formatPrice(b.price)}</span>
            <span>{formatQty(b.quantity)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
