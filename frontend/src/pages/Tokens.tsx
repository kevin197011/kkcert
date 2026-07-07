import { useEffect, useState } from 'react'
import { api, APIToken } from '../api'

import { PageHeader } from '../components/PageHeader'

export default function Tokens() {
  const [tokens, setTokens] = useState<APIToken[]>([])
  const [form, setForm] = useState({ name: '', role: 'operator', expires_days: 0 })
  const [created, setCreated] = useState('')
  const [loading, setLoading] = useState(true)

  function load() {
    api.listTokens().then(setTokens).finally(() => setLoading(false))
  }

  useEffect(load, [])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    const res = await api.createToken(form)
    setCreated(res.token)
    setForm({ name: '', role: 'operator', expires_days: 0 })
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
      <PageHeader title="API Token" subtitle="为 AI Agent 与自动化脚本创建访问凭证">
        <a className="btn btn-secondary btn-sm" href="/api/docs" target="_blank" rel="noreferrer">Swagger 文档</a>
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

      <div className="card">
        <h3 className="card-title">创建 Token（AI Agent / 自动化）</h3>
        <form onSubmit={handleCreate}>
          <div className="form-row">
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
          </div>
          <div className="form-group">
            <label>有效天数（0 = 永久）</label>
            <input type="number" value={form.expires_days} onChange={e => setForm({ ...form, expires_days: +e.target.value })} />
          </div>
          <button type="submit" className="btn">创建 Token</button>
        </form>
        <p className="field-hint">请求头格式：Authorization: Bearer kkcert_xxx...</p>
      </div>

      <div className="card">
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
              <tr><td colSpan={6} className="empty">暂无 Token</td></tr>
            ) : tokens.map(t => (
              <tr key={t.id}>
                <td>{t.name}</td>
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
      </div>
    </div>
  )
}
