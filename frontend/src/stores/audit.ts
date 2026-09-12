import { defineStore } from 'pinia'
import { ref } from 'vue'
import { listAuditLogs } from '../api/audit'
import type { AuditFilters, AuditLog } from '../types/audit'

export const useAuditStore = defineStore('audit', () => {
  const items = ref<AuditLog[]>([])
  const loading = ref(false)
  async function load(filters: AuditFilters = {}) { loading.value = true; try { items.value = (await listAuditLogs(filters)).items } finally { loading.value = false } }
  return { items, loading, load }
})
