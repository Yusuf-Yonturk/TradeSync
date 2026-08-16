import './Modals.css';

const fmt = (n) => n.toLocaleString("tr-TR", { maximumFractionDigits: 4 });

// Bakiyeler sunucuda tutulur; burada yalnızca gösterilir.
export default function WalletModal({ onClose, wallet, setMarket }) {
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
          <span>Cüzdan</span>
          <button className="modal-close" onClick={onClose}>&times;</button>
        </div>
        <div className="modal-body wallet-body">
          <div className="wallet-list-header">
            <span style={{ width: '160px' }}>Varlık</span>
            <span style={{ flex: 1, textAlign: 'right', paddingRight: '24px' }}>Kullanılabilir</span>
            <span style={{ width: '120px', textAlign: 'right' }}>Emirde</span>
          </div>
          <div className="wallet-balances-list">
            {wallet.length === 0 && <div className="wallet-item">Bakiye yükleniyor...</div>}
            {wallet.map(({ asset: coin, available, locked }) => (
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
                  <span className="amount-val">{fmt(available)}</span>
                  <span className="amount-cur">{coin}</span>
                </div>
                <div style={{ width: '120px', textAlign: 'right', color: 'var(--text-muted)' }}>{fmt(locked)}</div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
