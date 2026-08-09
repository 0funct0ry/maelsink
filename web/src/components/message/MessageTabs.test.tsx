import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import MessageTabs from './MessageTabs'
import * as apiClient from '../../lib/apiClient'
import type { MessageDetail } from '../../lib/apiTypes'

vi.mock('../../lib/apiClient', () => ({
  getRaw: vi.fn(),
}))

const message: MessageDetail = {
  id: 'm1',
  from: 'a@example.com',
  to: ['b@example.com'],
  cc: [],
  subject: 'Hello',
  size_bytes: 100,
  has_attachments: false,
  attachment_count: 0,
  received_at: '2024-01-01T00:00:00Z',
  parse_warning: false,
  read: true,
  headers: [{ name: 'From', value: 'a@example.com' }],
  text_body: 'plain text body',
  html_body: '<p>hi</p>',
  attachments: [],
  raw_size_bytes: 100,
}

describe('MessageTabs', () => {
  it('shows tabs in the required order, defaulting to Rendered HTML', () => {
    render(<MessageTabs message={message} />)
    const tabs = screen.getAllByRole('tab')
    expect(tabs.map((t) => t.textContent)).toEqual(['Rendered HTML', 'Plain Text', 'Raw Source', 'Headers'])
    expect(tabs[0]).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByTitle('Rendered message preview')).toBeInTheDocument()
  })

  it('does not include allow-scripts in the iframe sandbox', () => {
    render(<MessageTabs message={message} />)
    const iframe = screen.getByTitle('Rendered message preview')
    expect(iframe.getAttribute('sandbox')).toBe('allow-same-origin')
  })

  it('switches to Plain Text and renders the body', () => {
    render(<MessageTabs message={message} />)
    fireEvent.click(screen.getByText('Plain Text'))
    expect(screen.getByText('plain text body')).toBeInTheDocument()
  })

  it('lazily fetches raw source only when that tab is activated', async () => {
    vi.mocked(apiClient.getRaw).mockResolvedValue('RAW SOURCE CONTENT')
    render(<MessageTabs message={message} />)
    expect(apiClient.getRaw).not.toHaveBeenCalled()

    fireEvent.click(screen.getByText('Raw Source'))
    expect(apiClient.getRaw).toHaveBeenCalledWith('m1')
    await waitFor(() => expect(screen.getByText('RAW SOURCE CONTENT')).toBeInTheDocument())

    fireEvent.click(screen.getByText('Plain Text'))
    fireEvent.click(screen.getByText('Raw Source'))
    expect(apiClient.getRaw).toHaveBeenCalledTimes(1)
  })

  it('shows headers in the Headers tab', () => {
    render(<MessageTabs message={message} />)
    fireEvent.click(screen.getByText('Headers'))
    expect(screen.getByText('From')).toBeInTheDocument()
  })
})
