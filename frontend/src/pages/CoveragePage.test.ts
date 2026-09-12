import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ElAlert, ElButton } from 'element-plus'
import type { CoverageEvaluation, CoverageEvidencePack } from '../types/coverage-evaluation'
import CoveragePage from './CoveragePage.vue'

function makeEvaluation(overrides: Partial<CoverageEvaluation>): CoverageEvaluation {
  return {
    id: 31,
    scenario_id: 7,
    algorithm_version: 'hazop-cover-v1.0.0',
    input_snapshot: { scenario_id: 7, causes: ['cooling loss'], consequences: ['overpressure'] },
    coverage_score: 90,
    uncovered_paths: [],
    risk_rank_before: 'critical',
    risk_rank_after: 'medium',
    evaluation_state: 'completed',
    explanation: {
      summary: 'ok',
      paths: [],
      score_steps: [{ step: 1, rule: 'independent-layer-combination', contribution: 0.9, running_score: 90, explanation: 'combined' }],
      deduplicated_safeguards: [{ independence_key: 'SIS-A', kept_id: 2, ignored_ids: [3], reason: 'same logic solver' }],
      boundary_note: 'offline only',
      reference_time: '2026-09-01T00:00:00Z',
    },
    evaluated_by: 9,
    evaluated_by_name: 'engineer',
    evaluated_at: '2026-09-01T00:00:00Z',
    input_hash: 'hash-31',
    duration_milliseconds: 12,
    ...overrides,
  }
}

const completedEvaluation = makeEvaluation({})
const failedEvaluation = makeEvaluation({
  id: 42,
  evaluation_state: 'failed',
  coverage_score: 0,
  failure_reason: 'snapshot requires persisted node and scenario',
  explanation: { summary: '', paths: [], score_steps: [], deduplicated_safeguards: [], boundary_note: '', reference_time: '' },
})
const voidedEvaluation = makeEvaluation({ id: 53, evaluation_state: 'voided', input_hash: 'hash-53' })
const queuedEvaluation = makeEvaluation({
  id: 64,
  evaluation_state: 'queued',
  coverage_score: 0,
  explanation: { summary: '', paths: [], score_steps: [], deduplicated_safeguards: [], boundary_note: '', reference_time: '' },
})

function envelope<T>(data: T) {
  return { code: 'OK', message: 'success', data, request_id: 'req-test' }
}

type FetchMock = ReturnType<typeof vi.fn>

function mountPageWith(evaluation: CoverageEvaluation, exportResponse: (id: number, path: string) => Response): { wrapper: VueWrapper; fetchMock: FetchMock } {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/coverage', component: { template: '<div />' } }, { path: '/login', component: { template: '<div />' } }],
  })
  router.push('/coverage')
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const path = String(input)
    if (path.includes('/evidence-pack')) {
      const match = path.match(/coverage-evaluations\/(\d+)\/evidence-pack/)
      const id = match ? Number(match[1]) : 0
      return exportResponse(id, path)
    }
    if (path.includes('/coverage-evaluations')) return new Response(JSON.stringify(envelope({ items: [evaluation], total: 1, page: 1, page_size: 20 })), { status: 200 })
    if (path.includes('/deviation-scenarios')) return new Response(JSON.stringify(envelope({ items: [{ id: 7, process_node_id: 1, guideword: 'more', parameter: 'temperature', cause: 'cooling loss', consequence: 'overpressure', likelihood: 4, severity: 5, scenario_state: 'analyzed', version: 1 }], total: 1 })), { status: 200 })
    if (path.includes('/safeguards')) return new Response(JSON.stringify(envelope({ items: [], total: 0 })), { status: 200 })
    return new Response(JSON.stringify(envelope({})), { status: 200 })
  })
  vi.stubGlobal('fetch', fetchMock)
  const wrapper = mount(CoveragePage, {
    global: {
      plugins: [router, ElButton, ElAlert],
      stubs: {
        'el-select': { template: '<div class="el-select-stub"><slot /><slot name="append" /></div>' },
        'el-option': true,
        'el-tooltip': { template: '<span><slot /></span>' },
        'el-drawer': true,
        'router-link': { template: '<a><slot /></a>' },
      },
    },
  })
  return { wrapper, fetchMock }
}

