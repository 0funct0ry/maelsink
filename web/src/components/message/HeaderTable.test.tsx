import { render, screen } from '@testing-library/react'
import HeaderTable from './HeaderTable'

describe('HeaderTable', () => {
  it('renders headers in order, including duplicates', () => {
    render(
      <HeaderTable
        headers={[
          { name: 'From', value: 'a@example.com' },
          { name: 'Received', value: 'hop1' },
          { name: 'Received', value: 'hop2' },
        ]}
      />,
    )
    const rows = screen.getAllByRole('row')
    expect(rows).toHaveLength(3)
    expect(rows[1].textContent).toContain('hop1')
    expect(rows[2].textContent).toContain('hop2')
    expect(screen.getAllByText('Received')).toHaveLength(2)
  })

  it('renders nothing when there are no headers', () => {
    render(<HeaderTable headers={[]} />)
    expect(screen.queryAllByRole('row')).toHaveLength(0)
  })
})
