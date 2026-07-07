import { useEffect, useState } from 'react'
import { api, Settings } from '../api'
import { PageHeader } from '../components/PageHeader'
import { Tabs } from '../components/Tabs'

const SETTING_TABS = [
  { id: 'acme', label: 'ACME' },
  { id: 'auth', label: '单点登录' },
  { id: 'dns', label: 'GoDaddy' },
  { id: 'git', label: 'Git 同步' },
  { id: 'schedule', label: '续签策略' },
] as const

type TabId = (typeof SETTING_TABS)[number]['id']

export default function SettingsPage() {
  const [settings, setSettings] = useState<Settings | null>(null)
  const [saving, setSaving] = useState(false)
  const [tab, setTab] = useState<TabId>('acme')

  useEffect(() => {
    api.getSettings().then(setSettings)
  }, [])

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    if (!settings) return
    setSaving(true)
    try {
      await api.putSettings(settings)
      alert('保存成功')
    } catch (err) {
      alert('保存失败: ' + (err as Error).message)
    } finally {
      setSaving(false)
    }
  }

  async function handleResetACME() {
    if (!confirm('确认重置 ACME 账户？下次申请将重新注册。')) return
    await api.resetACME()
    alert('ACME 账户已重置')
  }

  function update<K extends keyof Settings>(key: K, value: Settings[K]) {
    setSettings(s => s ? { ...s, [key]: value } : s)
  }

  if (!settings) return <p className="loading">加载中...</p>

  return (
    <div>
      <PageHeader title="系统设置" subtitle="按模块分别配置，避免单页堆砌" />

      <Tabs tabs={[...SETTING_TABS]} active={tab} onChange={id => setTab(id as TabId)} />

      <form onSubmit={handleSave}>
        <div className="card settings-panel">
          {tab === 'acme' && (
            <>
              <h3 className="card-title">ACME / Let's Encrypt</h3>
              <div className="form-group">
                <label>注册邮箱</label>
                <input
                  type="email"
                  value={settings.acme_email}
                  onChange={e => update('acme_email', e.target.value)}
                  placeholder="admin@example.com"
                />
                <p className="field-hint">用于 Let's Encrypt 账户注册与到期通知</p>
              </div>
              <div className="form-group">
                <label className="checkbox-label">
                  <input type="checkbox" checked={settings.acme_staging} onChange={e => update('acme_staging', e.target.checked)} />
                  使用 Staging 环境（测试）
                </label>
              </div>
              <button type="button" className="btn btn-danger btn-sm" onClick={handleResetACME}>重置 ACME 账户</button>
            </>
          )}

          {tab === 'auth' && (
            <>
              <h3 className="card-title">OIDC 单点登录</h3>
              <div className="form-group">
                <label className="checkbox-label">
                  <input type="checkbox" checked={settings.oidc_enabled} onChange={e => update('oidc_enabled', e.target.checked)} />
                  启用 OIDC SSO
                </label>
              </div>
              <div className="form-group">
                <label>Issuer URL</label>
                <input value={settings.oidc_issuer} onChange={e => update('oidc_issuer', e.target.value)} placeholder="https://idp.example.com" />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Client ID</label>
                  <input value={settings.oidc_client_id} onChange={e => update('oidc_client_id', e.target.value)} />
                </div>
                <div className="form-group">
                  <label>Client Secret</label>
                  <input type="password" value={settings.oidc_client_secret} onChange={e => update('oidc_client_secret', e.target.value)} placeholder="留空则不修改" />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Redirect URL</label>
                  <input value={settings.oidc_redirect_url} onChange={e => update('oidc_redirect_url', e.target.value)} placeholder="https://kkcert.example.com/api/auth/oidc/callback" />
                </div>
                <div className="form-group">
                  <label>新用户默认角色</label>
                  <select value={settings.oidc_default_role} onChange={e => update('oidc_default_role', e.target.value)}>
                    <option value="viewer">只读</option>
                    <option value="operator">运维</option>
                    <option value="admin">管理员</option>
                  </select>
                </div>
              </div>
            </>
          )}

          {tab === 'dns' && (
            <>
              <h3 className="card-title">GoDaddy DNS</h3>
              <div className="form-row">
                <div className="form-group">
                  <label>API Key</label>
                  <input value={settings.godaddy_api_key} onChange={e => update('godaddy_api_key', e.target.value)} />
                </div>
                <div className="form-group">
                  <label>API Secret</label>
                  <input type="password" value={settings.godaddy_api_secret} onChange={e => update('godaddy_api_secret', e.target.value)} placeholder="留空则不修改" />
                </div>
              </div>
              <p className="field-hint">DNS 托管须在 GoDaddy，用于 ACME DNS-01 验证</p>
            </>
          )}

          {tab === 'git' && (
            <>
              <h3 className="card-title">Git 同步</h3>
              <div className="form-group">
                <label>仓库地址</label>
                <input value={settings.git_repo_url} onChange={e => update('git_repo_url', e.target.value)} placeholder="git@github.com:org/certs.git" />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>分支</label>
                  <input value={settings.git_branch} onChange={e => update('git_branch', e.target.value)} />
                </div>
                <div className="form-group">
                  <label>证书目录</label>
                  <input value={settings.git_certs_dir} onChange={e => update('git_certs_dir', e.target.value)} />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>鉴权方式</label>
                  <select value={settings.git_auth_type} onChange={e => update('git_auth_type', e.target.value)}>
                    <option value="ssh">SSH Key</option>
                    <option value="token">HTTPS Token</option>
                  </select>
                </div>
                <div className="form-group">
                  {settings.git_auth_type === 'ssh' ? (
                    <>
                      <label>SSH 私钥路径</label>
                      <input value={settings.git_ssh_key_path} onChange={e => update('git_ssh_key_path', e.target.value)} placeholder="/data/id_rsa" />
                    </>
                  ) : (
                    <>
                      <label>Access Token</label>
                      <input type="password" value={settings.git_token} onChange={e => update('git_token', e.target.value)} placeholder="留空则不修改" />
                    </>
                  )}
                </div>
              </div>
            </>
          )}

          {tab === 'schedule' && (
            <>
              <h3 className="card-title">续签与定时任务</h3>
              <div className="form-row">
                <div className="form-group">
                  <label>提前续签天数</label>
                  <input type="number" value={settings.renew_before_days} onChange={e => update('renew_before_days', +e.target.value)} />
                </div>
                <div className="form-group">
                  <label>检测 Cron 表达式</label>
                  <input value={settings.check_cron} onChange={e => update('check_cron', e.target.value)} placeholder="0 3 * * *" />
                </div>
              </div>
              <div className="form-group">
                <label>数据清理 Cron 表达式</label>
                <input value={settings.cleanup_cron} onChange={e => update('cleanup_cron', e.target.value)} placeholder="0 4 * * *" />
                <p className="field-hint">定期物理删除归档域名证书、无效证书记录和过期会话</p>
              </div>
              <div className="form-group">
                <label className="checkbox-label">
                  <input type="checkbox" checked={settings.auto_renew_enabled} onChange={e => update('auto_renew_enabled', e.target.checked)} />
                  启用自动续签
                </label>
              </div>
            </>
          )}

          <div className="settings-save-bar">
            <button type="submit" className="btn" disabled={saving}>
              {saving ? '保存中...' : '保存设置'}
            </button>
          </div>
        </div>
      </form>
    </div>
  )
}
