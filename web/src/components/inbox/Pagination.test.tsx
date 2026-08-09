import { render, screen, fireEvent } from '@testing-library/react'
import Pagination from './Pagination'
import { useMessageStore } from '../../stores/useMessageStore'

describe('Pagination', () => {
  it('shows the current range and total', () => {
    useMessageStore.setState({ offset: 0, limit: 50, total: 120 })
    render(<Pagination />)
    expect(screen.getByText('1-50 of 120')).toBeInTheDocument()
  })

  it('disables Previous on the first page', () => {
    useMessageStore.setState({ offset: 0, limit: 50, total: 120 })
    render(<Pagination />)
    expect(screen.getByLabelText('Previous page')).toBeDisabled()
    expect(screen.getByLabelText('Next page')).not.toBeDisabled()
  })

  it('disables Next on the last page', () => {
    useMessageStore.setState({ offset: 100, limit: 50, total: 120 })
    render(<Pagination />)
    expect(screen.getByLabelText('Next page')).toBeDisabled()
    expect(screen.getByLabelText('Previous page')).not.toBeDisabled()
  })

  it('calls setPage with offset - limit on Previous click', () => {
    const setPage = vi.fn()
    useMessageStore.setState({ offset: 50, limit: 50, total: 120, setPage })
    render(<Pagination />)
    fireEvent.click(screen.getByLabelText('Previous page'))
    expect(setPage).toHaveBeenCalledWith(0)
  })

  it('calls setPage with offset + limit on Next click', () => {
    const setPage = vi.fn()
    useMessageStore.setState({ offset: 0, limit: 50, total: 120, setPage })
    render(<Pagination />)
    fireEvent.click(screen.getByLabelText('Next page'))
    expect(setPage).toHaveBeenCalledWith(50)
  })

  it('clamps Previous to 0 when offset is smaller than limit', () => {
    const setPage = vi.fn()
    useMessageStore.setState({ offset: 20, limit: 50, total: 120, setPage })
    render(<Pagination />)
    fireEvent.click(screen.getByLabelText('Previous page'))
    expect(setPage).toHaveBeenCalledWith(0)
  })
})
