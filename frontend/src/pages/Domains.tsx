import { useEffect, useState } from 'react'
import { api, Domain, canWriteDomain } from '../api'
import { useAuth } from '../auth'
import { PageHeader } from '../components/PageHeader'
import { Sheet } from '../components/Sheet'

type RenewStatus = { domain: string; text: string; kind: 'info' | 'ok' | 'err' }

function sleep(ms: number) {
  return new Promise(r => setTimeout(r, ms))
}

export default function Domains() {
  const { user } = useAuth()
  const writable = user && canWriteDomain(user.role)
  const [domains, setDomains] = useState<Domain[]>([])
  const [input, setInput] = useState('')
  const [wildcard, setWildcard] = useState(true)
  const [loading, setLoading] = useState(true)
  const [renewing, setRenewing] = useState<string | null>(null)
  const [status, setStatus] = useState<RenewStatus | null>(null)
  const [sheetOpen, setSheetOpen] = useState(false)

  function load() {
    api.listDomains().then(setDomains).finally(() => setLoading(false))
  }

  useEffect(load, [])

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault()
    if (!input.trim()) return
    await api.createDomains(input, wildcard)
    setInput('')
    setSheetOpen(false)
    load()
  }

  async function handleDelete(id: string) {
    if (!confirm('确认删除该域名？')) return
    await api.deleteDomain(id)
    load()
  }

  async function handleRenew(d: Domain) {
    setRenewing(d.id)
    setStatus({ domain: d.domain, text: '正在提交申请...', kind: 'info' })
    const renewStarted = Date.now()
    try {
      await api.renewDomain(d.id)
      setStatus({ domain: d.domain, text: '证书申请中，DNS 传播可能需要 1～5 分钟...', kind: 'info' })

      while (Date.now() - renewStarted < 720_000) {
        await sleep(3000)
        const logs = await api.listLogs()
        const hit = logs.find(l =>
          l.action === 'renew' &&
          l.domain === d.domain &&
          new Date(l.created_at).getTime() >= renewStarted
        )
        if (!hit) continue
        if (hit.level === 'error') {
          setStatus({ domain: d.domain, text: hit.message, kind: 'err' })
          return
        }
        if (hit.level === 'info' && hit.message !== 'started') {
          setStatus({ domain: d.domain, text: `申请成功：${hit.message}`, kind: 'ok' })
          return
        }
      }
      setStatus({ domain: d.domain, text: '申请超时，请到操作日志查看详情', kind: 'err' })
    } catch (e) {
      setStatus({ domain: d.domain, text: (e as Error).message, kind: 'err' })
    } finally {
      setRenewing(null)
    }
  }

  if (loading) return <p className="loading">加载中...</p>

  return (
    <div>
      <PageHeader title="域名管理" subtitle={`共 ${domains.length} 个根域名`}>
        {writable && (
          <button type="button" className="btn" onClick={() => setSheetOpen(true)}>添加域名</button>
        )}
      </PageHeader>

      {status && (
        <div className={`notice notice-${status.kind}`}>
          <strong>{status.domain}</strong> — {status.text}
          <button type="button" className="notice-close" onClick={() => setStatus(null)}>×</button>
        </div>
      )}

      <div className="card card-elevated">
        {domains.length === 0 ? (
          <div className="attention-empty">
            <strong>暂无域名</strong>
            {writable ? '点击右上角「添加域名」开始管理证书' : '请联系运维添加域名'}
          </div>
        ) : (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>域名</th>
                  <th>通配符</th>
                  <th>状态</th>
                  <th>添加时间</th>
                  {writable && <th>操作</th>}
                </tr>
              </thead>
              <tbody>
                {domains.map(d => (
                  <tr key={d.id}>
                    <td className="cell-domain">{d.domain}</td>
                    <td>{d.wildcard ? '是' : '否'}</td>
                    <td>{d.enabled ? '启用' : '禁用'}</td>
                    <td>{new Date(d.created_at).toLocaleString('zh-CN')}</td>
                    {writable && (
                    <td className="actions">
                      <button
                        className="btn btn-sm"
                        disabled={renewing === d.id}
                        onClick={() => handleRenew(d)}
                      >
                        {renewing === d.id ? '申请中...' : '申请/续签'}
                      </button>
                      <button className="btn btn-sm btn-danger" disabled={!!renewing} onClick={() => handleDelete(d.id)}>删除</button>
                    </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <Sheet
        open={sheetOpen}
        title="添加域名"
        subtitle="每行一个根域名，支持批量导入"
        onClose={() => setSheetOpen(false)}
      >
        <form onSubmit={handleAdd}>
          <div className="form-group">
            <label>根域名列表</label>
            <textarea
              value={input}
              onChange={e => setInput(e.target.value)}
              placeholder="example.com&#10;another.com"
              rows={6}
            />
          </div>
          <div className="form-group">
            <label className="checkbox-label">
              <input type="checkbox" checked={wildcard} onChange={e => setWildcard(e.target.checked)} />
              同时申请通配符证书 (*.domain)
            </label>
          </div>
          <div className="sheet-actions">
            <button type="button" className="btn btn-secondary" onClick={() => setSheetOpen(false)}>取消</button>
            <button type="submit" className="btn">添加</button>
          </div>
        </form>
      </Sheet>
    </div>
  )
}
