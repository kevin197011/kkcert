import { useEffect, useMemo, useRef, useState } from 'react'

const PAGE_SIZES = [10, 20, 50, 100] as const

function pickSmartPageSize(total: number): number {
  if (total <= 10) return 10
  if (total <= 50) return 20
  return 50
}

function readStoredPageSize(key: string): number | null {
  const raw = localStorage.getItem(`kkcert_page_size_${key}`)
  if (!raw) return null
  const n = parseInt(raw, 10)
  return PAGE_SIZES.includes(n as (typeof PAGE_SIZES)[number]) ? n : null
}

export function usePagination<T>(items: T[], key: string) {
  const [page, setPage] = useState(1)
  const [pageSize, setPageSizeState] = useState(20)
  const seeded = useRef(false)

  const total = items.length
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  useEffect(() => {
    if (seeded.current || total === 0) return
    const stored = readStoredPageSize(key)
    setPageSizeState(stored ?? pickSmartPageSize(total))
    seeded.current = true
  }, [key, total])

  useEffect(() => {
    if (page > totalPages) setPage(totalPages)
  }, [page, totalPages])

  const pageItems = useMemo(() => {
    const start = (page - 1) * pageSize
    return items.slice(start, start + pageSize)
  }, [items, page, pageSize])

  function setPageSize(size: number) {
    setPageSizeState(size)
    localStorage.setItem(`kkcert_page_size_${key}`, String(size))
    setPage(1)
  }

  return {
    pageItems,
    page,
    setPage,
    pageSize,
    setPageSize,
    pageSizes: PAGE_SIZES,
    total,
    totalPages,
    showPagination: total > pageSize,
  }
}
