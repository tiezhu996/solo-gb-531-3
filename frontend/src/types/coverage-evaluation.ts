import type { CoverageState } from './enums/coverage-state'

export interface PathEvidence {
  path_id?: string
  node_code?: string
  cause: string
  consequence: string
  safeguard_ids?: number[]
  safeguard_names?: string[]
  independence_keys?: string[]
  combined_protection?: number
  covered?: boolean
  protected?: boolean
  path_score?: number
  reason?: string
}

export interface ScoringStep {
  step?: number
  rule?: string
  input?: string
  contribution?: number
  running_score?: number
  explanation?: string
  label?: string
  value?: number | string
  detail?: string
}

export interface DeduplicatedSafeguard {
  independence_key: string
  kept_id: number
  ignored_ids: number[]
  reason: string
}

export interface EvaluationExplanation {
  summary: string
  paths: PathEvidence[]
  score_steps: ScoringStep[]
  deduplicated_safeguards: DeduplicatedSafeguard[]
  boundary_note: string
  reference_time: string
}

export interface CoverageSnapshot {
  scenario_id?: number
  scenario_version?: number
  causes?: string[]
  consequences?: string[]
  safeguards?: unknown[]
  input_hash?: string
}

export interface CoverageEvaluation {
  id: number
  scenario_id: number
  algorithm_version: string
  input_snapshot: CoverageSnapshot | string
  coverage_score: number
  uncovered_paths: PathEvidence[] | string
  risk_rank_before: string
  risk_rank_after: string
  evaluation_state: CoverageState
  explanation: EvaluationExplanation | string
  evaluated_by: number
  evaluated_by_name?: string
  evaluated_at: string
  input_hash?: string
  deduplicated_safeguards?: DeduplicatedSafeguard[]
  duration_milliseconds?: number
  determinism_replay_passed?: boolean
}

export interface CoverageRunInput { scenario_id: number }
