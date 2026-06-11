import './Modals.css';

export default function WalletModal({ onClose, balances, setBalances, setMarket }) {
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
