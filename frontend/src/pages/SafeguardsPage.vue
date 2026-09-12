<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Ban, FileCheck2, Pencil, Plus, RefreshCw, RotateCcw, ShieldCheck } from 'lucide-vue-next'
import AppShell from '../components/common/AppShell.vue'
import PageHeader from '../components/common/PageHeader.vue'
import EvidenceDrawer from '../components/common/EvidenceDrawer.vue'
import { useAuth } from '../hooks/useAuth'
import { useSafeguardStore } from '../stores/safeguard'
import { useDeviationScenarioStore } from '../stores/deviation-scenario'
import { errorMessage } from '../api/client'
import type { Safeguard, SafeguardInput, SafeguardType } from '../types/safeguard'

const store = useSafeguardStore()
const scenarios = useDeviationScenarioStore()
const { canEdit, canReview } = useAuth()
const dialog = ref(false)
const drawer = ref(false)
const editingId = ref<number>()
const evidenceTarget = ref<Safeguard>()
const filterScenario = ref<number>()
const saving = ref(false)
const form = reactive<SafeguardInput>({ name: '', safeguard_type: 'interlock', target_scenario_id: 0, independence_key: '', effectiveness: 0.8, test_interval_days: 365, last_verified_at: new Date().toISOString(), evidence_note: '' })
const typeLabels: Record<SafeguardType, string> = { alarm: '报警与人员响应', interlock: '安全联锁', relief: '泄放保护', procedural: '管理程序', containment: '物理包容', detection: '检测与响应' }
const types = Object.keys(typeLabels) as SafeguardType[]
const visible = computed(() => filterScenario.value ? store.items.filter((x) => x.target_scenario_id === filterScenario.value) : store.items)
const duplicateKeys = computed(() => visible.value.reduce<Record<string, number>>((acc, item) => { acc[item.independence_key] = (acc[item.independence_key] ?? 0) + 1; return acc }, {}))
const scenarioLabel = (id: number) => { const item = scenarios.items.find((x) => x.id === id); return item ? `#${item.id} ${item.guideword.toUpperCase()} ${item.parameter}` : `#${id}` }
function expiry(item: Safeguard) { if (item.verification_expires_at) return item.verification_expires_at; if (!item.last_verified_at) return undefined; const date = new Date(item.last_verified_at); date.setUTCDate(date.getUTCDate() + item.test_interval_days); return date.toISOString() }
function expiryLabel(item: Safeguard) { const value = expiry(item); return value ? new Date(value).toLocaleDateString('zh-CN') : '尚未验证' }
function isExpired(item: Safeguard) { const value = expiry(item); return item.verification_expired ?? (!value || new Date(value).getTime() < Date.now()) }

async function refresh() { try { await Promise.all([store.load(), scenarios.load()]) } catch (error) { ElMessage.error(errorMessage(error)) } }
function resetForm() { Object.assign(form, { name: '', safeguard_type: 'interlock' as SafeguardType, target_scenario_id: scenarios.items[0]?.id ?? 0, independence_key: '', effectiveness: 0.8, test_interval_days: 365, last_verified_at: new Date().toISOString(), evidence_note: '' }) }
function openCreate() { editingId.value = undefined; resetForm(); dialog.value = true }
function openEdit(item: Safeguard) { editingId.value = item.id; Object.assign(form, { name: item.name, safeguard_type: item.safeguard_type, target_scenario_id: item.target_scenario_id, independence_key: item.independence_key, effectiveness: item.effectiveness, test_interval_days: item.test_interval_days, last_verified_at: item.last_verified_at, evidence_note: item.evidence_note }); dialog.value = true }
function showEvidence(item: Safeguard) { evidenceTarget.value = item; drawer.value = true }
async function save() { saving.value = true; try { editingId.value ? await store.update(editingId.value, form) : await store.create(form); dialog.value = false; ElMessage.success(editingId.value ? '保护层已更新' : '保护层已登记') } catch (error) { ElMessage.error(errorMessage(error)) } finally { saving.value = false } }
async function action(item: Safeguard, kind: 'verify' | 'invalidate' | 'restore') { try { if (kind === 'verify') await store.verify(item.id, item.evidence_note || '台账复核完成'); else if (kind === 'invalidate') await store.invalidate(item.id, '安全复核员标记失效'); else await store.restore(item.id, '证据复核后恢复'); ElMessage.success('保护层生命周期已更新') } catch (error) { ElMessage.error(errorMessage(error)) } }
onMounted(refresh)
</script>

