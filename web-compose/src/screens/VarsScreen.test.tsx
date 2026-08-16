import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import VarsScreen from './VarsScreen'
import { useVarsStore } from '../stores/useVarsStore'

function makeMemoryStorage(): Storage {
  const data = new Map<string, string>()
  return {
    getItem: (key: string) => (data.has(key) ? data.get(key)! : null),
    setItem: (key: string, value: string) => data.set(key, value),
    removeItem: (key: string) => data.delete(key),
    clear: () => data.clear(),
    key: (index: number) => Array.from(data.keys())[index] ?? null,
    get length() {
      return data.size
    },
  } as Storage
}

beforeEach(() => {
  vi.stubGlobal('localStorage', makeMemoryStorage())
  useVarsStore.setState({ vars: {} })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('VarsScreen', () => {
  it('adds a row via the add-row UI', () => {
    render(<VarsScreen />)

    fireEvent.change(screen.getByLabelText('Key'), { target: { value: 'foo' } })
    fireEvent.change(screen.getByLabelText('Value'), { target: { value: 'bar' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add' }))

    expect(screen.getByText('foo')).toBeInTheDocument()
    expect(useVarsStore.getState().vars).toEqual({ foo: 'bar' })
  })

  it('edits a row value', () => {
    useVarsStore.getState().setVar('foo', 'bar')
    render(<VarsScreen />)

    const input = screen.getByDisplayValue('bar')
    fireEvent.change(input, { target: { value: 'baz' } })

    expect(useVarsStore.getState().vars.foo).toBe('baz')
  })

  it('deletes a row', () => {
    useVarsStore.getState().setVar('foo', 'bar')
    render(<VarsScreen />)

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))

    expect(screen.queryByText('foo')).not.toBeInTheDocument()
    expect(useVarsStore.getState().vars).toEqual({})
  })
})
