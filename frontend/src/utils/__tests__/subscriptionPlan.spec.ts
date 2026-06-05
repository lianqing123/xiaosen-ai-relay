import { describe, expect, it } from 'vitest'
import { getPlanValidityDays, normalizePlanValidityUnit } from '../subscriptionPlan'

describe('subscriptionPlan utilities', () => {
  it('normalizes canonical and legacy validity units', () => {
    expect(normalizePlanValidityUnit('day')).toBe('day')
    expect(normalizePlanValidityUnit('days')).toBe('day')
    expect(normalizePlanValidityUnit('week')).toBe('week')
    expect(normalizePlanValidityUnit('weeks')).toBe('week')
    expect(normalizePlanValidityUnit('month')).toBe('month')
    expect(normalizePlanValidityUnit('months')).toBe('month')
    expect(normalizePlanValidityUnit('year')).toBe('year')
    expect(normalizePlanValidityUnit('years')).toBe('year')
    expect(normalizePlanValidityUnit('  YEAR  ')).toBe('year')
    expect(normalizePlanValidityUnit('quarter')).toBe('day')
  })

  it('computes real subscription days for each supported unit', () => {
    expect(getPlanValidityDays(30, 'day')).toBe(30)
    expect(getPlanValidityDays(2, 'week')).toBe(14)
    expect(getPlanValidityDays(2, 'month')).toBe(60)
    expect(getPlanValidityDays(2, 'year')).toBe(730)
  })
})
