import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { api, setToken } from '../api'
import { useAuth } from '../auth'
import { ThemeToggle } from '../theme'
import { Logo } from '../Logo'
import { AnimatedBackground } from '../components/AnimatedBackground'

export default function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [oidcEnabled, setOidcEnabled] = useState(false)
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const { refresh } = useAuth()

  useEffect(() => {
    const token = params.get('token')
    if (token) {
      setToken(token)
      refresh().then(() => navigate('/', { replace: true }))
      return
    }
    fetch('/api/auth/config').then(r => r.json()).then(c => setOidcEnabled(c.oidc_enabled)).catch(() => {})
  }, [params, navigate, refresh])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const { token } = await api.login(username, password)
      setToken(token)
      await refresh()
      navigate('/', { replace: true })
    } catch {
      setError('用户名或密码错误')
    }
  }

  return (
    <div className="login-page">
      <AnimatedBackground className="ambient-bg-login" />

      <div className="login-topbar">
        <ThemeToggle />
      </div>

      <div className="login-layout">
        <div className="login-hero">
          <div className="login-hero-icon"><Logo /></div>
          <h1>KKCert</h1>
          <p className="login-hero-tagline">TLS 证书全生命周期运维</p>
          <ul className="login-features">
            <li>Let's Encrypt 自动申请与续签</li>
            <li>GoDaddy DNS-01 验证</li>
            <li>Git 仓库自动同步</li>
          </ul>
        </div>

        <div className="login-card card">
          <h2>登录</h2>
          <p className="login-sub">进入证书运维管理平台</p>
          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label>用户名</label>
              <input value={username} onChange={e => setUsername(e.target.value)} autoFocus placeholder="admin" />
            </div>
            <div className="form-group">
              <label>密码</label>
              <input type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="••••••••" />
            </div>
            {error && <p className="login-error">{error}</p>}
            <button type="submit" className="btn btn-block">登录</button>
          </form>
          {oidcEnabled && (
            <>
              <div className="login-divider">或</div>
              <button className="btn btn-secondary btn-block" onClick={() => api.oidcLogin()}>
                SSO 登录
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
