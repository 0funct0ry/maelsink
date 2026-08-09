import { render, screen, fireEvent } from '@testing-library/react'
import Toast from './Toast'

describe('Toast', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders the message', () => {
    render(<Toast variant="info" message="Hello" onDismiss={() => {}} />)
    expect(screen.getByText('Hello')).toBeInTheDocument()
  })

  it('renders info/success/danger variants', () => {
    const { rerender } = render(<Toast variant="info" message="m" onDismiss={() => {}} />)
    expect(screen.getByRole('status')).toBeInTheDocument()

    rerender(<Toast variant="success" message="m" onDismiss={() => {}} />)
    expect(screen.getByRole('status').className).toMatch(/success/)

    rerender(<Toast variant="danger" message="m" onDismiss={() => {}} />)
    expect(screen.getByRole('status').className).toMatch(/danger/)
  })

  it('calls onDismiss when the close button is clicked', () => {
    const onDismiss = vi.fn()
    render(<Toast variant="info" message="m" onDismiss={onDismiss} />)
    fireEvent.click(screen.getByRole('button', { name: 'Dismiss' }))
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('auto-dismisses after ~4000ms', () => {
    vi.useFakeTimers()
    const onDismiss = vi.fn()
    render(<Toast variant="info" message="m" onDismiss={onDismiss} />)
    expect(onDismiss).not.toHaveBeenCalled()
    vi.advanceTimersByTime(3999)
    expect(onDismiss).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })
})
