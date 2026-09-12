import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as safeguardApi from '../api/safeguard'
import type { Safeguard, SafeguardInput } from '../types/safeguard'

export const useSafeguardStore = defineStore('safeguards', () => {
  const items = ref<Safeguard[]>([])
  const selectedId = ref<number>()
  const loading = ref(false)
  const selected = computed(() => items.value.find((item) => item.id === selectedId.value))
  async function load(scenarioId?: number) { loading.value = true; try { items.value = (await safeguardApi.listSafeguards(scenarioId)).items; if (!items.value.some((x) => x.id === selectedId.value)) selectedId.value = items.value[0]?.id } finally { loading.value = false } }
  async function create(input: SafeguardInput) { const item = await safeguardApi.createSafeguard(input); await load(); selectedId.value = item.id; return item }
  async function update(id: number, input: SafeguardInput) { const item = await safeguardApi.updateSafeguard(id, input); await load(); return item }
  async function verify(id: number, note: string) { const item = await safeguardApi.verifySafeguard(id, note); await load(); return item }
  async function invalidate(id: number, reason: string) { const item = await safeguardApi.invalidateSafeguard(id, reason); await load(); return item }
  async function restore(id: number, reason: string) { const item = await safeguardApi.restoreSafeguard(id, reason); await load(); return item }
  return { items, selectedId, selected, loading, load, create, update, verify, invalidate, restore }
})
