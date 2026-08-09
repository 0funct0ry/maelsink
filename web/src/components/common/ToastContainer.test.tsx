import { render, screen, fireEvent } from '@testing-library/react'
import ToastContainer from './ToastContainer'
import { useUIStore } from '../../stores/useUIStore'

afterEach(() => {
  useUIStore.setState({ toasts: [] })
})

describe('ToastContainer', () => {
  it('renders nothing when there are no toasts', () => {
    render(<ToastContainer />)
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('renders multiple toasts from the store', () => {
    useUIStore.setState({
      toasts: [
        { id: 't1', variant: 'info', message: 'first' },
        { id: 't2', variant: 'success', message: 'second' },
      ],
    })
    render(<ToastContainer />)
    expect(screen.getByText('first')).toBeInTheDocument()
    expect(screen.getByText('second')).toBeInTheDocument()
  })

  it('dismissing one toast does not affect the others', () => {
    useUIStore.setState({
      toasts: [
        { id: 't1', variant: 'info', message: 'first' },
        { id: 't2', variant: 'success', message: 'second' },
      ],
    })
    render(<ToastContainer />)
    const dismissButtons = screen.getAllByRole('button', { name: 'Dismiss' })
    fireEvent.click(dismissButtons[0])

    expect(screen.queryByText('first')).not.toBeInTheDocument()
    expect(screen.getByText('second')).toBeInTheDocument()
    expect(useUIStore.getState().toasts).toHaveLength(1)
  })
})