<template>
  <AppShell>
    <PageHeader eyebrow="INDEPENDENT PROTECTION REGISTER" title="保护层台账" description="核对覆盖目标、独立性键、有效性与验证期限；同一路径的重复独立性键只计一次。">
      <el-button :loading="store.loading" @click="refresh"><RefreshCw :size="16" />刷新</el-button>
      <el-button v-if="canEdit" type="primary" @click="openCreate"><Plus :size="16" />登记保护层</el-button>
    </PageHeader>
    <section class="filter-bar"><div><ShieldCheck :size="16" /><el-select v-model="filterScenario" placeholder="全部偏差场景" clearable><el-option v-for="item in scenarios.items" :key="item.id" :label="scenarioLabel(item.id)" :value="item.id" /></el-select></div><span>{{ visible.length }} 项保护措施 · {{ Object.keys(duplicateKeys).length }} 个独立键</span></section>
    <section class="data-section">
      <el-table v-loading="store.loading" :data="visible" row-key="id" empty-text="暂无保护措施">
        <el-table-column label="保护措施" min-width="220"><template #default="{ row }"><div class="primary-cell"><strong>{{ row.name }}</strong><span>{{ typeLabels[row.safeguard_type as SafeguardType] || row.safeguard_type }}</span></div></template></el-table-column>
        <el-table-column label="覆盖目标" min-width="190"><template #default="{ row }">{{ scenarioLabel(row.target_scenario_id) }}</template></el-table-column>
        <el-table-column label="独立性键" min-width="150"><template #default="{ row }"><code class="independence-key">{{ row.independence_key }}</code><small v-if="duplicateKeys[row.independence_key] > 1" class="duplicate-note">同场景将去重</small></template></el-table-column>
        <el-table-column label="有效性" width="115"><template #default="{ row }"><span class="numeric">{{ Math.round(row.effectiveness * 100) }}%</span></template></el-table-column>
        <el-table-column label="验证有效期" min-width="170"><template #default="{ row }"><span class="date-cell" :class="{ expired: isExpired(row) }">{{ expiryLabel(row) }}</span><small class="cell-note">间隔 {{ row.test_interval_days }} 天</small></template></el-table-column>
        <el-table-column label="生命周期" width="110"><template #default="{ row }"><span class="state-label" :class="isExpired(row) ? 'expired' : row.lifecycle_state">{{ isExpired(row) ? '已过期' : row.lifecycle_state }}</span></template></el-table-column>
        <el-table-column label="操作" width="195" fixed="right"><template #default="{ row }"><el-tooltip content="查看证据"><el-button circle text aria-label="查看证据" @click="showEvidence(row)"><FileCheck2 :size="16" /></el-button></el-tooltip><el-tooltip v-if="canEdit" content="编辑"><el-button circle text aria-label="编辑" @click="openEdit(row)"><Pencil :size="16" /></el-button></el-tooltip><el-tooltip v-if="canReview" content="记录本次验证"><el-button circle text type="success" aria-label="验证保护层" @click="action(row, 'verify')"><ShieldCheck :size="16" /></el-button></el-tooltip><el-tooltip v-if="canReview && ['pending','active','expired'].includes(row.lifecycle_state)" content="标记失效"><el-button circle text type="danger" aria-label="标记失效" @click="action(row, 'invalidate')"><Ban :size="16" /></el-button></el-tooltip><el-tooltip v-if="canReview && row.lifecycle_state === 'invalid'" content="恢复到待验证状态"><el-button circle text type="success" aria-label="恢复" @click="action(row, 'restore')"><RotateCcw :size="16" /></el-button></el-tooltip></template></el-table-column>
      </el-table>
    </section>
    <el-dialog v-model="dialog" :title="editingId ? '编辑保护层' : '登记独立保护层'" width="min(700px, 94vw)">
      <el-form label-position="top" @submit.prevent="save">
        <div class="form-grid two"><el-form-item label="措施名称" required><el-input v-model="form.name" /></el-form-item><el-form-item label="保护类型" required><el-select v-model="form.safeguard_type"><el-option v-for="type in types" :key="type" :label="typeLabels[type]" :value="type" /></el-select></el-form-item></div>
        <el-form-item label="目标偏差场景" required><el-select v-model="form.target_scenario_id" :disabled="Boolean(editingId)"><el-option v-for="item in scenarios.items" :key="item.id" :label="scenarioLabel(item.id)" :value="item.id" /></el-select></el-form-item>
        <div class="form-grid two"><el-form-item label="独立性键" required><el-input v-model="form.independence_key" placeholder="例如 SIS-101-TRIP" /></el-form-item><el-form-item label="有效性 (0-1)" required><el-input-number v-model="form.effectiveness" :min="0" :max="1" :step="0.05" :precision="2" /></el-form-item></div>
        <div class="form-grid two"><el-form-item label="试验间隔 (天)" required><el-input-number v-model="form.test_interval_days" :min="1" :max="3650" /></el-form-item><el-form-item label="最近验证时间" required><el-date-picker v-model="form.last_verified_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" :disabled="Boolean(editingId)" /></el-form-item></div>
        <el-form-item label="证据说明" required><el-input v-model="form.evidence_note" type="textarea" :rows="3" placeholder="记录试验报告、责任人和证据位置" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="dialog = false">取消</el-button><el-button type="primary" :loading="saving" @click="save">保存保护层</el-button></template>
    </el-dialog>
    <EvidenceDrawer v-model="drawer" :title="evidenceTarget?.name" :note="evidenceTarget?.evidence_note" :evidence="evidenceTarget" />
  </AppShell>
</template>
