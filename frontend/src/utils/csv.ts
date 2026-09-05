export function escapeCsvCell(value: unknown): string {
  const text = value == null ? '' : String(value)
  return /[",\r\n]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text
}

export function buildCsv(rows: unknown[][]): string {
  return '\uFEFF' + rows.map((row) => row.map(escapeCsvCell).join(',')).join('\r\n')
}
