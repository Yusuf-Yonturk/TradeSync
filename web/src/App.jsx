import { useState } from "react";
import { useMarketData } from "./hooks/useMarketData";
import AuthScreen from "./components/AuthScreen/AuthScreen";
import TopBar from "./components/TopBar/TopBar";
import OrderBookPanel from "./components/OrderBook/OrderBookPanel";
import PriceChart from "./components/PriceChart/PriceChart";
import OrderForm from "./components/OrderForm/OrderForm";
import TradeListPanel from "./components/TradeList/TradeListPanel";
import SettingsModal from "./components/Modals/SettingsModal";
import WalletModal from "./components/Modals/WalletModal";

export default function App() {
  const [user, setUser] = useState(() => {
    const saved = localStorage.getItem("ts_user");
    if (saved === "undefined") return null;
    return saved ? JSON.parse(saved) : null;
  });

  const [market, setMarket] = useState("BTC-USD");
  const [showSettings, setShowSettings] = useState(false);
  const [showWallet, setShowWallet] = useState(false);

  const [balances, setBalances] = useState({ USD: 100000, BTC: 10, ETH: 100, SOL: 500 });

  const { trades, depth, klines, isConnected, tickerPrices } = useMarketData(market, user);

  const handleLogin = (userData) => {
    setUser(userData);
    localStorage.setItem("ts_user", JSON.stringify(userData));
  };

  const handleLogout = () => {
    setUser(null);
    localStorage.removeItem("ts_user");
  };

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
        <OrderBookPanel depth={depth} lastPrice={trades[0]?.price} />
        <PriceChart market={market} klines={klines} latestTrade={trades[0]} />
        <div className="right-col">
          <OrderForm user={user} market={market} depth={depth} balances={balances} />
          <TradeListPanel trades={trades} user={user} market={market} />
        </div>
      </div>

      {showSettings && <SettingsModal onClose={() => setShowSettings(false)} user={user} setUser={setUser} />}
      {showWallet && <WalletModal onClose={() => setShowWallet(false)} balances={balances} setBalances={setBalances} setMarket={setMarket} />}
    </div>
  );
}
