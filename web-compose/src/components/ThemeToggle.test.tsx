import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import ThemeToggle from './ThemeToggle'
import { useThemeStore } from '../stores/useThemeStore'

beforeEach(() => {
  useThemeStore.setState({ theme: 'system' })
  document.documentElement.removeAttribute('data-theme')
})

afterEach(() => {
  document.documentElement.removeAttribute('data-theme')
})

describe('ThemeToggle', () => {
  it('marks the active theme as pressed', () => {
    useThemeStore.setState({ theme: 'dark' })
    render(<ThemeToggle />)
    expect(screen.getByTitle('Dark')).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByTitle('Light')).toHaveAttribute('aria-pressed', 'false')
  })

  it('switches theme on click', () => {
    render(<ThemeToggle />)
    fireEvent.click(screen.getByTitle('Dark'))
    expect(useThemeStore.getState().theme).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
  })
})
