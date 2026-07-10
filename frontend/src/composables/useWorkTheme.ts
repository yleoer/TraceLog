import { computed, readonly, ref } from 'vue'

export type WorkThemeKey = 'slate' | 'indigo' | 'teal' | 'graphite'

export interface WorkTheme {
  key: WorkThemeKey
  label: string
  description: string
  primary: string
  hover: string
  pressed: string
  highlight: string
  deep: string
  soft: string
  rgb: string
}

export const workThemes: WorkTheme[] = [
  {
    key: 'slate',
    label: '深雾蓝',
    description: '沉稳、可靠，适合长时间专注',
    primary: '#416C9B',
    hover: '#365E89',
    pressed: '#2D5075',
    highlight: '#6F92B8',
    deep: '#365E89',
    soft: '#EAF1F7',
    rgb: '65, 108, 155'
  },
  {
    key: 'indigo',
    label: '靛青蓝',
    description: '清晰、现代，带一点科技感',
    primary: '#5367C8',
    hover: '#4659B4',
    pressed: '#3A4A98',
    highlight: '#7F8FE0',
    deep: '#4659B4',
    soft: '#EEF0FC',
    rgb: '83, 103, 200'
  },
  {
    key: 'teal',
    label: '深青绿',
    description: '自然、舒缓，降低视觉疲劳',
    primary: '#287C78',
    hover: '#216A67',
    pressed: '#1A5855',
    highlight: '#55A7A1',
    deep: '#216A67',
    soft: '#E8F4F3',
    rgb: '40, 124, 120'
  },
  {
    key: 'graphite',
    label: '石墨蓝',
    description: '中性、克制，突出内容本身',
    primary: '#516174',
    hover: '#435264',
    pressed: '#364250',
    highlight: '#7B8998',
    deep: '#435264',
    soft: '#EDF0F3',
    rgb: '81, 97, 116'
  }
]

const storageKey = 'tracelog:work-theme:v1'
const selectedThemeKey = ref<WorkThemeKey>(readStoredTheme())
const activeTheme = computed(() => workThemes.find((theme) => theme.key === selectedThemeKey.value) ?? workThemes[0])

applyThemeVariables(activeTheme.value)

export function useWorkTheme() {
  function setTheme(key: WorkThemeKey) {
    const theme = workThemes.find((item) => item.key === key)
    if (!theme) return
    selectedThemeKey.value = key
    applyThemeVariables(theme)
    try {
      localStorage.setItem(storageKey, key)
    } catch {
      // The embedded webview may disable localStorage in exceptional modes.
    }
  }

  return {
    themes: workThemes,
    selectedThemeKey: readonly(selectedThemeKey),
    activeTheme,
    setTheme
  }
}

function readStoredTheme(): WorkThemeKey {
  try {
    const stored = localStorage.getItem(storageKey)
    if (workThemes.some((theme) => theme.key === stored)) return stored as WorkThemeKey
  } catch {
    // Fall back to the recommended theme when storage is unavailable.
  }
  return 'slate'
}

function applyThemeVariables(theme: WorkTheme) {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  root.dataset.workTheme = theme.key
  root.style.setProperty('--accent', theme.primary)
  root.style.setProperty('--accent-hover', theme.hover)
  root.style.setProperty('--accent-pressed', theme.pressed)
  root.style.setProperty('--accent-highlight', theme.highlight)
  root.style.setProperty('--accent-deep', theme.deep)
  root.style.setProperty('--accent-soft', theme.soft)
  root.style.setProperty('--accent-rgb', theme.rgb)
}
