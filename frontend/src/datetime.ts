const TZ = 'Asia/Shanghai'
const LOCALE = 'zh-CN'

export function formatDateTime(iso: string) {
  return new Date(iso).toLocaleString(LOCALE, { timeZone: TZ })
}

export function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString(LOCALE, { timeZone: TZ })
}
