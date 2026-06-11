import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { fetchTrades, fetchDepth, fetchKlines, submitOrder, login, register } from "./api";
import { createChart, CandlestickSeries } from "lightweight-charts";

function formatTime(t) {
  const d = new Date(t);
  const hh = d.getHours().toString().padStart(2, '0');
  const mm = d.getMinutes().toString().padStart(2, '0');
  const ss = d.getSeconds().toString().padStart(2, '0');
  const ms = d.getMilliseconds().toString().padStart(3, '0');
  return `${hh}:${mm}:${ss}:${ms}`;
}

function AuthScreen({ onLoginSuccess }) {
  const [isLogin, setIsLogin] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    const fd = new FormData(e.target);
    const username = fd.get("u");
    const password = fd.get("p");
    
    try {
      if (isLogin) {
        const res = await login(username, password);
        onLoginSuccess(res);
      } else {
        const name = fd.get("n");
        const res = await register(name, username, password);
        onLoginSuccess(res);
      }
    } catch(err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="app-container" style={{ justifyContent: 'center', alignItems: 'center' }}>
      <div className="auth-overlay">
        <div className="auth-card premium-glass">
          <div className="auth-logo">TradeSync<span className="pro-badge">PRO</span></div>
          <h2>{isLogin ? "Hoş Geldiniz" : "Hesap Oluşturun"}</h2>
          <p className="auth-subtitle">{isLogin ? "İşlem yapmaya başlamak için giriş yapın" : "TradeSync dünyasına katılın"}</p>
          
          {error && <div className="auth-error">{error}</div>}
          
          <form onSubmit={handleSubmit} className="auth-form">
            {!isLogin && (
              <div className="input-group-auth">
                <label>İsim Soyisim</label>
                <input name="n" placeholder="Adınız" required />
              </div>
            )}
            <div className="input-group-auth">
              <label>Kullanıcı Adı</label>
              <input name="u" placeholder="Kullanıcı adınız" required />
            </div>
            <div className="input-group-auth">
              <label>Şifre</label>
              <input name="p" type="password" placeholder="••••••••" required />
            </div>
            <button type="submit" className="submit btn-action btn-buy auth-btn" disabled={loading}>
              {loading ? "Yükleniyor..." : (isLogin ? "Giriş Yap" : "Kayıt Ol")}
            </button>
          </form>
          
          <div className="auth-switch">
            {isLogin ? "Hesabınız yok mu?" : "Zaten hesabınız var mı?"} 
            <button type="button" onClick={() => { setIsLogin(!isLogin); setError(""); }} className="btn-link">
              {isLogin ? "Kayıt Ol" : "Giriş Yap"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
const formatPrice = (p) => Number(p / 10000).toLocaleString("tr-TR", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
const formatQty = (q) => Number(q / 10000).toLocaleString("tr-TR", { maximumFractionDigits: 4 });

function TopBar({ market, setMarket, isConnected, onOpenSettings, onOpenWallet, onLogout, user, balances, tickerPrices }) {
  return (
    <div className="top-bar">
      <div className="logo-area">
        <span className="logo-text">TradeSync</span>
      </div>
      <div className="ticker-tape">
        <div className="ticker-item"><span className="muted">BTC-USD</span> <span className="green">{tickerPrices['BTC-USD'] ? formatPrice(tickerPrices['BTC-USD']) : '15.000,00'}</span></div>
        <div className="ticker-item"><span className="muted">ETH-USD</span> <span className="red">{tickerPrices['ETH-USD'] ? formatPrice(tickerPrices['ETH-USD']) : '3.000,00'}</span></div>
        <div className="ticker-item"><span className="muted">SOL-USD</span> <span className="green">{tickerPrices['SOL-USD'] ? formatPrice(tickerPrices['SOL-USD']) : '145,20'}</span></div>
      </div>
      <div className="status-area">
        <div className="latency">
          <div className="latency-dot"></div>
          <span>12ms</span>
        </div>
        <select 
          className="market-select"
          value={market} 
          onChange={e => setMarket(e.target.value)}
        >
          <option value="BTC-USD">BTC-USD</option>
          <option value="ETH-USD">ETH-USD</option>
          <option value="SOL-USD">SOL-USD</option>
        </select>
        <div className="user-settings" onClick={onOpenSettings} style={{ marginRight: '8px' }}>
          ⚙️ Ayarlar
        </div>
        {user && balances && (
          <ProfileDropdown user={user} balances={balances} onLogout={onLogout} onOpenSettings={onOpenSettings} onOpenWallet={onOpenWallet} />
        )}
      </div>
    </div>
  );
}

function OrderBookPanel({ depth, lastPrice }) {
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

function PriceChart({ market, klines, latestTrade }) {
  const chartContainerRef = useRef();
  const chartRef = useRef();
  const candleSeriesRef = useRef();

  useEffect(() => {
    try {
      if (!chartContainerRef.current) return;
      
      const chart = createChart(chartContainerRef.current, {
        autoSize: true,
        layout: {
          background: { type: 'solid', color: '#121212' },
          textColor: '#848e9c',
        },
        grid: {
          vertLines: { color: '#2a2a2a' },
          horzLines: { color: '#2a2a2a' },
        },
        timeScale: {
          timeVisible: true,
          secondsVisible: false,
        },
        crosshair: {
          mode: 0,
        },
      });

      const series = chart.addSeries(CandlestickSeries, {
        upColor: '#02c076',
        downColor: '#f84960',
        borderVisible: false,
        wickUpColor: '#02c076',
        wickDownColor: '#f84960',
      });
      
      candleSeriesRef.current = series;
      chartRef.current = chart;

      return () => {
        chart.remove();
        chartRef.current = null;
      };
    } catch(err) {
      console.error("Chart init error:", err);
    }
  }, []);

  // Update data when klines change
  useEffect(() => {
    try {
      if (candleSeriesRef.current && klines.length > 0) {
        candleSeriesRef.current.setData(klines);
        if (chartRef.current) {
          chartRef.current.timeScale().fitContent();
        }
      }
    } catch(err) {
      console.error("Chart setData error:", err);
    }
  }, [klines]);

  // Update latest candle with latest trade
  useEffect(() => {
    try {
      if (candleSeriesRef.current && latestTrade && latestTrade.symbol === market) {
        const ts = Math.floor(new Date(latestTrade.traded_at || latestTrade.timestamp).getTime() / 1000);
        const minute = ts - (ts % 60);
        
        let lastCandle = null;
        if (klines && klines.length > 0) {
          lastCandle = klines[klines.length - 1];
        }

        if (lastCandle && lastCandle.time === minute) {
          candleSeriesRef.current.update({
            time: minute,
            open: lastCandle.open / 10000,
            high: Math.max(lastCandle.high, latestTrade.price) / 10000,
            low: Math.min(lastCandle.low, latestTrade.price) / 10000,
            close: latestTrade.price / 10000
          });
        } else {
          // If no previous candle, lightweight charts sometimes errors on update.
          // In that case, we can fetch klines again or ignore it until polling catches up.
          candleSeriesRef.current.update({
            time: minute,
            open: latestTrade.price / 10000,
            high: latestTrade.price / 10000,
            low: latestTrade.price / 10000,
            close: latestTrade.price / 10000
          });
        }
      }
    } catch(err) {
      console.error("Chart update error:", err);
    }
  }, [latestTrade, market, klines]);

  return (
    <div className="panel" style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
      <div className="panel-header">{market} / Chart</div>
      <div className="chart-container" ref={chartContainerRef} style={{ flex: 1, minHeight: '400px', width: '100%', position: 'relative' }}>
      </div>
    </div>
  );
}

function ProfileDropdown({ user, balances, onLogout, onOpenSettings, onOpenWallet }) {
  const [open, setOpen] = useState(false);
  
  return (
    <div className="profile-container" onMouseLeave={() => setOpen(false)}>
      <div className="user-profile" onClick={() => setOpen(!open)}>
        <div className="avatar">
           <img src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${user.display_name || user.user_id}`} alt="avatar" />
        </div>
        <span>{user.display_name || user.user_id.split('-')[0]}</span>
      </div>
      
      {open && (
        <div className="profile-dropdown menu-dropdown">
          <button className="dropdown-menu-item" onClick={() => { setOpen(false); onOpenWallet(); }}>Cüzdan</button>
          <button className="dropdown-menu-item" onClick={() => { setOpen(false); onOpenSettings(); }}>Ayarlar</button>
          <button className="dropdown-menu-item logout" onClick={onLogout}>Çıkış Yap</button>
        </div>
      )}
    </div>
  );
}

function OrderForm({ user, market, depth, balances }) {
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
  }, [market, depth, side]);

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
          Cüzdan: <span style={{color: '#fff'}}>{balances["USD"].toLocaleString()} USD</span> / <span style={{color: '#fff'}}>{balances[crypto]} {crypto}</span>
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

function TradeListPanel({ trades, user }) {
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

function SettingsModal({ onClose, user, setUser }) {
  const [newName, setNewName] = useState(user.display_name || user.user_id.split('-')[0]);

  const handleSaveName = () => {
    const updatedUser = { ...user, display_name: newName };
    setUser(updatedUser);
    localStorage.setItem("ts_user", JSON.stringify(updatedUser));
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <span>Ayarlar</span>
          <button className="modal-close" onClick={onClose}>&times;</button>
        </div>
        <div className="modal-body">
          <div className="setting-row" style={{ flexDirection: 'column', alignItems: 'flex-start', gap: '8px' }}>
            <span>Görünen İsim</span>
            <div style={{ display: 'flex', gap: '8px', width: '100%' }}>
              <input 
                value={newName} 
                onChange={e => setNewName(e.target.value)} 
                className="ts-input"
                style={{ flex: 1 }}
              />
              <button className="btn-action btn-buy" style={{ flex: 'none', width: '80px', height: '36px' }} onClick={handleSaveName}>Kaydet</button>
            </div>
          </div>
          <div className="setting-row">
            <span>Pro Layout / Simple Layout</span>
            <label className="switch">
              <input type="checkbox" defaultChecked />
              <span className="slider"></span>
            </label>
          </div>
          <div className="setting-row">
            <span>Bildirim Sesleri</span>
            <label className="switch">
              <input type="checkbox" />
              <span className="slider"></span>
            </label>
          </div>
        </div>
      </div>
    </div>
  );
}

function WalletModal({ onClose, balances, setBalances, setMarket }) {
  const handleAdd = (coin) => {
    setBalances(prev => ({ ...prev, [coin]: prev[coin] + (coin === 'USD' ? 1000 : 1) }));
  };
  const handleRemove = (coin) => {
    setBalances(prev => ({ ...prev, [coin]: Math.max(0, prev[coin] - (coin === 'USD' ? 1000 : 1)) }));
  };
  
  const navigateToCoin = (coin) => {
    if (coin !== 'USD') {
      setMarket(`${coin}-USD`);
      onClose();
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content wallet-modal" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <span>Cüzdan Yönetimi</span>
          <button className="modal-close" onClick={onClose}>&times;</button>
        </div>
        <div className="modal-body wallet-body">
          <div className="wallet-list-header">
            <span style={{ width: '160px' }}>Varlık</span>
            <span style={{ flex: 1, textAlign: 'right', paddingRight: '24px' }}>Bakiye</span>
            <span style={{ width: '80px', textAlign: 'center' }}>Ekle/Çıkar</span>
          </div>
          <div className="wallet-balances-list">
            {Object.entries(balances).map(([coin, amount]) => (
              <div key={coin} className="wallet-item">
                <div 
                  className="coin-info" 
                  onClick={() => navigateToCoin(coin)} 
                  style={{ cursor: coin !== 'USD' ? 'pointer' : 'default' }}
                >
                  <div className={`coin-icon ${coin.toLowerCase()}`}>{coin.charAt(0)}</div>
                  <div className="coin-details">
                    <span className="coin-symbol">{coin}</span>
                    {coin !== 'USD' && <span className="coin-trade-link">Piyasaya Git &rarr;</span>}
                  </div>
                </div>
                <div className="coin-amount-box">
                  <span className="amount-val">{amount.toLocaleString()}</span>
                  <span className="amount-cur">{coin}</span>
                </div>
                <div className="wallet-actions">
                  <button className="btn-add" onClick={() => handleAdd(coin)}>+</button>
                  <button className="btn-sub" onClick={() => handleRemove(coin)}>-</button>
                </div>
              </div>
            ))}
          </div>
          <div className="wallet-usd-action" style={{ marginTop: '24px', borderTop: '1px solid var(--border)', paddingTop: '16px' }}>
            <h4 style={{ marginBottom: '12px', color: 'var(--text-muted)', fontSize: '12px', textTransform: 'uppercase', letterSpacing: '1px' }}>Dolar Al / Sat</h4>
            <div className="wallet-actions" style={{ justifyContent: 'flex-start', gap: '12px' }}>
              <button className="btn-buy" style={{ padding: '8px 16px', border: 'none', borderRadius: '4px', color: '#fff', cursor: 'pointer', width: 'auto', height: 'auto', fontWeight: '500' }} onClick={() => handleAdd('USD')}>+ 1000 USD Al</button>
              <button className="btn-sell" style={{ padding: '8px 16px', border: 'none', borderRadius: '4px', color: '#fff', cursor: 'pointer', width: 'auto', height: 'auto', fontWeight: '500' }} onClick={() => handleRemove('USD')}>- 1000 USD Sat</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function App() {
  const [user, setUser] = useState(() => {
    const saved = localStorage.getItem("ts_user");
    if (saved === "undefined") return null; return saved ? JSON.parse(saved) : null;
  });

  const [trades, setTrades] = useState([]);
  const [klines, setKlines] = useState([]);
  const [depth, setDepth] = useState({ bids: [], asks: [] });
  const [market, setMarket] = useState("BTC-USD");
  const [isConnected, setIsConnected] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [showWallet, setShowWallet] = useState(false);

  const [balances, setBalances] = useState({ "USD": 100000, "BTC": 10, "ETH": 100, "SOL": 500 });

  const filteredTrades = useMemo(() => trades.filter(t => t.symbol === market), [trades, market]);

  const handleLogin = (userData) => {
    setUser(userData);
    localStorage.setItem("ts_user", JSON.stringify(userData));
  };

  const handleLogout = () => {
    setUser(null);
    localStorage.removeItem("ts_user");
  };

  const loadTrades = useCallback(async () => {
    try {
      const data = await fetchTrades(market);
      setTrades(data);
    } catch {
      // sessiz
    }
  }, [market]);

  useEffect(() => {
    loadTrades();
  }, [loadTrades]);

  const [tickerPrices, setTickerPrices] = useState({});

  const loadDepth = useCallback(async () => {
    if (!market) return;
    try {
      const data = await fetchDepth(market);
      setDepth(data);
    } catch {
      // sessiz
    }
  }, [market]);

  const loadKlines = useCallback(async () => {
    if (!market) return;
    try {
      const data = await fetchKlines(market);
      setKlines(data || []);
    } catch {
      // sessiz
    }
  }, [market]);

  useEffect(() => {
    loadDepth();
    loadKlines();
  }, [loadDepth, loadKlines]);

  useEffect(() => {
    if (!user) return;
    const ws = new WebSocket("wss://api.yusufyonturk.com/tradesync/gateway/ws");
    ws.onopen = () => setIsConnected(true);
    ws.onclose = () => setIsConnected(false);
    ws.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        const sym = data.Symbol || data.symbol;
        
        if (data.price !== undefined && data.quantity !== undefined) {
           setTickerPrices(prev => ({ ...prev, [sym]: data.price }));
        }

        if (sym === market) {
          // Check if it's depth data (has Bids/Asks)
          if (data.Bids !== undefined || data.Asks !== undefined || data.bids !== undefined || data.asks !== undefined) {
            setDepth({
              bids: data.Bids || data.bids || [],
              asks: data.Asks || data.asks || []
            });
          } else if (data.price !== undefined && data.quantity !== undefined) {
            // It's a trade
            setTrades((prev) => [data, ...prev].slice(0, 100));
          }
        }
      } catch {}
    };
    return () => ws.close();
  }, [user, market]);


  if (!user) {
    return <AuthScreen onLoginSuccess={handleLogin} />;
  }

  return (
    <div className="app-container">

      <TopBar 
        market={market} 
        setMarket={setMarket} 
        isConnected={isConnected} 
        onOpenSettings={() => setShowSettings(true)}
        onOpenWallet={() => setShowWallet(true)}
        user={user}
        balances={balances}
        onLogout={handleLogout}
        tickerPrices={tickerPrices}
      />
      
      <div className="dashboard">
        <OrderBookPanel depth={depth} lastPrice={filteredTrades[0]?.price} />
        <PriceChart market={market} klines={klines} latestTrade={filteredTrades[0]} />
        <div className="right-col">
          <OrderForm user={user} market={market} depth={depth} balances={balances} />
          <TradeListPanel trades={filteredTrades} user={user} />
        </div>
      </div>

      {showSettings && <SettingsModal onClose={() => setShowSettings(false)} user={user} setUser={setUser} />}
      {showWallet && <WalletModal onClose={() => setShowWallet(false)} balances={balances} setBalances={setBalances} setMarket={setMarket} />}
    </div>
  );
}
