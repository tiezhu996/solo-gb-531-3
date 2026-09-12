import { api, json, query } from './client'
import { normalizePage, type PageData } from '../types/common'
import type { DeviationScenario, DeviationScenarioInput, ScenarioState } from '../types/deviation-scenario'

export interface ScenarioFilters { process_node_id?: number; state?: string }

export async function listDeviationScenarios(filters: ScenarioFilters = {}): Promise<PageData<DeviationScenario>> {
  return normalizePage(await api<PageData<DeviationScenario> | DeviationScenario[]>(`/deviation-scenarios${query({ process_node_id: filters.process_node_id, state: filters.state })}`))
}
export const getDeviationScenario = (id: number) => api<DeviationScenario>(`/deviation-scenarios/${id}`)
export const createDeviationScenario = (input: DeviationScenarioInput) => api<DeviationScenario>('/deviation-scenarios', json('POST', input))
export const updateDeviationScenario = (id: number, input: DeviationScenarioInput, version: number) => api<DeviationScenario>(`/deviation-scenarios/${id}`, json('PUT', { ...input, version }))
export const transitionDeviationScenario = (id: number, to_state: ScenarioState, version: number, comment: string) => api<DeviationScenario>(`/deviation-scenarios/${id}/transition`, json('POST', { to_state, version, comment }))
