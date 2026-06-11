import { useState, useEffect } from 'react';
import { submitOrder } from '../../api';
import './OrderForm.css';

export default function OrderForm({ user, market, depth, balances }) {
  const [side, setSide] = useState("BUY");
  const [form, setForm] = useState({ price: "", quantity: "" });
  const [loading, setLoading] = useState(false);

  const crypto = market.split("-")[0];

  const set = (k, v) => setForm(f => ({ ...f, [k]: v }));

  const parseNumber = (val) => {
    if (!val) return 0;
    return Number(val.toString().replace(/\./g, '').replace(/,/g, '.'));
  };

  const formatInputValue = (num, isQty = false) => {
    if (!num) return "";
    return Number(num).toLocaleString("tr-TR", { maximumFractionDigits: isQty ? 4 : 2 });
  };

  const toggleSide = (newSide) => {
    setSide(newSide);
    if (newSide === "BUY" && depth?.asks?.length > 0) {
      set("price", formatInputValue(depth.asks[0].price / 10000));
    } else if (newSide === "SELL" && depth?.bids?.length > 0) {
      set("price", formatInputValue(depth.bids[0].price / 10000));
    }
  };

  useEffect(() => {
    // Reset form when market changes
    setForm({ price: "", quantity: "" });
  }, [market]);

  useEffect(() => {
    // Fill best price automatically ONLY if empty
    if (!form.price) {
      if (side === "BUY" && depth?.asks?.length > 0) {
        set("price", formatInputValue(depth.asks[0].price / 10000));
      } else if (side === "SELL" && depth?.bids?.length > 0) {
        set("price", formatInputValue(depth.bids[0].price / 10000));
      }
    }
  }, [market, depth, side, form.price]);

  const handlePercent = (pct) => {
    if (!form.price) return;
    const price = parseNumber(form.price);
    if (price <= 0) return;

    if (side === "BUY") {
      const availableUSD = balances["USD"] * (pct / 100);
      const qty = availableUSD / price;
      set("quantity", qty > 0 ? formatInputValue(qty, true) : "");
    } else {
      const availableCrypto = balances[crypto] * (pct / 100);
      const qty = availableCrypto;
      set("quantity", qty > 0 ? formatInputValue(qty, true) : "");
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!form.price || !form.quantity) return;
    setLoading(true);
    try {
      const parsedPrice = parseNumber(form.price);
      const parsedQty = parseNumber(form.quantity);
      
      const payload = {
        user_id: user.user_id,
        symbol: market,
        side,
        price: Math.floor(parsedPrice * 10000),
        quantity: Math.floor(parsedQty * 10000),
      };
      await submitOrder(payload);
      set("quantity", ""); // Sadece miktarı sıfırla
    } catch(err) {
      alert("Hata: " + err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = (k, val) => {
    // Sadece sayılara, virgüle ve noktaya izin ver
    const cleaned = val.replace(/[^0-9.,]/g, '');
    set(k, cleaned);
  };

  return (
    <div className="panel">
      <div className="panel-header" style={{ display: 'flex', justifyContent: 'space-between' }}>
        <span>Emir Gönder</span>
        <div style={{ fontSize: '11px', color: '#848e9c' }}>
          Cüzdan: <span style={{color: '#fff'}}>{balances["USD"]?.toLocaleString() || 0} USD</span> / <span style={{color: '#fff'}}>{balances[crypto] || 0} {crypto}</span>
        </div>
      </div>
      
      <div className="side-toggle" style={{ margin: '16px 16px 0' }}>
        <button
          type="button"
          className={`side-btn buy${side === "BUY" ? " active" : ""}`}
          onClick={() => toggleSide("BUY")}
        >
          AL
        </button>
        <button
          type="button"
          className={`side-btn sell${side === "SELL" ? " active" : ""}`}
          onClick={() => toggleSide("SELL")}
        >
          SAT
        </button>
      </div>

      <form className="order-form-content" onSubmit={handleSubmit}>
        <div className="input-group">
          <span className="input-label">Fiyat</span>
          <input type="text" inputMode="decimal" value={form.price} onChange={e => handleInputChange("price", e.target.value)} required />
          <span className="input-suffix">USD</span>
        </div>
        <div className="input-group">
          <span className="input-label">Miktar</span>
          <input type="text" inputMode="decimal" value={form.quantity} onChange={e => handleInputChange("quantity", e.target.value)} required />
          <span className="input-suffix">{crypto}</span>
        </div>
        
        <div className="percent-slider">
          {[25, 50, 75, 100].map(pct => (
            <button key={pct} type="button" className="pct-btn" onClick={() => handlePercent(pct)}>{pct}%</button>
          ))}
        </div>

        <button type="submit" className={`btn-action ${side === "BUY" ? "btn-buy" : "btn-sell"}`} disabled={loading} style={{ marginTop: '8px' }}>
          {side === "BUY" ? `${crypto} AL` : `${crypto} SAT`}
        </button>
      </form>
    </div>
  );
}
