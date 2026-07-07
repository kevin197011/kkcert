/** 登录页 / 后台共用的轻量动态背景（纯 CSS 动画） */
export function AnimatedBackground({ className = '' }: { className?: string }) {
  return (
    <div className={`ambient-bg ${className}`} aria-hidden>
      <div className="ambient-grid" />
      <div className="ambient-beam" />
      <div className="ambient-orb ambient-orb-1" />
      <div className="ambient-orb ambient-orb-2" />
      <div className="ambient-orb ambient-orb-3" />
      <div className="ambient-orb ambient-orb-4" />
    </div>
  )
}
