import { useEffect, useState } from 'react'
import { api, OpLog } from '../api'
import { PageHeader } from '../components/PageHeader'
import { TablePagination } from '../components/TablePagination'
import { formatDateTime } from '../datetime'
import { usePagination } from '../hooks/usePagination'

export default function Logs() {
  const [logs, setLogs] = useState<OpLog[]>([])
  const [loading, setLoading] = useState(true)
  const pagination = usePagination(logs, 'logs')

  useEffect(() => {
    api.listLogs().then(setLogs).finally(() => setLoading(false))
  }, [])

  if (loading) return <p className="loading">加载中...</p>

  return (
    <div>
      <PageHeader title="操作日志" subtitle={`共 ${logs.length} 条系统操作与任务记录`} />
      <div className="card card-elevated">
        {logs.length === 0 ? (
          <p className="empty">暂无日志</p>
        ) : (
          <div className="table-wrap">
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
                {pagination.pageItems.map(l => (
                  <tr key={l.id}>
                    <td>{formatDateTime(l.created_at)}</td>
                    <td className={l.level === 'error' ? 'log-error' : 'log-info'}>{l.level}</td>
                    <td>{l.action}</td>
                    <td>{l.domain || '-'}</td>
                    <td>{l.message}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            <TablePagination
              page={pagination.page}
              pageSize={pagination.pageSize}
              total={pagination.total}
              totalPages={pagination.totalPages}
              pageSizes={pagination.pageSizes}
              show={pagination.showPagination}
              onPageChange={pagination.setPage}
              onPageSizeChange={pagination.setPageSize}
            />
          </div>
        )}
      </div>
    </div>
  )
}
