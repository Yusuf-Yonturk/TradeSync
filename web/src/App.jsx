import { useState, useCallback, useEffect, useMemo } from "react";
import { useMarketData } from "./hooks/useMarketData";
import AuthScreen from "./components/AuthScreen/AuthScreen";
import TopBar from "./components/TopBar/TopBar";
import OrderBookPanel from "./components/OrderBook/OrderBookPanel";
import PriceChart from "./components/PriceChart/PriceChart";
import OrderForm from "./components/OrderForm/OrderForm";
import TradeListPanel from "./components/TradeList/TradeListPanel";
import SettingsModal from "./components/Modals/SettingsModal";
import WalletModal from "./components/Modals/WalletModal";
import { fetchBalances, AuthError } from "./api";

// Bakiye emirler ve eşleşmelerle değiştiği için bu aralıkla yenilenir
const BALANCE_REFRESH_MS = 3000;

export default function App() {
  const [user, setUser] = useState(() => {
    try {
      const saved = JSON.parse(localStorage.getItem("ts_user"));
      // Token'siz eski kayitlar emir gonderemez, yeniden giris gerekir.
      return saved?.user_id && saved?.token ? saved : null;
    } catch {
      return null;
    }
  });

  const [market, setMarket] = useState("BTC-USD");
  const [showSettings, setShowSettings] = useState(false);
  const [showWallet, setShowWallet] = useState(false);

  // Sunucudan gelen bakiye satırları: { asset, available, locked } (birim cinsinden)
  const [wallet, setWallet] = useState([]);

  const { trades, depth, klines, isConnected, tickerPrices } = useMarketData(market, user);

  const handleLogin = (userData) => {
    setUser(userData);
    localStorage.setItem("ts_user", JSON.stringify(userData));
  };

  // Alt bileşenlerin effect'leri her render'da yeniden kurulmasın diye sabit referans
  const handleLogout = useCallback(() => {
    setUser(null);
    setWallet([]);
    localStorage.removeItem("ts_user");
  }, []);

  const token = user?.token;

  const loadWallet = useCallback(() => {
    if (!token) return;
    fetchBalances(token)
      .then((rows) => {
        setWallet(rows.map((r) => ({
          asset: r.asset,
          available: r.available / 10000,
          locked: r.locked / 10000,
        })));
      })
      .catch((err) => {
        if (err instanceof AuthError) handleLogout();
      });
  }, [token, handleLogout]);

  useEffect(() => {
    if (!token) return;
    loadWallet();
    const timer = setInterval(loadWallet, BALANCE_REFRESH_MS);
    return () => clearInterval(timer);
  }, [token, loadWallet]);

  // Emir formu yalnızca kullanılabilir tutarla ilgilenir
  const balances = useMemo(
    () => Object.fromEntries(wallet.map((w) => [w.asset, w.available])),
    [wallet]
  );

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
        onLogout={handleLogout}
        tickerPrices={tickerPrices}
      />
      
      <div className="dashboard">
        <OrderBookPanel depth={depth} lastPrice={trades[0]?.price} />
        <PriceChart market={market} klines={klines} latestTrade={trades[0]} />
        <div className="right-col">
          <OrderForm key={market} user={user} market={market} depth={depth} balances={balances} onSessionExpired={handleLogout} onOrderPlaced={loadWallet} />
          <TradeListPanel trades={trades} user={user} market={market} onSessionExpired={handleLogout} />
        </div>
      </div>

      {showSettings && <SettingsModal onClose={() => setShowSettings(false)} user={user} setUser={setUser} />}
      {showWallet && <WalletModal onClose={() => setShowWallet(false)} wallet={wallet} setMarket={setMarket} />}
    </div>
  );
}
