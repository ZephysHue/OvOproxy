import { ref } from 'vue'

export type ThemeMode = 'system' | 'dark' | 'light'

const stored = (localStorage.getItem('themeMode') || '') as ThemeMode
export const themeMode = ref<ThemeMode>(
  stored === 'dark' || stored === 'light' || stored === 'system' ? stored : 'system',
)

const media = window.matchMedia('(prefers-color-scheme: dark)')

function resolvedTheme(mode: ThemeMode): 'dark' | 'light' {
  if (mode === 'system') {
    return media.matches ? 'dark' : 'light'
  }
  return mode
}

export function applyTheme() {
  const next = resolvedTheme(themeMode.value)
  document.documentElement.setAttribute('data-theme', next)
  document.body.setAttribute('data-theme', next)
}

export function setThemeMode(next: ThemeMode) {
  themeMode.value = next
  localStorage.setItem('themeMode', next)
  applyTheme()
}

let inited = false
export function initTheme() {
  if (inited) return
  inited = true
  applyTheme()
  media.addEventListener('change', () => {
    if (themeMode.value === 'system') {
      applyTheme()
    }
  })
}
