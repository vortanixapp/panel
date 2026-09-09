"use client";

import { createContext, useContext, useEffect, useState, useMemo } from 'react'
import { getCookie, setCookie, removeCookie } from '@/lib/cookies'

type Theme = 'dark' | 'light' | 'system'
type ResolvedTheme = Exclude<Theme, 'system'>

const DEFAULT_THEME = 'system'
const THEME_COOKIE_NAME = 'vite-ui-theme'
const THEME_COOKIE_MAX_AGE = 60 * 60 * 24 * 365 

type ThemeProviderProps = {
  children: React.ReactNode
  defaultTheme?: Theme
  storageKey?: string
}

type ThemeProviderState = {
  defaultTheme: Theme
  resolvedTheme: ResolvedTheme
  theme: Theme
  setTheme: (theme: Theme, origin?: { x: number; y: number }) => void
  resetTheme: () => void
}

const initialState: ThemeProviderState = {
  defaultTheme: DEFAULT_THEME,
  resolvedTheme: 'light',
  theme: DEFAULT_THEME,
  setTheme: () => null,
  resetTheme: () => null,
}

const ThemeContext = createContext<ThemeProviderState>(initialState)

export function ThemeProvider({
  children,
  defaultTheme = DEFAULT_THEME,
  storageKey = THEME_COOKIE_NAME,
  ...props
}: ThemeProviderProps) {
  const [theme, _setTheme] = useState<Theme>(
    () => (getCookie(storageKey) as Theme) || defaultTheme
  )

  const resolvedTheme = useMemo((): ResolvedTheme => {
    if (theme === 'system') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches
        ? 'dark'
        : 'light'
    }
    return theme as ResolvedTheme
  }, [theme])

  useEffect(() => {
    const root = window.document.documentElement
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

    const applyTheme = (currentResolvedTheme: ResolvedTheme) => {
      root.classList.remove('light', 'dark') 
      root.classList.add(currentResolvedTheme) 
    }

    const handleChange = () => {
      if (theme === 'system') {
        const systemTheme = mediaQuery.matches ? 'dark' : 'light'
        applyTheme(systemTheme)
      }
    }

    applyTheme(resolvedTheme)

    mediaQuery.addEventListener('change', handleChange)

    return () => mediaQuery.removeEventListener('change', handleChange)
  }, [theme, resolvedTheme])

  const setTheme = (theme: Theme, origin?: { x: number; y: number }) => {
    setCookie(storageKey, theme, THEME_COOKIE_MAX_AGE)

    const root = document.documentElement
    const next: ResolvedTheme =
      theme === 'system'
        ? window.matchMedia('(prefers-color-scheme: dark)').matches
          ? 'dark'
          : 'light'
        : (theme as ResolvedTheme)

    const start = (
      document as Document & {
        startViewTransition?: (cb: () => void) => { finished: Promise<void> }
      }
    ).startViewTransition

    if (!origin || typeof start !== 'function') {
      _setTheme(theme)
      return
    }

    const { x, y } = origin
    const radius = Math.hypot(
      Math.max(x, window.innerWidth - x),
      Math.max(y, window.innerHeight - y)
    )

    root.style.setProperty('--vx-theme-x', `${x}px`)
    root.style.setProperty('--vx-theme-y', `${y}px`)
    root.style.setProperty('--vx-theme-r', `${radius}px`)
    root.dataset.themeSwitching = 'true'

    const transition = start.call(document, () => {
      root.classList.remove('light', 'dark')
      root.classList.add(next)
      _setTheme(theme)
    })

    void transition.finished
      .catch(() => undefined)
      .finally(() => {
        delete root.dataset.themeSwitching
        root.style.removeProperty('--vx-theme-x')
        root.style.removeProperty('--vx-theme-y')
        root.style.removeProperty('--vx-theme-r')
      })
  }

  const resetTheme = () => {
    removeCookie(storageKey)
    _setTheme(DEFAULT_THEME)
  }

  const contextValue = {
    defaultTheme,
    resolvedTheme,
    resetTheme,
    theme,
    setTheme,
  }

  return (
    <ThemeContext value={contextValue} {...props}>
      {children}
    </ThemeContext>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export const useTheme = () => {
  const context = useContext(ThemeContext)

  if (!context) throw new Error('useTheme must be used within a ThemeProvider')

  return context
}
