import { onUnmounted, ref } from 'vue'
import { useCoverageEvaluationStore } from '../stores/coverage-evaluation'
import type { CoverageEvaluation } from '../types/coverage-evaluation'

const terminal = new Set(['completed', 'failed', 'confirmed', 'voided'])

export function useCoverageRun() {
  const store = useCoverageEvaluationStore()
  const running = ref(false)
  const polling = ref(false)
  let generation = 0
  let timer: ReturnType<typeof setTimeout> | undefined

  function stop() {
    generation += 1
    polling.value = false
    if (timer) clearTimeout(timer)
    timer = undefined
  }

  async function watch(run: CoverageEvaluation) {
    const activeGeneration = ++generation
    polling.value = !terminal.has(run.evaluation_state)
    let current = run
    while (!terminal.has(current.evaluation_state) && generation === activeGeneration) {
      await new Promise<void>((resolve) => { timer = setTimeout(resolve, 700) })
      if (generation !== activeGeneration) break
      current = await store.refresh(run.id)
    }
    polling.value = false
    return current
  }

  async function launch(scenarioId: number) {
    stop()
    running.value = true
    try {
      const key = `coverage-${scenarioId}-${crypto.randomUUID()}`
      return await watch(await store.run(scenarioId, key))
    } finally { running.value = false }
  }

  onUnmounted(stop)
  return { running, polling, launch, stop }
}
