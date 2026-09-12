import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as scenarioApi from '../api/deviation-scenario'
import type { DeviationScenario, DeviationScenarioInput, ScenarioState } from '../types/deviation-scenario'

export const useDeviationScenarioStore = defineStore('deviation-scenarios', () => {
  const items = ref<DeviationScenario[]>([])
  const selectedId = ref<number>()
  const loading = ref(false)
  const selected = computed(() => items.value.find((item) => item.id === selectedId.value))
  async function load(processNodeId?: number) { loading.value = true; try { items.value = (await scenarioApi.listDeviationScenarios({ process_node_id: processNodeId })).items; if (!items.value.some((x) => x.id === selectedId.value)) selectedId.value = items.value[0]?.id } finally { loading.value = false } }
  async function create(input: DeviationScenarioInput) { const item = await scenarioApi.createDeviationScenario(input); await load(); selectedId.value = item.id; return item }
  async function update(id: number, input: DeviationScenarioInput) {
    const current = items.value.find((item) => item.id === id)
    if (!current) throw new Error('场景版本不存在，请刷新后重试。')
    const item = await scenarioApi.updateDeviationScenario(id, input, current.version)
    await load(); return item
  }
  async function transition(id: number, target: ScenarioState, comment: string) {
    const current = items.value.find((item) => item.id === id)
    if (!current) throw new Error('场景版本不存在，请刷新后重试。')
    const item = await scenarioApi.transitionDeviationScenario(id, target, current.version, comment)
    await load(); selectedId.value = item.id; return item
  }
  return { items, selectedId, selected, loading, load, create, update, transition }
})
