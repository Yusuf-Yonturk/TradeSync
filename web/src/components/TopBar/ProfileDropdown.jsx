import { useState } from 'react';

export default function ProfileDropdown({ user, balances, onLogout, onOpenSettings, onOpenWallet }) {
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
