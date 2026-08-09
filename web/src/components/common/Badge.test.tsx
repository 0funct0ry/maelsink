import { render, screen } from '@testing-library/react'
import Badge from './Badge'

describe('Badge', () => {
  it('renders children', () => {
    render(<Badge>Delivered</Badge>)
    expect(screen.getByText('Delivered')).toBeInTheDocument()
  })

  it('defaults to the default variant', () => {
    render(<Badge>x</Badge>)
    expect(screen.getByText('x').className).toMatch(/surface-2/)
  })

  it.each([
    ['success', 'success'],
    ['warning', 'warning'],
    ['danger', 'danger'],
    ['accent', 'accent'],
  ] as const)('applies %s variant classes', (variant, expected) => {
    render(<Badge variant={variant}>x</Badge>)
    expect(screen.getByText('x').className).toMatch(new RegExp(expected))
  })
})