function successfulPack(evaluation: CoverageEvaluation): CoverageEvidencePack {
  return {
    pack_version: 'evidence-pack-v1',
    exported_at: '2026-09-12T00:00:00Z',
    evaluation_id: evaluation.id,
    scenario_id: 7,
    algorithm_version: 'hazop-cover-v1.0.0',
    idempotency_key: `key-${evaluation.id}`,
    input_hash: evaluation.input_hash ?? '',
    input_snapshot: evaluation.input_snapshot as CoverageEvidencePack['input_snapshot'],
    coverage_score: evaluation.coverage_score,
    score_steps: typeof evaluation.explanation === 'object' ? evaluation.explanation.score_steps ?? [] : [],
    uncovered_paths: [],
    deduplicated_safeguards: [],
    risk_rank_before: 'critical',
    risk_rank_after: 'medium',
    state: { code: evaluation.evaluation_state, label: evaluation.evaluation_state, readable_summary: '状态说明' },
    evaluated_by: 9,
    evaluated_by_name: 'engineer',
    evaluated_at: '2026-09-01T00:00:00Z',
    duration_milliseconds: 12,
    boundary_note: 'offline only',
  }
}

describe('CoveragePage evidence pack export', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.setItem('hazop_coverage_token', 'token')
    localStorage.setItem('hazop_coverage_user', JSON.stringify({ id: 9, username: 'engineer', role: 'process_engineer', display_name: 'Engineer' }))
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it('downloads a completed evidence pack without changing existing evaluation views', async () => {
    const createObjectURL = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:evidence')
    const revokeObjectURL = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    const { wrapper, fetchMock } = mountPageWith(completedEvaluation, () => new Response(JSON.stringify(envelope(successfulPack(completedEvaluation))), { status: 200 }))
    await flushPromises()

    const exportButtons = wrapper.findAll('button[aria-label="导出证据包"]')
    expect(exportButtons.length).toBeGreaterThan(0)
    await exportButtons[0].trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('/coverage-evaluations/31/evidence-pack'), expect.anything())
    expect(createObjectURL).toHaveBeenCalledTimes(1)
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:evidence')
    expect(click).toHaveBeenCalledTimes(1)
    expect(document.body.textContent).toContain('证据包已导出')
    // 既有评分、路径和状态视图保持不变
    expect(wrapper.text()).toContain('/ 100')
    expect(wrapper.text()).toContain('待确认')
  })

  it('shows the failed-state explanation and an explicit export error without downloading', async () => {
    const createObjectURL = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:should-not-exist')
    const { wrapper } = mountPageWith(failedEvaluation, () => new Response(JSON.stringify({ code: 'VALIDATION_FAILED', message: 'evidence pack rejected: frozen input snapshot is not valid immutable JSON', request_id: 'req-fail' }), { status: 422 }))
    await flushPromises()

    expect(wrapper.text()).toContain('评估计算失败')
    expect(wrapper.text()).toContain('snapshot requires persisted node and scenario')

    await wrapper.find('button[aria-label="导出证据包"]').trigger('click')
    await flushPromises()

    expect(URL.createObjectURL).not.toHaveBeenCalled()
    expect(document.body.textContent).toContain('证据包导出失败')
    expect(document.body.textContent).toContain('frozen input snapshot')
  })

  it('keeps the voided evaluation readable and still exports its historical evidence', async () => {
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:voided')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    const { wrapper } = mountPageWith(voidedEvaluation, () => new Response(JSON.stringify(envelope(successfulPack(voidedEvaluation))), { status: 200 }))
    await flushPromises()

    expect(wrapper.text()).toContain('评估已作废')
    expect(wrapper.text()).toContain('不再作为保护层覆盖依据')

    await wrapper.find('button[aria-label="导出证据包"]').trigger('click')
    await flushPromises()

    expect(click).toHaveBeenCalledTimes(1)
    expect(document.body.textContent).toContain('包含当前状态的可读说明')
  })

  it('explains a queued evaluation and reports a missing export target as 404 without downloading', async () => {
    const createObjectURL = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:should-not-exist')
    const { wrapper } = mountPageWith(queuedEvaluation, () => new Response(JSON.stringify({ code: 'NOT_FOUND', message: 'coverage evaluation was not found', request_id: 'req-missing' }), { status: 404 }))
    await flushPromises()

    expect(wrapper.text()).toContain('等待计算')

    await wrapper.find('button[aria-label="导出证据包"]').trigger('click')
    await flushPromises()

    expect(URL.createObjectURL).not.toHaveBeenCalled()
    expect(document.body.textContent).toContain('证据包导出失败')
    expect(document.body.textContent).toContain('coverage evaluation was not found')
  })
})
