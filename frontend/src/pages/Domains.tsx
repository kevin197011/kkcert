import { useEffect, useState } from 'react'
import { api, Certificate, Domain, canWriteDomain } from '../api'
import { useAuth } from '../auth'
import { PageHeader } from '../components/PageHeader'
import { Sheet } from '../components/Sheet'
import { TablePagination } from '../components/TablePagination'
import { usePagination } from '../hooks/usePagination'

type RenewStatus = { domain: string; text: string; kind: 'info' | 'ok' | 'err' }

function sleep(ms: number) {
  return new Promise(r => setTimeout(r, ms))
}

function statusLabel(s: string) {
  switch (s) {
    case 'ok': return '正常'
    case 'warning': return '即将过期'
    case 'expired': return '已过期'
    default: return s
  }
}

export default function Domains() {
  const { user } = useAuth()
  const writable = user && canWriteDomain(user.role)
  const [domains, setDomains] = useState<Domain[]>([])
  const [certByDomainId, setCertByDomainId] = useState<Map<string, Certificate>>(new Map())
  const [input, setInput] = useState('')
  const [wildcard, setWildcard] = useState(true)
  const [loading, setLoading] = useState(true)
  const [renewing, setRenewing] = useState<string | null>(null)
  const [status, setStatus] = useState<RenewStatus | null>(null)
  const [sheetOpen, setSheetOpen] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [gitNotice, setGitNotice] = useState<{ kind: 'ok' | 'err'; text: string } | null>(null)

  function load() {
    Promise.all([api.listDomains(), api.listCertificates()])
      .then(([domainList, certs]) => {
        setDomains(domainList)
        setCertByDomainId(new Map(certs.map(c => [c.domain_id, c])))
      })
      .finally(() => setLoading(false))
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
    if (!confirm('确认删除该域名？关联证书将一并清除。')) return
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
          load()
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

  async function handleSyncAllGit() {
    setSyncing(true)
    setGitNotice(null)
    try {
      const res = await api.syncAllCertsGit()
      setGitNotice({ kind: 'ok', text: `已将 ${res.count} 个域名证书同步到 Git 仓库` })
    } catch (e) {
      setGitNotice({ kind: 'err', text: (e as Error).message })
    } finally {
      setSyncing(false)
    }
  }

  const certCount = certByDomainId.size
  const pagination = usePagination(domains, 'domains')

  if (loading) return <p className="loading">加载中...</p>

  return (
    <div>
      <PageHeader title="域名与证书" subtitle={`${domains.length} 个域名 · ${certCount} 张有效证书`}>
        {writable && (
          <>
            <button
              type="button"
              className="btn btn-secondary"
              disabled={syncing || certCount === 0}
              onClick={handleSyncAllGit}
            >
              {syncing ? '同步中...' : '一键同步 Git'}
            </button>
            <button type="button" className="btn" onClick={() => setSheetOpen(true)}>添加域名</button>
          </>
        )}
      </PageHeader>

      {status && (
        <div className={`notice notice-${status.kind}`}>
          <strong>{status.domain}</strong> — {status.text}
          <button type="button" className="notice-close" onClick={() => setStatus(null)}>×</button>
        </div>
      )}

      {gitNotice && (
        <div className={`notice notice-${gitNotice.kind}`}>
          {gitNotice.text}
          <button type="button" className="notice-close" onClick={() => setGitNotice(null)}>×</button>
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
                  <th>检测</th>
                  <th>证书状态</th>
                  <th>过期时间</th>
                  <th>剩余天数</th>
                  {writable && <th>操作</th>}
                </tr>
              </thead>
              <tbody>
                {pagination.pageItems.map(d => {
                  const cert = certByDomainId.get(d.id)
                  return (
                    <tr key={d.id}>
                      <td className="cell-domain">{d.domain}</td>
                      <td>{d.wildcard ? '是' : '否'}</td>
                      <td>{d.enabled ? '启用' : '禁用'}</td>
                      <td>
                        {cert
                          ? <span className={`badge ${cert.status}`}>{statusLabel(cert.status)}</span>
                          : <span className="text-muted">未签发</span>}
                      </td>
                      <td>{cert ? new Date(cert.expires_at).toLocaleDateString('zh-CN') : '—'}</td>
                      <td>
                        {cert
                          ? <span className={`days-left days-${cert.status}`}>{cert.days_left} 天</span>
                          : '—'}
                      </td>
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
