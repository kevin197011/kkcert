import { useEffect, useState } from 'react'
import { api, APIToken } from '../api'
import { PageHeader } from '../components/PageHeader'
import { Sheet } from '../components/Sheet'
import { TablePagination } from '../components/TablePagination'
import { usePagination } from '../hooks/usePagination'

export default function Tokens() {
  const [tokens, setTokens] = useState<APIToken[]>([])
  const [form, setForm] = useState({ name: '', role: 'operator', expires_days: 0 })
  const [created, setCreated] = useState('')
  const [loading, setLoading] = useState(true)
  const [sheetOpen, setSheetOpen] = useState(false)
  const pagination = usePagination(tokens, 'tokens')

  function load() {
    api.listTokens().then(setTokens).finally(() => setLoading(false))
  }

  useEffect(load, [])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    const res = await api.createToken(form)
    setCreated(res.token)
    setForm({ name: '', role: 'operator', expires_days: 0 })
    setSheetOpen(false)
    load()
  }

  async function handleRevoke(id: string) {
    if (!confirm('确认吊销该 Token？')) return
    await api.deleteToken(id)
    load()
  }

  if (loading) return <p className="loading">加载中...</p>

  return (
    <div>
      <PageHeader title="API Token" subtitle={`共 ${tokens.length} 个有效 Token`}>
        <a className="btn btn-secondary btn-sm" href="/api/docs" target="_blank" rel="noreferrer">Swagger</a>
        <button type="button" className="btn" onClick={() => setSheetOpen(true)}>创建 Token</button>
      </PageHeader>

      {created && (
        <div className="notice notice-ok">
          <div>
            <strong>Token 已创建（仅显示一次，请立即复制）：</strong>
            <code style={{ display: 'block', marginTop: 8, wordBreak: 'break-all' }}>{created}</code>
          </div>
          <button type="button" className="notice-close" onClick={() => setCreated('')}>×</button>
        </div>
      )}

      <div className="card card-elevated">
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>名称</th>
                <th>前缀</th>
                <th>角色</th>
                <th>创建时间</th>
                <th>最后使用</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {tokens.length === 0 ? (
                <tr><td colSpan={6} className="empty">暂无 Token，点击右上角创建</td></tr>
              ) : pagination.pageItems.map(t => (
                <tr key={t.id}>
                  <td className="cell-domain">{t.name}</td>
                  <td><code>{t.prefix}</code></td>
                  <td>{t.role}</td>
                  <td>{new Date(t.created_at).toLocaleString('zh-CN')}</td>
                  <td>{t.last_used_at ? new Date(t.last_used_at).toLocaleString('zh-CN') : '-'}</td>
                  <td>
                    <button className="btn btn-sm btn-danger" onClick={() => handleRevoke(t.id)}>吊销</button>
                  </td>
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
      </div>

      <Sheet
        open={sheetOpen}
        title="创建 Token"
        subtitle="供 AI Agent 与自动化脚本调用 API"
        onClose={() => setSheetOpen(false)}
      >
        <form onSubmit={handleCreate}>
          <div className="form-group">
            <label>名称</label>
            <input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} placeholder="cursor-agent" required />
          </div>
          <div className="form-group">
            <label>角色</label>
            <select value={form.role} onChange={e => setForm({ ...form, role: e.target.value })}>
              <option value="operator">运维</option>
              <option value="viewer">只读</option>
              <option value="admin">管理员</option>
            </select>
          </div>
          <div className="form-group">
            <label>有效天数（0 = 永久）</label>
            <input type="number" value={form.expires_days} onChange={e => setForm({ ...form, expires_days: +e.target.value })} />
          </div>
          <p className="field-hint">请求头：Authorization: Bearer kkcert_xxx...</p>
          <div className="sheet-actions">
            <button type="button" className="btn btn-secondary" onClick={() => setSheetOpen(false)}>取消</button>
            <button type="submit" className="btn">创建</button>
          </div>
        </form>
      </Sheet>
    </div>
  )
}
