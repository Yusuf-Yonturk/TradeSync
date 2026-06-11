import { useState } from 'react';
import { login, register } from '../../api';
import './AuthScreen.css';

export default function AuthScreen({ onLoginSuccess }) {
  const [isLogin, setIsLogin] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError]     = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    const fd       = new FormData(e.target);
    const username = fd.get('u');
    const password = fd.get('p');

    try {
      if (isLogin) {
        const res = await login(username, password);
        onLoginSuccess(res);
      } else {
        const name = fd.get('n');
        const res  = await register(name, username, password);
        onLoginSuccess(res);
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const switchMode = () => { setIsLogin(!isLogin); setError(''); };

  return (
    <div className="auth-overlay">
      <div className="auth-card premium-glass">
        <div className="auth-logo">
          TradeSync<span className="pro-badge">PRO</span>
        </div>

        <h2>{isLogin ? 'Hoş Geldiniz' : 'Hesap Oluşturun'}</h2>
        <p className="auth-subtitle">
          {isLogin ? 'İşlem yapmaya başlamak için giriş yapın' : 'TradeSync dünyasına katılın'}
        </p>

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
          <button
            type="submit"
            className="btn-action btn-buy auth-btn"
            disabled={loading}
          >
            {loading ? 'Yükleniyor...' : isLogin ? 'Giriş Yap' : 'Kayıt Ol'}
          </button>
        </form>

        <div className="auth-switch">
          {isLogin ? 'Hesabınız yok mu?' : 'Zaten hesabınız var mı?'}
          <button type="button" onClick={switchMode} className="btn-link">
            {isLogin ? 'Kayıt Ol' : 'Giriş Yap'}
          </button>
        </div>
      </div>
    </div>
  );
}
