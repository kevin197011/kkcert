type LogoProps = { className?: string }

/** KKCert 品牌标识：盾形 + K 字，侧栏 / 登录页 / favicon 统一使用 */
export function Logo({ className }: LogoProps) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      <path d="M12 3l8 3v6c0 5-3.5 8.5-8 9-4.5-.5-8-4-8-9V6l8-3z" />
      <path d="M9.5 8v8M9.5 8l4.5 5.5M9.5 12l4 4" />
    </svg>
  )
}
