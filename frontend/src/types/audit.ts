export interface AuditLog {
  id: number
  actor_id: number
  actor_name?: string
  actor_role?: string
  request_id: string
  entity_type: string
  entity_id: number
  action: string
  before_snapshot?: unknown
  after_snapshot?: unknown
  input_hash?: string
  algorithm_version?: string
  duration_ms?: number
  result_summary?: unknown
  created_at: string
}

export interface AuditFilters {
  entity_type?: string
  actor_id?: number | string
  request_id?: string
  from?: string
  to?: string
}
