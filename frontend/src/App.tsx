import { NavLink, Navigate, Route, Routes } from 'react-router-dom'
import { useAuth } from './auth'
import { canManageUsers, canWriteDomain, canWriteSettings } from './api'
import { UserMenu } from './components/UserMenu'
import { AnimatedBackground } from './components/AnimatedBackground'
import { PageFooter } from './components/PageFooter'
import {
  IconDashboard, IconGlobe, IconCertificate, IconSettings,
  IconUsers, IconKey, IconLogs,
} from './icons'
import { Logo } from './Logo'
import Dashboard from './pages/Dashboard'
import Domains from './pages/Domains'
import Certificates from './pages/Certificates'
import Settings from './pages/Settings'
import Logs from './pages/Logs'
import Users from './pages/Users'
import Tokens from './pages/Tokens'
import Login from './pages/Login'

function RequireAuth({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return <p className="loading">加载中...</p>
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

function NavItem({ to, end, icon, children }: { to: string; end?: boolean; icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <NavLink to={to} end={end} className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}>
      <span className="nav-icon">{icon}</span>
      <span>{children}</span>
    </NavLink>
  )
}

function Layout() {
  const { user } = useAuth()

  return (
    <div className="layout">
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

        <nav className="sidebar-nav">
          <div className="nav-section">
            <span className="nav-section-label">监控</span>
            <NavItem to="/" end icon={<IconDashboard />}>概览</NavItem>
            <NavItem to="/certificates" icon={<IconCertificate />}>证书列表</NavItem>
            <NavItem to="/logs" icon={<IconLogs />}>操作日志</NavItem>
          </div>

          <div className="nav-section">
            <span className="nav-section-label">管理</span>
            <NavItem to="/domains" icon={<IconGlobe />}>域名管理</NavItem>
            {canWriteSettings(user!.role) && (
              <NavItem to="/settings" icon={<IconSettings />}>系统设置</NavItem>
            )}
            {canManageUsers(user!.role) && (
              <>
                <NavItem to="/users" icon={<IconUsers />}>用户管理</NavItem>
                <NavItem to="/tokens" icon={<IconKey />}>API Token</NavItem>
              </>
            )}
          </div>
        </nav>
      </aside>

      <div className="main-shell">
        <AnimatedBackground className="ambient-bg-app" />
        <header className="topbar">
          <UserMenu />
        </header>
        <main className="main">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/domains" element={<Domains />} />
            <Route path="/certificates" element={<Certificates />} />
            {canWriteSettings(user!.role) && <Route path="/settings" element={<Settings />} />}
            {canManageUsers(user!.role) && <Route path="/users" element={<Users />} />}
            {canManageUsers(user!.role) && <Route path="/tokens" element={<Tokens />} />}
            <Route path="/logs" element={<Logs />} />
          </Routes>
        </main>
        <PageFooter />
      </div>
    </div>
  )
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/*" element={<RequireAuth><Layout /></RequireAuth>} />
    </Routes>
  )
}

export { canWriteDomain }
