import { api, query } from './client'
import { normalizePage, type PageData } from '../types/common'
import type { AuditFilters, AuditLog } from '../types/audit'

export async function listAuditLogs(filters: AuditFilters = {}): Promise<PageData<AuditLog>> {
  return normalizePage(await api<PageData<AuditLog> | AuditLog[]>(`/audit-logs${query({ entity_type: filters.entity_type, actor_id: filters.actor_id, request_id: filters.request_id, from: filters.from, to: filters.to })}`))
}
