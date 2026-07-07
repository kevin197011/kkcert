type Props = {
  page: number
  pageSize: number
  total: number
  totalPages: number
  pageSizes: readonly number[]
  show: boolean
  onPageChange: (page: number) => void
  onPageSizeChange: (size: number) => void
}

export function TablePagination({
  page,
  pageSize,
  total,
  totalPages,
  pageSizes,
  show,
  onPageChange,
  onPageSizeChange,
}: Props) {
  if (!show) return null

  const start = total === 0 ? 0 : (page - 1) * pageSize + 1
  const end = Math.min(page * pageSize, total)

  const pages = buildPageList(page, totalPages)

  return (
    <div className="table-pagination">
      <span className="table-pagination-info">
        第 {start}–{end} 条，共 {total} 条
      </span>
      <div className="table-pagination-controls">
        <label className="table-pagination-size">
          每页
          <select value={pageSize} onChange={e => onPageSizeChange(+e.target.value)}>
            {pageSizes.map(n => (
              <option key={n} value={n}>{n}</option>
            ))}
          </select>
        </label>
        <button
          type="button"
          className="btn btn-sm btn-secondary"
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
        >
          上一页
        </button>
        <div className="table-pagination-pages">
          {pages.map((p, i) =>
            p === '…' ? (
              <span key={`gap-${i}`} className="table-pagination-ellipsis">…</span>
            ) : (
              <button
                key={p}
                type="button"
                className={`table-pagination-page${p === page ? ' active' : ''}`}
                onClick={() => onPageChange(p)}
              >
                {p}
              </button>
            )
          )}
        </div>
        <button
          type="button"
          className="btn btn-sm btn-secondary"
          disabled={page >= totalPages}
          onClick={() => onPageChange(page + 1)}
        >
          下一页
        </button>
      </div>
    </div>
  )
}

function buildPageList(current: number, total: number): (number | '…')[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1)
  }
  const pages: (number | '…')[] = [1]
  if (current > 3) pages.push('…')
  const from = Math.max(2, current - 1)
  const to = Math.min(total - 1, current + 1)
  for (let p = from; p <= to; p++) pages.push(p)
  if (current < total - 2) pages.push('…')
  pages.push(total)
  return pages
}
