import { api, json, query } from './client'
import { normalizePage, type PageData } from '../types/common'
import type { CoverageEvaluation, CoverageRunInput } from '../types/coverage-evaluation'

export async function listCoverageEvaluations(scenarioId?: number): Promise<PageData<CoverageEvaluation>> {
  return normalizePage(await api<PageData<CoverageEvaluation> | CoverageEvaluation[]>(`/coverage-evaluations${query({ scenario_id: scenarioId })}`))
}
export const getCoverageEvaluation = (id: number) => api<CoverageEvaluation>(`/coverage-evaluations/${id}`)
export const runCoverageEvaluation = (input: CoverageRunInput, key: string) => api<CoverageEvaluation>('/coverage-evaluations', json('POST', input, { 'Idempotency-Key': key }))
export const confirmCoverageEvaluation = (id: number) => api<CoverageEvaluation>(`/coverage-evaluations/${id}/confirm`, json('POST'))
export const voidCoverageEvaluation = (id: number) => api<CoverageEvaluation>(`/coverage-evaluations/${id}/void`, json('POST'))
export const replayCoverageEvaluation = (id: number) => api<CoverageEvaluation>(`/coverage-evaluations/${id}/replay`, json('POST'))
