import { api, json, query } from './client'
import { normalizePage, type PageData } from '../types/common'
import type { Safeguard, SafeguardInput } from '../types/safeguard'

export async function listSafeguards(scenarioId?: number): Promise<PageData<Safeguard>> {
  return normalizePage(await api<PageData<Safeguard> | Safeguard[]>(`/safeguards${query({ scenario_id: scenarioId })}`))
}
export const getSafeguard = (id: number) => api<Safeguard>(`/safeguards/${id}`)
export const createSafeguard = (input: SafeguardInput) => api<Safeguard>('/safeguards', json('POST', input))
export const updateSafeguard = (id: number, input: SafeguardInput) => api<Safeguard>(`/safeguards/${id}`, json('PUT', input))
export const verifySafeguard = (id: number, evidence_note: string) => api<Safeguard>(`/safeguards/${id}/verify`, json('POST', { verified_at: new Date().toISOString(), evidence_note }))
export const invalidateSafeguard = (id: number, reason: string) => api<Safeguard>(`/safeguards/${id}/invalidate`, json('POST', { reason }))
export const restoreSafeguard = (id: number, reason: string) => api<Safeguard>(`/safeguards/${id}/restore`, json('POST', { reason }))
