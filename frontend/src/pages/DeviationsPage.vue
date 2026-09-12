<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowRight, Pencil, Plus, RefreshCw, RotateCcw, ShieldCheck } from 'lucide-vue-next'
import AppShell from '../components/common/AppShell.vue'
import PageHeader from '../components/common/PageHeader.vue'
import RiskBadge from '../components/common/RiskBadge.vue'
import ScenarioStateTimeline from '../components/common/ScenarioStateTimeline.vue'
import { useAuth } from '../hooks/useAuth'
import { useDeviationScenarioStore } from '../stores/deviation-scenario'
import { useProcessNodeStore } from '../stores/process-node'
import { useSafeguardStore } from '../stores/safeguard'
import { errorMessage } from '../api/client'
import { deviationGuidewords, deviationGuidewordLabels, type DeviationGuideword } from '../types/enums/deviation-guideword'
import type { DeviationScenario, DeviationScenarioInput, ScenarioState } from '../types/deviation-scenario'

const scenarios = useDeviationScenarioStore()
const nodes = useProcessNodeStore()
const safeguards = useSafeguardStore()
const { canEdit, canReview } = useAuth()
const dialog = ref(false)
const editingId = ref<number>()
const saving = ref(false)
const filterState = ref('')
const transitionComment = ref('')
const form = reactive<DeviationScenarioInput>({ process_node_id: 0, guideword: 'no', parameter: '', cause: '', consequence: '', likelihood: 1, severity: 1 })
const visible = computed(() => filterState.value ? scenarios.items.filter((x) => x.scenario_state === filterState.value) : scenarios.items)
const selected = computed(() => scenarios.selected)
const selectedSafeguards = computed(() => safeguards.items.filter((x) => x.target_scenario_id === selected.value?.id))
const nodeName = (id: number) => nodes.items.find((x) => x.id === id)?.node_code ?? `#${id}`

async function refresh() { try { await Promise.all([scenarios.load(), nodes.load(), safeguards.load()]) } catch (error) { ElMessage.error(errorMessage(error)) } }
function openCreate() { editingId.value = undefined; Object.assign(form, { process_node_id: nodes.items[0]?.id ?? 0, guideword: 'no' as DeviationGuideword, parameter: '', cause: '', consequence: '', likelihood: 1, severity: 1 }); dialog.value = true }
function openEdit(item: DeviationScenario) { editingId.value = item.id; Object.assign(form, { process_node_id: item.process_node_id, guideword: item.guideword, parameter: item.parameter, cause: item.cause, consequence: item.consequence, likelihood: item.likelihood, severity: item.severity }); dialog.value = true }
async function save() { saving.value = true; try { editingId.value ? await scenarios.update(editingId.value, form) : await scenarios.create(form); dialog.value = false; ElMessage.success(editingId.value ? '偏差内容已更新' : '偏差场景已创建') } catch (error) { ElMessage.error(errorMessage(error)) } finally { saving.value = false } }
async function transition(target: ScenarioState) { if (!selected.value) return; try { await scenarios.transition(selected.value.id, target, transitionComment.value || `迁移至 ${stateLabel(target)}`); transitionComment.value = ''; ElMessage.success(`场景已进入“${stateLabel(target)}”`) } catch (error) { ElMessage.error(errorMessage(error)) } }
function stateLabel(state: ScenarioState) { return ({ draft: '草稿', analyzed: '已分析', verified: '已复核', accepted: '已接受', rework: '退回修订' })[state] }
const availableTransitions = computed(() => {
  const state = selected.value?.scenario_state
  if (!state) return [] as ScenarioState[]
  if (state === 'draft' || state === 'rework') return canEdit.value ? ['analyzed'] as ScenarioState[] : []
  if (state === 'analyzed') return canReview.value ? ['verified', 'rework'] as ScenarioState[] : []
  if (state === 'verified') return canReview.value ? ['accepted', 'rework'] as ScenarioState[] : []
  return [] as ScenarioState[]
})
onMounted(refresh)
</script>

