/** 登录页 / 后台共用背景：网格拓扑 + 链路脉冲（运维架构风格） */
export function AnimatedBackground({ className = '' }: { className?: string }) {
  return (
    <div className={`ambient-bg ${className}`} aria-hidden>
      <div className="ambient-topology" />
      <div className="ambient-link ambient-link-1" />
      <div className="ambient-link ambient-link-2" />
      <div className="ambient-link ambient-link-3" />
      <div className="ambient-packet ambient-packet-1" />
      <div className="ambient-packet ambient-packet-2" />
      <div className="ambient-node ambient-node-1" />
      <div className="ambient-node ambient-node-2" />
      <div className="ambient-node ambient-node-3" />
      <div className="ambient-node ambient-node-4" />
      <div className="ambient-node ambient-node-5" />
    </div>
  )
}
