import { describe, expect, it } from 'vitest'
import { riskLevel } from './risk'

describe('riskLevel', () => {
  it('uses a 5x5 matrix when no rank is supplied', () => {
    expect(riskLevel('', 4)).toBe('low')
    expect(riskLevel('', 9)).toBe('medium')
    expect(riskLevel('', 20)).toBe('high')
  })

  it('honors explicit textual risk ranks', () => {
    expect(riskLevel('critical', 1)).toBe('high')
    expect(riskLevel('moderate', 1)).toBe('medium')
  })
})
