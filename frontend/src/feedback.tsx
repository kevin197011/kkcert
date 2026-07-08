import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  ReactNode,
} from 'react'

export type FeedbackKind = 'info' | 'ok' | 'err'

type ToastItem = {
  id: string
  kind: FeedbackKind
  title?: string
  message: string
}

type ConfirmOpts = {
  title: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  danger?: boolean
}

type ConfirmState = ConfirmOpts & {
  resolve: (ok: boolean) => void
}

type FeedbackCtx = {
  toast: (opts: { kind: FeedbackKind; title?: string; message: string; duration?: number }) => void
  confirm: (opts: ConfirmOpts) => Promise<boolean>
}

const FeedbackContext = createContext<FeedbackCtx | null>(null)

let toastSeq = 0

function ToastIcon({ kind }: { kind: FeedbackKind }) {
  const paths = {
    info: <path d="M12 8v4m0 4h.01M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18z" />,
    ok: <path d="M9 12l2 2 4-4m6 2a9 9 0 1 1-18 0 9 9 0 0 1 18 0z" />,
    err: <path d="M12 9v4m0 4h.01M10.29 3.86L2.82 17a2 2 0 0 0 1.71 3h14.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />,
  }
  return (
    <span className={`toast-icon toast-icon-${kind}`} aria-hidden>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        {paths[kind]}
      </svg>
    </span>
  )
}

function ConfirmModal({
  state,
  onClose,
}: {
  state: ConfirmState
  onClose: (ok: boolean) => void
}) {
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose(false)
    }
    document.addEventListener('keydown', onKey)
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = ''
    }
  }, [onClose])

  return (
    <div className="confirm-root">
      <button type="button" className="confirm-backdrop" aria-label="取消" onClick={() => onClose(false)} />
      <div className="confirm-dialog" role="alertdialog" aria-modal="true" aria-labelledby="confirm-title">
        <h3 id="confirm-title" className="confirm-title">{state.title}</h3>
        <p className="confirm-message">{state.message}</p>
        <div className="confirm-actions">
          <button type="button" className="btn btn-secondary" onClick={() => onClose(false)}>
            {state.cancelLabel ?? '取消'}
          </button>
          <button
            type="button"
            className={state.danger ? 'btn btn-danger' : 'btn'}
            onClick={() => onClose(true)}
          >
            {state.confirmLabel ?? '确认'}
          </button>
        </div>
      </div>
    </div>
  )
}

export function FeedbackProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([])
  const [confirm, setConfirm] = useState<ConfirmState | null>(null)
  const timers = useRef<Map<string, number>>(new Map())

  const dismissToast = useCallback((id: string) => {
    const t = timers.current.get(id)
    if (t) window.clearTimeout(t)
    timers.current.delete(id)
    setToasts(prev => prev.filter(x => x.id !== id))
  }, [])

  const toast = useCallback((opts: { kind: FeedbackKind; title?: string; message: string; duration?: number }) => {
    const id = `t-${++toastSeq}`
    setToasts(prev => [...prev, { id, kind: opts.kind, title: opts.title, message: opts.message }])
    const ms = opts.duration ?? (opts.kind === 'err' ? 6000 : 4000)
    const timer = window.setTimeout(() => dismissToast(id), ms)
    timers.current.set(id, timer)
  }, [dismissToast])

  const askConfirm = useCallback((opts: ConfirmOpts) => {
    return new Promise<boolean>(resolve => {
      setConfirm({ ...opts, resolve })
    })
  }, [])

  function closeConfirm(ok: boolean) {
    confirm?.resolve(ok)
    setConfirm(null)
  }

  return (
    <FeedbackContext.Provider value={{ toast, confirm: askConfirm }}>
      {children}
      {confirm && <ConfirmModal state={confirm} onClose={closeConfirm} />}
      <div className="toast-stack" aria-live="polite">
        {toasts.map(t => (
          <div key={t.id} className={`toast toast-${t.kind}`} role="status">
            <ToastIcon kind={t.kind} />
            <div className="toast-body">
              {t.title && <div className="toast-title">{t.title}</div>}
              <div className="toast-message">{t.message}</div>
            </div>
            <button type="button" className="toast-close" onClick={() => dismissToast(t.id)} aria-label="关闭">×</button>
          </div>
        ))}
      </div>
    </FeedbackContext.Provider>
  )
}

export function useFeedback() {
  const ctx = useContext(FeedbackContext)
  if (!ctx) throw new Error('useFeedback must be used within FeedbackProvider')
  return ctx
}
