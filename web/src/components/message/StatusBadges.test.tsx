import { render, screen } from '@testing-library/react'
import StatusBadges from './StatusBadges'

describe('StatusBadges', () => {
  it('always renders a Captured badge and a size badge', () => {
    render(
      <StatusBadges
        message={{ parse_warning: false, has_attachments: false, attachment_count: 0, size_bytes: 1024 }}
      />,
    )
    expect(screen.getByText('Captured')).toBeInTheDocument()
    expect(screen.getByText('1.0 KB')).toBeInTheDocument()
    expect(screen.queryByText(/Parse warning/)).not.toBeInTheDocument()
    expect(screen.queryByText(/attachment/)).not.toBeInTheDocument()
  })

  it('renders a parse warning badge', () => {
    render(
      <StatusBadges
        message={{ parse_warning: true, has_attachments: false, attachment_count: 0, size_bytes: 100 }}
      />,
    )
    expect(screen.getByText(/Parse warning/)).toBeInTheDocument()
  })

  it('renders an attachment count badge', () => {
    render(
      <StatusBadges
        message={{ parse_warning: false, has_attachments: true, attachment_count: 3, size_bytes: 100 }}
      />,
    )
    expect(screen.getByText(/3 attachments/)).toBeInTheDocument()
  })

  it('renders all badges when applicable', () => {
    render(
      <StatusBadges
        message={{ parse_warning: true, has_attachments: true, attachment_count: 1, size_bytes: 100 }}
      />,
    )
    expect(screen.getByText('Captured')).toBeInTheDocument()
    expect(screen.getByText(/Parse warning/)).toBeInTheDocument()
    expect(screen.getByText(/1 attachment/)).toBeInTheDocument()
  })
})