<template>
  <AppShell>
    <PageHeader eyebrow="DEVIATION WORKSHEET" title="偏差分析台" description="沿状态流编制原因与后果，由不同人员完成复核；退回与非法迁移均保留明确证据。">
      <el-button :loading="scenarios.loading" @click="refresh"><RefreshCw :size="16" />刷新</el-button>
      <el-button v-if="canEdit" type="primary" @click="openCreate"><Plus :size="16" />新建偏差</el-button>
    </PageHeader>
    <div class="workspace-toolbar"><el-select v-model="filterState" placeholder="全部状态" clearable><el-option v-for="state in ['draft','analyzed','verified','accepted','rework']" :key="state" :label="stateLabel(state as ScenarioState)" :value="state" /></el-select><span>{{ visible.length }} 条场景</span></div>
    <div class="split-workspace deviation-workspace">
      <section class="data-section">
        <div class="section-heading"><h2>场景清单</h2><span>选择一条查看状态与保护层</span></div>
        <button v-for="item in visible" :key="item.id" class="scenario-row" :class="{ selected: scenarios.selectedId === item.id }" @click="scenarios.selectedId = item.id">
          <span class="scenario-index">{{ nodeName(item.process_node_id) }} · V{{ item.version }}</span>
          <span class="scenario-copy"><strong>{{ deviationGuidewordLabels[item.guideword] }} {{ item.parameter }}</strong><small>{{ item.cause }} → {{ item.consequence }}</small></span>
          <RiskBadge :likelihood="item.likelihood" :severity="item.severity" />
          <span class="state-label" :class="item.scenario_state">{{ stateLabel(item.scenario_state) }}</span>
        </button>
        <div v-if="!visible.length" class="empty-inline">暂无符合条件的偏差场景</div>
      </section>
      <section v-if="selected" class="editor-panel">
        <div class="detail-heading"><div><p class="eyebrow">SCENARIO #{{ selected.id }} · V{{ selected.version }}</p><h2>{{ deviationGuidewordLabels[selected.guideword] }} {{ selected.parameter }}</h2></div><el-tooltip v-if="canEdit && ['draft','rework'].includes(selected.scenario_state)" content="编辑场景"><el-button circle text aria-label="编辑场景" @click="openEdit(selected)"><Pencil :size="16" /></el-button></el-tooltip></div>
        <ScenarioStateTimeline :state="selected.scenario_state" />
        <dl class="evidence-pairs"><div><dt>工艺节点</dt><dd>{{ nodeName(selected.process_node_id) }}</dd></div><div><dt>风险矩阵</dt><dd><RiskBadge :likelihood="selected.likelihood" :severity="selected.severity" /> L{{ selected.likelihood }} × S{{ selected.severity }}</dd></div><div class="wide"><dt>原因</dt><dd>{{ selected.cause }}</dd></div><div class="wide"><dt>后果</dt><dd>{{ selected.consequence }}</dd></div></dl>
        <div class="related-strip"><div><ShieldCheck :size="17" /><strong>关联保护层</strong></div><span>{{ selectedSafeguards.length }} 项 · 独立键 {{ new Set(selectedSafeguards.map((x) => x.independence_key)).size }} 个</span></div>
        <div class="transition-desk"><p class="eyebrow">STATE TRANSITION</p><h2>评审动作</h2><template v-if="availableTransitions.length"><el-input v-model="transitionComment" type="textarea" :rows="2" placeholder="填写分析或复核意见" /><div class="transition-actions"><el-button v-for="target in availableTransitions" :key="target" :type="target === 'rework' ? 'warning' : 'primary'" @click="transition(target)"><RotateCcw v-if="target === 'rework'" :size="15" /><ArrowRight v-else :size="15" />{{ stateLabel(target) }}</el-button></div></template><p v-else class="read-only-note">当前身份在此状态下没有可执行迁移。复核人与场景作者必须不同。</p></div>
      </section>
      <section v-else class="editor-panel empty-state"><ShieldCheck :size="28" /><h2>选择偏差场景</h2><p>查看原因到后果路径、保护层和评审状态。</p></section>
    </div>
    <el-dialog v-model="dialog" :title="editingId ? '编辑偏差场景' : '新建偏差场景'" width="min(720px, 94vw)">
      <el-form label-position="top" @submit.prevent="save">
        <div class="form-grid two"><el-form-item label="工艺节点" required><el-select v-model="form.process_node_id"><el-option v-for="node in nodes.items.filter((x) => x.status === 'active')" :key="node.id" :label="`${node.node_code} · ${node.name}`" :value="node.id" /></el-select></el-form-item><el-form-item label="引导词" required><el-select v-model="form.guideword"><el-option v-for="word in deviationGuidewords" :key="word" :label="deviationGuidewordLabels[word]" :value="word" /></el-select></el-form-item></div>
        <el-form-item label="工艺参数" required><el-input v-model="form.parameter" placeholder="例如：流量、压力、温度" /></el-form-item>
        <div class="form-grid two"><el-form-item label="可能性 (1-5)" required><el-input-number v-model="form.likelihood" :min="1" :max="5" /></el-form-item><el-form-item label="严重度 (1-5)" required><el-input-number v-model="form.severity" :min="1" :max="5" /></el-form-item></div>
        <el-form-item label="原因" required><el-input v-model="form.cause" type="textarea" :rows="3" /></el-form-item><el-form-item label="后果" required><el-input v-model="form.consequence" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="dialog = false">取消</el-button><el-button type="primary" :loading="saving" @click="save">{{ editingId ? '保存新版本' : '创建场景' }}</el-button></template>
    </el-dialog>
  </AppShell>
</template>
