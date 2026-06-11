import { formatPrice } from '../../utils/format';
import ProfileDropdown from './ProfileDropdown';
import './TopBar.css';

export default function TopBar({
  market,
  setMarket,
  isConnected,
  onOpenSettings,
  onOpenWallet,
  onLogout,
  user,
  balances,
  tickerPrices
}) {
  return (
    <div className="top-bar">
      <div className="logo-area">
        <span className="logo-text">TradeSync</span>
      </div>
      <div className="ticker-tape">
        <div className="ticker-item">
          <span className="muted">BTC-USD</span>{' '}
          <span className="green">
            {tickerPrices['BTC-USD'] ? formatPrice(tickerPrices['BTC-USD']) : '15.000,00'}
          </span>
        </div>
        <div className="ticker-item">
          <span className="muted">ETH-USD</span>{' '}
          <span className="red">
            {tickerPrices['ETH-USD'] ? formatPrice(tickerPrices['ETH-USD']) : '3.000,00'}
          </span>
        </div>
        <div className="ticker-item">
          <span className="muted">SOL-USD</span>{' '}
          <span className="green">
            {tickerPrices['SOL-USD'] ? formatPrice(tickerPrices['SOL-USD']) : '145,20'}
          </span>
        </div>
      </div>
      <div className="status-area">
        <div className="latency">
          <div className="latency-dot" style={{ background: isConnected ? 'var(--buy)' : 'var(--sell)' }}></div>
          <span>12ms</span>
        </div>
        <select
          className="market-select"
          value={market}
          onChange={(e) => setMarket(e.target.value)}
        >
          <option value="BTC-USD">BTC-USD</option>
          <option value="ETH-USD">ETH-USD</option>
          <option value="SOL-USD">SOL-USD</option>
        </select>
        <div className="user-settings" onClick={onOpenSettings} style={{ marginRight: '8px' }}>
          ⚙️ Ayarlar
        </div>
        {user && balances && (
          <ProfileDropdown
            user={user}
            balances={balances}
            onLogout={onLogout}
            onOpenSettings={onOpenSettings}
            onOpenWallet={onOpenWallet}
          />
        )}
      </div>
    </div>
  );
}
