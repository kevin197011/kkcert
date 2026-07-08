import { NavLink } from 'react-router-dom'
import { useAuth } from '../auth'
import { canManageUsers, canWriteSettings } from '../api'
import {
  IconDashboard, IconCertificate, IconSettings,
  IconUsers, IconKey, IconLogs,
} from '../icons'
import { Logo } from '../Logo'

const NAV: {
  to: string
  label: string
  icon: React.ReactNode
  end?: boolean
  show?: (role: string) => boolean
}[] = [
  { to: '/', label: '概览', icon: <IconDashboard />, end: true },
  { to: '/domains', label: '域名与证书', icon: <IconCertificate /> },
  { to: '/logs', label: '操作日志', icon: <IconLogs /> },
  { to: '/tokens', label: 'API Token', icon: <IconKey />, show: canManageUsers },
  { to: '/users', label: '用户管理', icon: <IconUsers />, show: canManageUsers },
  { to: '/settings', label: '系统设置', icon: <IconSettings />, show: canWriteSettings },
]

function NavItem({ to, end, icon, children }: { to: string; end?: boolean; icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <NavLink to={to} end={end} className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}>
      <span className="nav-icon">{icon}</span>
      <span>{children}</span>
    </NavLink>
  )
}

export function Sidebar() {
  const { user } = useAuth()
  const role = user!.role
  const items = NAV.filter(n => !n.show || n.show(role))

  return (
    <aside className="sidebar">
      <div className="sidebar-glow sidebar-glow-top" aria-hidden />
      <div className="sidebar-glow sidebar-glow-bottom" aria-hidden />

      <div className="brand">
        <div className="brand-icon">
          <Logo />
        </div>
        <div className="brand-text">
          <h1>KKCert</h1>
          <span>证书运维</span>
        </div>
      </div>

      <nav className="sidebar-nav" aria-label="证书运维">
        <div className="nav-section">
          <span className="nav-section-label">证书运维</span>
          {items.map(n => (
            <NavItem key={n.to} to={n.to} end={n.end} icon={n.icon}>
              {n.label}
            </NavItem>
          ))}
        </div>
      </nav>
    </aside>
  )
}
