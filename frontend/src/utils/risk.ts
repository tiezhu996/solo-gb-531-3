export type RiskLevel = 'low' | 'medium' | 'high'

export function riskLevel(rank: string, numeric: number): RiskLevel {
  if (['critical', 'extreme', 'high', '重大', '高'].includes(rank.toLowerCase()) || numeric >= 16) return 'high'
  if (['medium', 'moderate', '中'].includes(rank.toLowerCase()) || numeric >= 8) return 'medium'
  return 'low'
}
