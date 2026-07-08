import { ReactNode } from 'react'

export type NoticeKind = 'info' | 'ok' | 'err'

const labels: Record<NoticeKind, string> = {
  info: '进行中',
  ok: '成功',
  err: '注意',
}

function NoticeIcon({ kind }: { kind: NoticeKind }) {
  const paths = {
    info: <path d="M12 8v4m0 4h.01M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18z" />,
    ok: <path d="M9 12l2 2 4-4m6 2a9 9 0 1 1-18 0 9 9 0 0 1 18 0z" />,
    err: <path d="M12 9v4m0 4h.01M10.29 3.86L2.82 17a2 2 0 0 0 1.71 3h14.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />,
  }
  return (
    <span className={`notice-icon notice-icon-${kind}`} aria-hidden>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        {paths[kind]}
      </svg>
    </span>
  )
}

export function Notice({
  kind,
  title,
  children,
  onClose,
}: {
  kind: NoticeKind
  title?: string
  children: ReactNode
  onClose?: () => void
}) {
  return (
    <div className={`notice notice-${kind}`} role={kind === 'err' ? 'alert' : 'status'}>
      <NoticeIcon kind={kind} />
      <div className="notice-body">
        <div className="notice-label">{labels[kind]}</div>
        {title && <div className="notice-title">{title}</div>}
        <div className="notice-message">{children}</div>
      </div>
      {onClose && (
        <button type="button" className="notice-close" onClick={onClose} aria-label="关闭">×</button>
      )}
    </div>
  )
}
