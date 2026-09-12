export type NodeStatus = 'active' | 'inactive'

export interface ProcessNode {
  id: number
  node_code: string
  name: string
  unit_name: string
  medium: string
  design_pressure: number
  design_temperature: number
  owner_team: string
  status: NodeStatus
  coverage_summary?: {
    scenario_count: number
    active_safeguards: number
    open_high_risk: number
    latest_coverage: number
    uncovered_path_count: number
  }
  scenario_count?: number
  highest_risk?: string
  created_at: string
  updated_at: string
}

export interface ProcessNodeInput {
  node_code: string
  name: string
  unit_name: string
  medium: string
  design_pressure: number
  design_temperature: number
  owner_team: string
  status?: NodeStatus
}
