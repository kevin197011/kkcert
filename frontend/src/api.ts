const TOKEN_KEY = 'kkcert_token'

function headers(): HeadersInit {
  const h: HeadersInit = { 'Content-Type': 'application/json' }
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) h['Authorization'] = `Bearer ${token}`
  return h
}

async function request<T>(path: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(`/api${path}`, { ...opts, headers: { ...headers(), ...opts?.headers } })
  if (res.status === 401) {
    localStorage.removeItem(TOKEN_KEY)
    if (!window.location.pathname.startsWith('/login')) {
      window.location.href = '/login'
    }
    throw new Error('unauthorized')
  }
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || res.statusText)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export interface User {
  id: string
  username: string
  email: string
  role: 'admin' | 'operator' | 'viewer'
  auth_type: string
  enabled?: boolean
}

export interface Domain {
  id: string
  domain: string
  wildcard: boolean
  enabled: boolean
  created_at: string
}

export interface Certificate {
  id: string
  domain_id: string
  domain: string
  expires_at: string
  issued_at: string
  days_left: number
  status: 'ok' | 'warning' | 'expired'
}

export interface Settings {
  acme_email: string
  acme_staging: boolean
  godaddy_api_key: string
  godaddy_api_secret: string
  git_repo_url: string
  git_branch: string
  git_auth_type: string
  git_ssh_key_path: string
  git_token: string
  git_certs_dir: string
  renew_before_days: number
  auto_renew_enabled: boolean
  check_cron: string
  cleanup_cron: string
  oidc_enabled: boolean
  oidc_issuer: string
  oidc_client_id: string
  oidc_client_secret: string
  oidc_redirect_url: string
  oidc_default_role: string
}

export interface APIToken {
  id: string
  name: string
  prefix: string
  role: string
  created_at: string
  expires_at?: string
  last_used_at?: string
}

export interface OpLog {
  id: string
  level: string
  action: string
  message: string
  domain?: string
  created_at: string
}

export const api = {
  login: (username: string, password: string) =>
    request<{ token: string }>('/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) }),
  me: () => request<User>('/auth/me'),
  logout: () => request<void>('/auth/logout', { method: 'POST' }),
  oidcLogin: () => { window.location.href = '/api/auth/oidc/login' },
  listUsers: () => request<User[]>('/users').then(r => r ?? []),
  createUser: (data: { username: string; email: string; password: string; role: string }) =>
    request<User>('/users', { method: 'POST', body: JSON.stringify(data) }),
  updateUser: (id: string, data: Partial<{ email: string; role: string; enabled: boolean; password: string }>) =>
    request<User>(`/users/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteUser: (id: string) => request<void>(`/users/${id}`, { method: 'DELETE' }),
  listDomains: () => request<Domain[]>('/domains').then(r => r ?? []),
  createDomains: (domains: string, wildcard: boolean) =>
    request<Domain[]>('/domains', { method: 'POST', body: JSON.stringify({ domains, wildcard }) }),
  deleteDomain: (id: string) => request<void>(`/domains/${id}`, { method: 'DELETE' }),
  renewDomain: (id: string) => request<{ status: string }>(`/domains/${id}/renew`, { method: 'POST' }),
  listCertificates: () => request<Certificate[]>('/certificates').then(r => r ?? []),
  syncAllCertsGit: () => request<{ status: string; count: number }>('/certificates/sync-git', { method: 'POST' }),
  getSettings: () => request<Settings>('/settings'),
  putSettings: (s: Settings) => request<{ status: string }>('/settings', { method: 'PUT', body: JSON.stringify(s) }),
  resetACME: () => request<{ status: string }>('/settings/acme/reset', { method: 'POST', body: '{}' }),
  listLogs: () => request<OpLog[]>('/logs').then(r => r ?? []),
  runCheck: () => request<{ status: string }>('/check/run', { method: 'POST' }),
  listTokens: () => request<APIToken[]>('/tokens').then(r => r ?? []),
  createToken: (data: { name: string; role: string; expires_days?: number }) =>
    request<{ token: string; record: APIToken }>('/tokens', { method: 'POST', body: JSON.stringify(data) }),
  deleteToken: (id: string) => request<void>(`/tokens/${id}`, { method: 'DELETE' }),
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function getToken() {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

export function canWriteDomain(role: string) {
  return role === 'admin' || role === 'operator'
}

export function canWriteSettings(role: string) {
  return role === 'admin'
}

export function canManageUsers(role: string) {
  return role === 'admin'
}

export function canEditUser(_actorRole: string, target: User) {
  return target.username !== 'admin'
}
