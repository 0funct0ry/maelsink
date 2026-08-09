import { render, screen, fireEvent } from '@testing-library/react'
import Modal from './Modal'

describe('Modal', () => {
  it('renders nothing when closed', () => {
    render(
      <Modal open={false} onClose={() => {}}>
        <div>content</div>
      </Modal>,
    )
    expect(screen.queryByText('content')).not.toBeInTheDocument()
  })

  it('renders children when open', () => {
    render(
      <Modal open onClose={() => {}}>
        <div>content</div>
      </Modal>,
    )
    expect(screen.getByText('content')).toBeInTheDocument()
  })

  it('closes on Escape key by default', () => {
    const onClose = vi.fn()
    render(
      <Modal open onClose={onClose}>
        <div>content</div>
      </Modal>,
    )
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('does not close on Escape when dismissable is false', () => {
    const onClose = vi.fn()
    render(
      <Modal open onClose={onClose} dismissable={false}>
        <div>content</div>
      </Modal>,
    )
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).not.toHaveBeenCalled()
  })

  it('closes on backdrop click', () => {
    const onClose = vi.fn()
    render(
      <Modal open onClose={onClose}>
        <div>content</div>
      </Modal>,
    )
    // The backdrop is the outer fixed container; click it directly.
    fireEvent.click(screen.getByText('content').parentElement!.parentElement!)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('does not close when clicking inside the card', () => {
    const onClose = vi.fn()
    render(
      <Modal open onClose={onClose}>
        <div>content</div>
      </Modal>,
    )
    fireEvent.click(screen.getByText('content'))
    expect(onClose).not.toHaveBeenCalled()
  })

  it('does not close on backdrop click when dismissable is false', () => {
    const onClose = vi.fn()
    render(
      <Modal open onClose={onClose} dismissable={false}>
        <div>content</div>
      </Modal>,
    )
    fireEvent.click(screen.getByText('content').parentElement!.parentElement!)
    expect(onClose).not.toHaveBeenCalled()
  })

  it('moves focus into the modal card on open', () => {
    render(
      <Modal open onClose={() => {}}>
        <div>content</div>
      </Modal>,
    )
    expect(screen.getByText('content').parentElement).toHaveFocus()
  })
})
