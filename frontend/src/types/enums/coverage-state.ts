export const coverageStates = ['queued', 'running', 'completed', 'failed', 'confirmed', 'voided'] as const
export type CoverageState = (typeof coverageStates)[number]

export const coverageStateLabels: Record<CoverageState, string> = {
  queued: '排队中',
  running: '计算中',
  completed: '待确认',
  failed: '失败',
  confirmed: '已确认',
  voided: '已作废',
}
