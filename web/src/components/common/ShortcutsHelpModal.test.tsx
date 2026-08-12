import { render, screen, fireEvent } from '@testing-library/react'
import ShortcutsHelpModal from './ShortcutsHelpModal'

describe('ShortcutsHelpModal', () => {
  it('is hidden until "?" is pressed', () => {
    render(<ShortcutsHelpModal />)
    expect(screen.queryByText('Keyboard shortcuts')).not.toBeInTheDocument()
  })

  it('opens on "?" and lists the existing shortcuts', () => {
    render(<ShortcutsHelpModal />)
    fireEvent.keyDown(window, { key: '?' })
    expect(screen.getByText('Keyboard shortcuts')).toBeInTheDocument()
    expect(screen.getByText('Focus the search box')).toBeInTheDocument()
    expect(screen.getByText('Back to inbox from a message, or close a dialog')).toBeInTheDocument()
  })

  it('does not open while typing in a text input', () => {
    render(
      <div>
        <input aria-label="some field" />
        <ShortcutsHelpModal />
      </div>,
    )
    screen.getByLabelText('some field').focus()
    fireEvent.keyDown(window, { key: '?' })
    expect(screen.queryByText('Keyboard shortcuts')).not.toBeInTheDocument()
  })

  it('closes on Escape', () => {
    render(<ShortcutsHelpModal />)
    fireEvent.keyDown(window, { key: '?' })
    expect(screen.getByText('Keyboard shortcuts')).toBeInTheDocument()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByText('Keyboard shortcuts')).not.toBeInTheDocument()
  })
})
