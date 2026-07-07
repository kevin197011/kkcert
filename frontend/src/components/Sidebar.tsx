import { NavLink } from 'react-router-dom'
import { useAuth } from '../auth'
import { canManageUsers, canWriteSettings } from '../api'
import {
  IconDashboard, IconCertificate, IconSettings,
  IconUsers, IconKey, IconLogs,
} from '../icons'
import { Logo } from '../Logo'

function NavItem({ to, end, icon, children }: { to: string; end?: boolean; icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <NavLink to={to} end={end} className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}>
      <span className="nav-icon">{icon}</span>
      <span>{children}</span>
    </NavLink>
  )
}

function NavSection({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="nav-section">
      <span className="nav-section-label">{label}</span>
      {children}
    </div>
  )
}

export function Sidebar() {
  const { user } = useAuth()
  const showSettings = canWriteSettings(user!.role)
  const showAdmin = canManageUsers(user!.role)
  const showSystem = showSettings || showAdmin

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
        <NavSection label="证书运维">
          <NavItem to="/" end icon={<IconDashboard />}>概览</NavItem>
          <NavItem to="/domains" icon={<IconCertificate />}>域名与证书</NavItem>
        </NavSection>
        <NavSection label="观测">
          <NavItem to="/logs" icon={<IconLogs />}>操作日志</NavItem>
        </NavSection>
      </nav>

      {showSystem && (
        <nav className="sidebar-nav sidebar-nav-bottom" aria-label="系统管理">
          <NavSection label="系统">
            {showSettings && (
              <NavItem to="/settings" icon={<IconSettings />}>系统设置</NavItem>
            )}
            {showAdmin && (
              <>
                <NavItem to="/users" icon={<IconUsers />}>用户管理</NavItem>
                <NavItem to="/tokens" icon={<IconKey />}>API Token</NavItem>
              </>
            )}
          </NavSection>
        </nav>
      )}
    </aside>
  )
}
