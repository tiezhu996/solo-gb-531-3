import type { DeviationGuideword } from './enums/deviation-guideword'
import type { ProcessNode } from './process-node'

export type ScenarioState = 'draft' | 'analyzed' | 'verified' | 'accepted' | 'rework'

export interface DeviationScenario {
  id: number
  process_node_id: number
  guideword: DeviationGuideword
  parameter: string
  cause: string
  consequence: string
  likelihood: number
  severity: number
  risk_score?: number
  risk_rank?: string
  scenario_state: ScenarioState
  version: number
  created_by: number
  created_by_name?: string
  reviewed_by?: number
  reviewed_by_name?: string
  process_node?: ProcessNode
  safeguard_count?: number
  updated_at: string
}

export interface DeviationScenarioInput {
  process_node_id: number
  guideword: DeviationGuideword
  parameter: string
  cause: string
  consequence: string
  likelihood: number
  severity: number
}

export const scenarioTransitions: Record<ScenarioState, ScenarioState[]> = {
  draft: ['analyzed'],
  analyzed: ['verified', 'rework'],
  verified: ['accepted', 'rework'],
  accepted: [],
  rework: ['analyzed'],
}
