import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, Certificate, canWriteDomain } from '../api'
import { useAuth } from '../auth'
import { PageHeader } from '../components/PageHeader'
import { IconCheck, IconAlert, IconX } from '../icons'

export default function Dashboard() {
  const { user } = useAuth()
  const writable = user && canWriteDomain(user.role)
  const [certs, setCerts] = useState<Certificate[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.listCertificates().then(setCerts).finally(() => setLoading(false))
  }, [])

  const ok = certs.filter(c => c.status === 'ok').length
  const warning = certs.filter(c => c.status === 'warning').length
  const expired = certs.filter(c => c.status === 'expired').length
  const attention = certs.filter(c => c.status !== 'ok')

  async function handleCheck() {
    await api.runCheck()
    alert('检测任务已启动，请稍后刷新查看结果')
  }

  if (loading) return <p className="loading">加载中...</p>

  return (
    <div>
      <PageHeader
        title={`你好，${user?.username}`}
        subtitle="证书健康概览"
      >
        {writable && <button className="btn" onClick={handleCheck}>立即检测</button>}
      </PageHeader>

      <div className="stats">
        <div className="stat-card ok">
          <div className="stat-icon"><IconCheck /></div>
          <div className="stat-body">
            <div className="value">{ok}</div>
            <div className="label">正常</div>
          </div>
        </div>
        <div className="stat-card warning">
          <div className="stat-icon"><IconAlert /></div>
          <div className="stat-body">
            <div className="value">{warning}</div>
            <div className="label">即将过期</div>
          </div>
        </div>
        <div className="stat-card expired">
          <div className="stat-icon"><IconX /></div>
          <div className="stat-body">
            <div className="value">{expired}</div>
            <div className="label">已过期</div>
          </div>
        </div>
      </div>

      <p className="page-section-title">需要关注</p>
      <div className="card card-elevated">
        <div className="table-wrap">
          {attention.length === 0 ? (
            <div className="attention-empty">
              <strong>全部证书状态正常</strong>
              暂无即将过期或已过期的证书
            </div>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>域名</th>
                  <th>过期时间</th>
                  <th>剩余天数</th>
                  <th>状态</th>
                </tr>
              </thead>
              <tbody>
                {attention.map(c => (
                  <tr key={c.id}>
                    <td className="cell-domain">{c.domain}</td>
                    <td>{new Date(c.expires_at).toLocaleDateString('zh-CN')}</td>
                    <td>
                      <span className={`days-left days-${c.status}`}>{c.days_left} 天</span>
                    </td>
                    <td><span className={`badge ${c.status}`}>{statusLabel(c.status)}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
        <Link to="/certificates" className="page-link">查看全部证书 →</Link>
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
