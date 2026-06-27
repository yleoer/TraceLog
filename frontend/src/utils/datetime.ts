// Centralized date/time formatting.
// Timestamps are stored in UTC; we render them in a single, configurable
// timezone (Asia/Shanghai by default). Override with VITE_APP_TIMEZONE.
export const APP_TIMEZONE = (import.meta.env.VITE_APP_TIMEZONE as string | undefined) || 'Asia/Shanghai'

function format(value: string, options: Intl.DateTimeFormatOptions): string {
  if (!value) return '-'
  const timestamp = Date.parse(value)
  if (Number.isNaN(timestamp)) return value
  return new Intl.DateTimeFormat('zh-CN', { ...options, timeZone: APP_TIMEZONE }).format(new Date(timestamp))
}

export function formatDateTime(value: string): string {
  return format(value, { dateStyle: 'short', timeStyle: 'short' })
}

export function formatDate(value: string): string {
  return format(value, { dateStyle: 'short' })
}

export function formatTime(value: string): string {
  return format(value, { timeStyle: 'short' })
}
