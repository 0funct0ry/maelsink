import { create } from 'zustand'

export type Theme = 'light' | 'dark' | 'system'

const THEME_STORAGE_KEY = 'maelsink-compose-theme'

function readStoredTheme(): Theme {
  try {
    const v = window.localStorage.getItem(THEME_STORAGE_KEY)
    if (v === 'light' || v === 'dark' || v === 'system') return v
  } catch {
    // localStorage unavailable — fall through to the default.
  }
  return 'system'
}

function writeStoredTheme(theme: Theme): void {
  try {
    window.localStorage.setItem(THEME_STORAGE_KEY, theme)
  } catch {
    // localStorage unavailable — the choice just won't survive a reload.
  }
}

// Applies theme to the document root as a data-theme attribute, which
// src/index.css's [data-theme="dark"]/[data-theme="light"] blocks key off
// of. 'system' removes the attribute entirely so the prefers-color-scheme
// media query (also in index.css) takes over. Mirrors web/'s useUIStore
// theme handling.
function applyTheme(theme: Theme): void {
  const root = window.document.documentElement
  if (theme === 'system') {
    root.removeAttribute('data-theme')
  } else {
    root.setAttribute('data-theme', theme)
  }
}

interface ThemeState {
  theme: Theme
  setTheme: (theme: Theme) => void
}

export const useThemeStore = create<ThemeState>((set) => ({
  theme: readStoredTheme(),
  setTheme: (theme) => {
    writeStoredTheme(theme)
    applyTheme(theme)
    set({ theme })
  },
}))

// Apply whatever theme was already persisted as soon as the module loads,
// so the very first paint uses it instead of a light-mode flash.
applyTheme(useThemeStore.getState().theme)
