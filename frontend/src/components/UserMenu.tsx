import { useEffect, useRef, useState } from 'react'
import { useAuth } from '../auth'

function roleLabel(role: string) {
  switch (role) {
    case 'admin': return '管理员'
    case 'operator': return '运维'
    case 'viewer': return '只读'
    default: return role
  }
}

export function UserMenu() {
  const { user, logout } = useAuth()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    if (open) document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  if (!user) return null

  return (
    <div className="user-menu" ref={ref}>
      <button
        type="button"
        className="user-menu-trigger"
        onClick={() => setOpen(v => !v)}
        aria-expanded={open}
        aria-haspopup="menu"
      >
        <div className="user-avatar">{user.username.charAt(0).toUpperCase()}</div>
        <span className="user-menu-name">{user.username}</span>
        <span className={`user-menu-chevron${open ? ' open' : ''}`} aria-hidden>▾</span>
      </button>

      {open && (
        <div className="user-menu-dropdown" role="menu">
          <div className="user-menu-header">
            <div className="user-avatar user-avatar-lg">{user.username.charAt(0).toUpperCase()}</div>
            <div className="user-menu-meta">
              <strong>{user.username}</strong>
              <span className={`role-badge role-${user.role}`}>{roleLabel(user.role)}</span>
            </div>
          </div>

          <div className="user-menu-divider" />

          <button
            type="button"
            role="menuitem"
            className="user-menu-item user-menu-logout"
            onClick={() => { setOpen(false); logout() }}
          >
            退出登录
          </button>
        </div>
      )}
    </div>
  )
}
