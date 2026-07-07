import { useEffect, useState } from 'react'
import { api, OpLog } from '../api'

import { PageHeader } from '../components/PageHeader'

export default function Logs() {
  const [logs, setLogs] = useState<OpLog[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.listLogs().then(setLogs).finally(() => setLoading(false))
  }, [])

  if (loading) return <p className="loading">加载中...</p>

  return (
    <div>
      <PageHeader title="操作日志" subtitle="最近 50 条系统操作与任务记录" />
      <div className="card card-elevated">
        {logs.length === 0 ? (
          <p className="empty">暂无日志</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>时间</th>
                <th>级别</th>
                <th>操作</th>
                <th>域名</th>
                <th>消息</th>
              </tr>
            </thead>
            <tbody>
              {logs.map(l => (
                <tr key={l.id}>
                  <td>{new Date(l.created_at).toLocaleString('zh-CN')}</td>
                  <td className={l.level === 'error' ? 'log-error' : 'log-info'}>{l.level}</td>
                  <td>{l.action}</td>
                  <td>{l.domain || '-'}</td>
                  <td>{l.message}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
