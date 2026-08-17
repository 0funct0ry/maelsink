import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import HeaderStrip from './HeaderStrip'
import { useConnectionStore } from '../stores/useConnectionStore'

beforeEach(() => {
  useConnectionStore.setState({ status: 'red', lastChecked: null, lastError: null })
})

afterEach(() => {
  useConnectionStore.setState({ status: 'red', lastChecked: null, lastError: null })
})

describe('HeaderStrip', () => {
  it('shows the unreachable label when disconnected', () => {
    render(<HeaderStrip />)
    expect(screen.getByText(/Target unreachable/)).toBeInTheDocument()
  })

  it('shows the connected label when green', () => {
    useConnectionStore.setState({ status: 'green' })
    render(<HeaderStrip />)
    expect(screen.getByText('Connected to target')).toBeInTheDocument()
  })

  it('shows the reachable-but-erroring label when yellow', () => {
    useConnectionStore.setState({ status: 'yellow' })
    render(<HeaderStrip />)
    expect(screen.getByText('Target reachable but reporting errors')).toBeInTheDocument()
  })
})
