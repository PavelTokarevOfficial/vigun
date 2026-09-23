export type DateRangeValue = {
  start: string
  end: string
}

export function dateInputValue(daysAgo = 0) {
  const date = new Date()
  date.setHours(12, 0, 0, 0)
  date.setDate(date.getDate() - daysAgo)
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${date.getFullYear()}-${month}-${day}`
}

export function formatDateRange(value: DateRangeValue) {
  const formatter = new Intl.DateTimeFormat('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    timeZone: 'UTC',
  })
  return `${formatter.format(new Date(`${value.start}T00:00:00Z`))} — ${formatter.format(new Date(`${value.end}T00:00:00Z`))}`
}
