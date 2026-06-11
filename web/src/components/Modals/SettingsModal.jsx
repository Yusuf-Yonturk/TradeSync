import { useState } from 'react';
import './Modals.css';

export default function SettingsModal({ onClose, user, setUser }) {
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
