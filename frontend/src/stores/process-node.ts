import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as nodeApi from '../api/process-node'
import type { ProcessNode, ProcessNodeInput } from '../types/process-node'

export const useProcessNodeStore = defineStore('process-nodes', () => {
  const items = ref<ProcessNode[]>([])
  const selectedId = ref<number>()
  const loading = ref(false)
  const selected = computed(() => items.value.find((item) => item.id === selectedId.value))
  async function load(search = '') { loading.value = true; try { items.value = (await nodeApi.listProcessNodes(search)).items; if (!selectedId.value) selectedId.value = items.value[0]?.id } finally { loading.value = false } }
  async function create(input: ProcessNodeInput) { const item = await nodeApi.createProcessNode(input); await load(); selectedId.value = item.id; return item }
  async function update(id: number, input: ProcessNodeInput) { const item = await nodeApi.updateProcessNode(id, input); await load(); return item }
  async function deactivate(id: number) { const item = await nodeApi.deactivateProcessNode(id); await load(); return item }
  return { items, selectedId, selected, loading, load, create, update, deactivate }
})
