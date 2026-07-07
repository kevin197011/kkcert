import { Navigate, Route, Routes } from 'react-router-dom'
import { useAuth } from './auth'
import { canManageUsers, canWriteSettings } from './api'
import { ThemeToggle } from './theme'
import { UserMenu } from './components/UserMenu'
import { AnimatedBackground } from './components/AnimatedBackground'
import { PageFooter } from './components/PageFooter'
import { Sidebar } from './components/Sidebar'
import Dashboard from './pages/Dashboard'
import Domains from './pages/Domains'
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

function Layout() {
  const { user } = useAuth()

  return (
    <div className="layout">
      <Sidebar />

      <div className="main-shell">
        <AnimatedBackground className="ambient-bg-app" />
        <header className="topbar">
          <div className="topbar-actions">
            <ThemeToggle />
            <UserMenu />
          </div>
        </header>
        <main className="main">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/domains" element={<Domains />} />
            <Route path="/certificates" element={<Navigate to="/domains" replace />} />
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

export { canWriteDomain } from './api'
