import { useEffect, useState } from 'react'
import { api, User } from '../api'

import { PageHeader } from '../components/PageHeader'

export default function Users() {
  const [users, setUsers] = useState<User[]>([])
  const [form, setForm] = useState({ username: '', email: '', password: '', role: 'viewer' })
  const [loading, setLoading] = useState(true)

  function load() {
    api.listUsers().then(setUsers).finally(() => setLoading(false))
  }

  useEffect(load, [])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    await api.createUser(form)
    setForm({ username: '', email: '', password: '', role: 'viewer' })
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
      <PageHeader title="用户管理" subtitle="本地账号与 OIDC 用户的角色分配" />

      <div className="card">
        <h3 style={{ marginBottom: 16 }}>新建用户</h3>
        <form onSubmit={handleCreate}>
          <div className="form-row">
            <div className="form-group">
              <label>用户名</label>
              <input value={form.username} onChange={e => setForm({ ...form, username: e.target.value })} required />
            </div>
            <div className="form-group">
              <label>邮箱</label>
              <input value={form.email} onChange={e => setForm({ ...form, email: e.target.value })} />
            </div>
          </div>
          <div className="form-row">
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
          </div>
          <button type="submit" className="btn">创建用户</button>
        </form>
      </div>

      <div className="card">
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
            {users.map(u => (
              <tr key={u.id}>
                <td>{u.username}</td>
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
  )
}
