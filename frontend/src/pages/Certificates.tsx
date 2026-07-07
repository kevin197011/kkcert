import { useEffect, useState } from 'react'
import { api, Certificate, canWriteDomain } from '../api'
import { useAuth } from '../auth'
import { PageHeader } from '../components/PageHeader'

export default function Certificates() {
  const { user } = useAuth()
  const writable = user && canWriteDomain(user.role)
  const [certs, setCerts] = useState<Certificate[]>([])
  const [loading, setLoading] = useState(true)
  const [syncing, setSyncing] = useState(false)
  const [notice, setNotice] = useState<{ kind: 'ok' | 'err'; text: string } | null>(null)

  useEffect(() => {
    api.listCertificates().then(setCerts).finally(() => setLoading(false))
  }, [])

  async function handleSyncAllGit() {
    setSyncing(true)
    setNotice(null)
    try {
      const res = await api.syncAllCertsGit()
      setNotice({ kind: 'ok', text: `已将 ${res.count} 个域名证书同步到 Git 仓库` })
    } catch (e) {
      setNotice({ kind: 'err', text: (e as Error).message })
    } finally {
      setSyncing(false)
    }
  }

  if (loading) return <p className="loading">加载中...</p>

  return (
    <div>
      <PageHeader title="证书列表" subtitle="全部已签发证书及到期状态">
        {writable && (
          <button className="btn" disabled={syncing || certs.length === 0} onClick={handleSyncAllGit}>
            {syncing ? '同步中...' : '一键同步 Git'}
          </button>
        )}
      </PageHeader>

      {notice && (
        <div className={`notice notice-${notice.kind}`}>
          {notice.text}
          <button type="button" className="notice-close" onClick={() => setNotice(null)}>×</button>
        </div>
      )}

      <div className="card card-elevated">
        {certs.length === 0 ? (
          <p className="empty">暂无证书</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>域名</th>
                <th>签发时间</th>
                <th>过期时间</th>
                <th>剩余天数</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              {certs.map(c => (
                <tr key={c.id}>
                  <td>{c.domain}</td>
                  <td>{new Date(c.issued_at).toLocaleString('zh-CN')}</td>
                  <td>{new Date(c.expires_at).toLocaleString('zh-CN')}</td>
                  <td>{c.days_left}</td>
                  <td><span className={`badge ${c.status}`}>{statusLabel(c.status)}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

function statusLabel(s: string) {
  switch (s) {
    case 'ok': return '正常'
    case 'warning': return '即将过期'
    case 'expired': return '已过期'
    default: return s
  }
}
