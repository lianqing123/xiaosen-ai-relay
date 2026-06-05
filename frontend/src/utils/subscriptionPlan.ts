export type PlanValidityUnit = 'day' | 'week' | 'month' | 'year'

export function normalizePlanValidityUnit(unit: string | null | undefined): PlanValidityUnit {
  switch ((unit || '').trim().toLowerCase()) {
    case 'week':
    case 'weeks':
      return 'week'
    case 'month':
    case 'months':
      return 'month'
    case 'year':
    case 'years':
      return 'year'
    case 'day':
    case 'days':
    default:
      return 'day'
  }
}

export function getPlanValidityDays(value: number, unit: string | null | undefined): number {
  const normalizedValue = Math.max(0, Math.floor(Number(value) || 0))
  switch (normalizePlanValidityUnit(unit)) {
    case 'week':
      return normalizedValue * 7
    case 'month':
      return normalizedValue * 30
    case 'year':
      return normalizedValue * 365
    default:
      return normalizedValue
  }
}
