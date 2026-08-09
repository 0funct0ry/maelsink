import { render, screen, fireEvent } from '@testing-library/react'
import ConfirmDialog from './ConfirmDialog'

describe('ConfirmDialog', () => {
  it('renders nothing when closed', () => {
    render(
      <ConfirmDialog open={false} onClose={() => {}} onConfirm={() => {}} title="Title" body="Body" />,
    )
    expect(screen.queryByText('Title')).not.toBeInTheDocument()
  })

  it('renders title and body when open', () => {
    render(<ConfirmDialog open onClose={() => {}} onConfirm={() => {}} title="Clear all?" body="This deletes everything." />)
    expect(screen.getByText('Clear all?')).toBeInTheDocument()
    expect(screen.getByText('This deletes everything.')).toBeInTheDocument()
  })

  it('uses default labels', () => {
    render(<ConfirmDialog open onClose={() => {}} onConfirm={() => {}} title="t" body="b" />)
    expect(screen.getByRole('button', { name: 'Confirm' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()
  })

  it('uses custom labels', () => {
    render(
      <ConfirmDialog
        open
        onClose={() => {}}
        onConfirm={() => {}}
        title="t"
        body="b"
        confirmLabel="Delete"
        cancelLabel="Nope"
      />,
    )
    expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Nope' })).toBeInTheDocument()
  })

  it('calls onClose (not onConfirm) when cancel clicked', () => {
    const onClose = vi.fn()
    const onConfirm = vi.fn()
    render(<ConfirmDialog open onClose={onClose} onConfirm={onConfirm} title="t" body="b" />)
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onClose).toHaveBeenCalledTimes(1)
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('calls onConfirm then onClose when confirm clicked', () => {
    const calls: string[] = []
    const onClose = vi.fn(() => calls.push('close'))
    const onConfirm = vi.fn(() => calls.push('confirm'))
    render(<ConfirmDialog open onClose={onClose} onConfirm={onConfirm} title="t" body="b" />)
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))
    expect(calls).toEqual(['confirm', 'close'])
  })

  it('applies danger styling to the confirm button when danger is true', () => {
    render(<ConfirmDialog open onClose={() => {}} onConfirm={() => {}} title="t" body="b" danger />)
    expect(screen.getByRole('button', { name: 'Confirm' }).className).toMatch(/danger/)
  })

  it('does not apply danger styling by default', () => {
    render(<ConfirmDialog open onClose={() => {}} onConfirm={() => {}} title="t" body="b" />)
    expect(screen.getByRole('button', { name: 'Confirm' }).className).not.toMatch(/danger/)
  })
})
