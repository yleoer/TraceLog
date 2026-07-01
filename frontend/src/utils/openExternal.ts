import { BrowserOpenURL } from '../wailsjs/runtime/runtime'

export function openExternalURL(url: string) {
  const value = String(url || '').trim()
  if (!isExternalURL(value)) return
  BrowserOpenURL(value)
}

export function openExternalClick(url: string) {
  return (event: MouseEvent) => {
    event.preventDefault()
    openExternalURL(url)
  }
}

export function isExternalURL(url: string) {
  return /^(https?:|mailto:)/i.test(String(url || '').trim())
}
