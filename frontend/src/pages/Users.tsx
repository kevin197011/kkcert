import { useEffect, useState } from 'react'
import { api, User } from '../api'
import { PageHeader } from '../components/PageHeader'
import { Sheet } from '../components/Sheet'

export default function Users() {
  const [users, setUsers] = useState<User[]>([])
  const [form, setForm] = useState({ username: '', email: '', password: '', role: 'viewer' })
  const [loading, setLoading] = useState(true)
  const [sheetOpen, setSheetOpen] = useState(false)

  function load() {
    api.listUsers().then(setUsers).finally(() => setLoading(false))
  }

  useEffect(load, [])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    await api.createUser(form)
    setForm({ username: '', email: '', password: '', role: 'viewer' })
    setSheetOpen(false)
    load()
  }

  async function handleRoleChange(id: string, role: string) {
    await api.updateUser(id, { role })
    load()
  }

  async function handleDelete(id: string) {
    if (!confirm('确认删除该用户？')) return
    try {
      await api.deleteUser(id)
      load()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  if (loading) return <p className="loading">加载中...</p>

  return (
    <div>
      <PageHeader title="用户管理" subtitle={`共 ${users.length} 个账号`}>
        <button type="button" className="btn" onClick={() => setSheetOpen(true)}>新建用户</button>
      </PageHeader>

      <div className="card card-elevated">
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>用户名</th>
                <th>邮箱</th>
                <th>角色</th>
                <th>类型</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {users.length === 0 ? (
                <tr><td colSpan={5} className="empty">暂无用户</td></tr>
              ) : users.map(u => (
                <tr key={u.id}>
                  <td className="cell-domain">{u.username}</td>
                  <td>{u.email || '-'}</td>
                  <td>
                    <select value={u.role} onChange={e => handleRoleChange(u.id, e.target.value)}>
                      <option value="admin">管理员</option>
                      <option value="operator">运维</option>
                      <option value="viewer">只读</option>
                    </select>
                  </td>
                  <td>{u.auth_type}</td>
                  <td>
                    <button className="btn btn-sm btn-danger" onClick={() => handleDelete(u.id)}>删除</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <Sheet
        open={sheetOpen}
        title="新建用户"
        subtitle="创建本地登录账号"
        onClose={() => setSheetOpen(false)}
      >
        <form onSubmit={handleCreate}>
          <div className="form-group">
            <label>用户名</label>
            <input value={form.username} onChange={e => setForm({ ...form, username: e.target.value })} required />
          </div>
          <div className="form-group">
            <label>邮箱</label>
            <input value={form.email} onChange={e => setForm({ ...form, email: e.target.value })} />
          </div>
          <div className="form-group">
            <label>密码</label>
            <input type="password" value={form.password} onChange={e => setForm({ ...form, password: e.target.value })} required />
          </div>
          <div className="form-group">
            <label>角色</label>
            <select value={form.role} onChange={e => setForm({ ...form, role: e.target.value })}>
              <option value="admin">管理员</option>
              <option value="operator">运维</option>
              <option value="viewer">只读</option>
            </select>
          </div>
          <div className="sheet-actions">
            <button type="button" className="btn btn-secondary" onClick={() => setSheetOpen(false)}>取消</button>
            <button type="submit" className="btn">创建</button>
          </div>
        </form>
      </Sheet>
    </div>
  )
}
