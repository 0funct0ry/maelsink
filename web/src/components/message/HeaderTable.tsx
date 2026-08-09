import type { HeaderEntry } from '../../lib/apiTypes'

interface HeaderTableProps {
  headers: HeaderEntry[]
}

export default function HeaderTable({ headers }: HeaderTableProps) {
  return (
    <table className="w-full border-collapse text-sm">
      <tbody>
        {headers.map((header, index) => (
          <tr key={`${header.name}-${index}`} className="border-b border-border-soft last:border-b-0">
            <td className="w-[170px] flex-none align-top py-[9px] pr-4 font-mono text-xs text-text-tertiary">
              {header.name}
            </td>
            <td className="break-all py-[9px] font-mono text-xs text-text-primary">{header.value}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
