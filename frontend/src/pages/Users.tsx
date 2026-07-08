import { useEffect, useState } from 'react'
import { api, User, canEditUser } from '../api'
import { useAuth } from '../auth'
import { PageHeader } from '../components/PageHeader'
import { Notice } from '../components/Notice'
import { Sheet } from '../components/Sheet'
import { useFeedback } from '../feedback'
import { TablePagination } from '../components/TablePagination'
import { usePagination } from '../hooks/usePagination'

const roleLabel: Record<string, string> = {
  admin: '管理员',
  operator: '运维',
  viewer: '只读',
}

export default function Users() {
  const { user: me } = useAuth()
  const { toast, confirm } = useFeedback()
  const [users, setUsers] = useState<User[]>([])
  const [form, setForm] = useState({ username: '', email: '', password: '', role: 'viewer' })
  const [edit, setEdit] = useState<User | null>(null)
  const [editForm, setEditForm] = useState({ email: '', role: 'viewer', enabled: true, password: '' })
  const [loading, setLoading] = useState(true)
  const [sheetOpen, setSheetOpen] = useState(false)
  const pagination = usePagination(users, 'users')

  const isAdmin = me?.role === 'admin'

  function load() {
    api.listUsers().then(setUsers).finally(() => setLoading(false))
  }

  useEffect(load, [])

  function openEdit(u: User) {
    setEdit(u)
    setEditForm({
      email: u.email || '',
      role: u.role,
      enabled: u.enabled !== false,
      password: '',
    })
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    const username = form.username
    try {
      await api.createUser(form)
      setForm({ username: '', email: '', password: '', role: 'viewer' })
      setSheetOpen(false)
      load()
      toast({ kind: 'ok', title: '用户已创建', message: username })
    } catch (e) {
      toast({ kind: 'err', title: '创建失败', message: (e as Error).message })
    }
  }

  async function handleSaveEdit(e: React.FormEvent) {
    e.preventDefault()
    if (!edit) return
    const data: { email: string; role: string; enabled: boolean; password?: string } = {
      email: editForm.email,
      role: editForm.role,
      enabled: editForm.enabled,
    }
    if (editForm.password) data.password = editForm.password
    try {
      await api.updateUser(edit.id, data)
      setEdit(null)
      load()
      toast({ kind: 'ok', message: `已更新用户 ${edit.username}` })
    } catch (err) {
      toast({ kind: 'err', title: '保存失败', message: (err as Error).message })
    }
  }

  async function handleDelete(id: string, username: string) {
    const ok = await confirm({
      title: '删除用户',
      message: `确认删除用户 ${username}？此操作不可撤销。`,
      confirmLabel: '删除',
      danger: true,
    })
    if (!ok) return
    try {
      await api.deleteUser(id)
      load()
      toast({ kind: 'ok', message: `已删除用户 ${username}` })
    } catch (e) {
      toast({ kind: 'err', title: '删除失败', message: (e as Error).message })
    }
  }

  if (loading) return <p className="loading">加载中...</p>

  return (
    <div>
      <PageHeader title="用户管理" subtitle={`共 ${users.length} 个账号`}>
        {isAdmin && (
          <button type="button" className="btn" onClick={() => setSheetOpen(true)}>新建用户</button>
        )}
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
              ) : pagination.pageItems.map(u => {
                const editable = canEditUser(me!.role, u)
                return (
                  <tr key={u.id}>
                    <td className="cell-domain">{u.username}</td>
                    <td>{u.email || '-'}</td>
                    <td>{roleLabel[u.role] || u.role}</td>
                    <td>{u.auth_type}</td>
                    <td className="actions">
                      {editable ? (
                        <>
                          <button type="button" className="btn btn-sm" onClick={() => openEdit(u)}>编辑</button>
                          {isAdmin && (
                            <button type="button" className="btn btn-sm btn-danger" onClick={() => handleDelete(u.id, u.username)}>删除</button>
                          )}
                        </>
                      ) : (
                        <span className="text-muted">—</span>
                      )}
                    </td>
                  </tr>
                )
              })}
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
        title="新建用户"
        subtitle="创建本地登录账号（仅管理员）"
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

      <Sheet
        open={!!edit}
        title={`编辑用户：${edit?.username ?? ''}`}
        subtitle="调整角色与账号信息"
        onClose={() => setEdit(null)}
      >
        {edit && (
          <form onSubmit={handleSaveEdit}>
            <div className="form-group">
              <label>邮箱</label>
              <input value={editForm.email} onChange={e => setEditForm({ ...editForm, email: e.target.value })} />
            </div>
            <div className="form-group">
              <label>角色</label>
              <select
                value={editForm.role}
                onChange={e => setEditForm({ ...editForm, role: e.target.value })}
              >
                <option value="admin">管理员</option>
                <option value="operator">运维</option>
                <option value="viewer">只读</option>
              </select>
              <p className="field-hint">内置 `admin` 账号不可编辑；其他用户可调整为管理员</p>
            </div>
            <div className="form-group">
              <label className="checkbox-label">
                <input
                  type="checkbox"
                  checked={editForm.enabled}
                  onChange={e => setEditForm({ ...editForm, enabled: e.target.checked })}
                />
                账号启用
              </label>
            </div>
            {edit.auth_type === 'local' && (
              <div className="form-group">
                <label>新密码</label>
                <input
                  type="password"
                  value={editForm.password}
                  onChange={e => setEditForm({ ...editForm, password: e.target.value })}
                  placeholder="留空则不修改"
                />
              </div>
            )}
            <div className="sheet-actions">
              <button type="button" className="btn btn-secondary" onClick={() => setEdit(null)}>取消</button>
              <button type="submit" className="btn">保存</button>
            </div>
          </form>
        )}
      </Sheet>
    </div>
  )
}
