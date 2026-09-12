<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Pencil, Plus, Power, RefreshCw, Search } from 'lucide-vue-next'
import AppShell from '../components/common/AppShell.vue'
import PageHeader from '../components/common/PageHeader.vue'
import RiskBadge from '../components/common/RiskBadge.vue'
import { useAuth } from '../hooks/useAuth'
import { useProcessNodeStore } from '../stores/process-node'
import { useDeviationScenarioStore } from '../stores/deviation-scenario'
import { errorMessage } from '../api/client'
import type { ProcessNode, ProcessNodeInput } from '../types/process-node'

const nodes = useProcessNodeStore()
const scenarios = useDeviationScenarioStore()
const { canEdit } = useAuth()
const search = ref('')
const dialog = ref(false)
const editingId = ref<number>()
const saving = ref(false)
const form = reactive<ProcessNodeInput>({ node_code: '', name: '', unit_name: '', medium: '', design_pressure: 0, design_temperature: 20, owner_team: '', status: 'active' })
const scenarioCounts = computed(() => scenarios.items.reduce<Record<number, number>>((acc, item) => { acc[item.process_node_id] = (acc[item.process_node_id] ?? 0) + 1; return acc }, {}))
function nodeRisk(id: number) {
  const score = scenarios.items.filter((item) => item.process_node_id === id).reduce((max, item) => Math.max(max, item.risk_score ?? item.likelihood * item.severity), 0)
  return score >= 20 ? 'critical' : score >= 12 ? 'high' : score >= 6 ? 'medium' : 'low'
}

async function refresh() { try { await Promise.all([nodes.load(search.value), scenarios.load()]) } catch (error) { ElMessage.error(errorMessage(error)) } }
function openCreate() { editingId.value = undefined; Object.assign(form, { node_code: '', name: '', unit_name: '', medium: '', design_pressure: 0, design_temperature: 20, owner_team: '', status: 'active' }); dialog.value = true }
function openEdit(node: ProcessNode) { editingId.value = node.id; Object.assign(form, { node_code: node.node_code, name: node.name, unit_name: node.unit_name, medium: node.medium, design_pressure: node.design_pressure, design_temperature: node.design_temperature, owner_team: node.owner_team, status: node.status }); dialog.value = true }
async function save() {
  saving.value = true
  try { editingId.value ? await nodes.update(editingId.value, form) : await nodes.create(form); dialog.value = false; ElMessage.success(editingId.value ? '节点边界已更新' : '工艺节点已建档') }
  catch (error) { ElMessage.error(errorMessage(error)) } finally { saving.value = false }
}
async function deactivate(node: ProcessNode) {
  try { await ElMessageBox.confirm(`停用 ${node.node_code} 后将不能新增偏差场景。`, '确认停用', { type: 'warning' }); await nodes.deactivate(node.id); ElMessage.success('节点已停用') }
  catch (error) { if (error !== 'cancel' && error !== 'close') ElMessage.error(errorMessage(error)) }
}
onMounted(refresh)
</script>

<template>
  <AppShell>
    <PageHeader eyebrow="PROCESS BOUNDARY REGISTER" title="工艺节点总览" description="维护分析边界并快速核对节点下偏差数量与当前风险；压力与温度均保留工程单位。">
      <el-button :loading="nodes.loading" @click="refresh"><RefreshCw :size="16" />刷新</el-button>
      <el-button v-if="canEdit" type="primary" @click="openCreate"><Plus :size="16" />新建节点</el-button>
    </PageHeader>
    <section class="filter-bar">
      <div><Search :size="16" /><el-input v-model="search" placeholder="节点编号、名称或介质" clearable @keyup.enter="refresh" /></div>
      <span>{{ nodes.items.length }} 个节点 · {{ scenarios.items.length }} 个偏差场景</span>
    </section>
    <section class="data-section">
      <el-table v-loading="nodes.loading" :data="nodes.items" row-key="id" empty-text="暂无工艺节点">
        <el-table-column label="节点" min-width="210">
          <template #default="{ row }"><div class="primary-cell"><strong>{{ row.node_code }}</strong><span>{{ row.name }}</span></div></template>
        </el-table-column>
        <el-table-column prop="unit_name" label="装置" min-width="140" />
        <el-table-column prop="medium" label="介质" min-width="120" />
        <el-table-column label="设计边界" min-width="180"><template #default="{ row }"><span class="numeric">{{ row.design_pressure.toFixed(2) }} MPa</span><small class="cell-note">{{ row.design_temperature.toFixed(1) }} °C</small></template></el-table-column>
        <el-table-column label="场景 / 风险" min-width="150"><template #default="{ row }"><span class="count-mark">{{ scenarioCounts[row.id] ?? row.coverage_summary?.scenario_count ?? 0 }} 条</span><RiskBadge :rank="nodeRisk(row.id)" /></template></el-table-column>
        <el-table-column prop="owner_team" label="责任团队" min-width="140" />
        <el-table-column label="状态" width="95"><template #default="{ row }"><span class="state-label" :class="row.status">{{ row.status === 'active' ? '在用' : '停用' }}</span></template></el-table-column>
        <el-table-column v-if="canEdit" label="操作" width="105" fixed="right">
          <template #default="{ row }"><el-tooltip content="编辑设计边界"><el-button circle text aria-label="编辑" @click="openEdit(row)"><Pencil :size="16" /></el-button></el-tooltip><el-tooltip content="停用节点"><el-button circle text type="danger" aria-label="停用" :disabled="row.status !== 'active'" @click="deactivate(row)"><Power :size="16" /></el-button></el-tooltip></template>
        </el-table-column>
      </el-table>
    </section>
    <el-dialog v-model="dialog" :title="editingId ? '修改节点设计边界' : '新建工艺节点'" width="min(650px, 94vw)">
      <el-form label-position="top" @submit.prevent="save">
        <div class="form-grid two"><el-form-item label="节点编号" required><el-input v-model="form.node_code" /></el-form-item><el-form-item label="节点名称" required><el-input v-model="form.name" /></el-form-item></div>
        <div class="form-grid two"><el-form-item label="所属装置" required><el-input v-model="form.unit_name" /></el-form-item><el-form-item label="介质" required><el-input v-model="form.medium" /></el-form-item></div>
        <div class="form-grid two"><el-form-item label="设计压力 (MPa)" required><el-input-number v-model="form.design_pressure" :precision="2" :min="0" :max="100" /></el-form-item><el-form-item label="设计温度 (°C)" required><el-input-number v-model="form.design_temperature" :precision="1" :min="-273" :max="1200" /></el-form-item></div>
        <el-form-item label="责任团队" required><el-input v-model="form.owner_team" /></el-form-item>
        <button type="submit" class="sr-only">提交</button>
      </el-form>
      <template #footer><el-button @click="dialog = false">取消</el-button><el-button type="primary" :loading="saving" @click="save">{{ editingId ? '保存边界' : '创建节点' }}</el-button></template>
    </el-dialog>
  </AppShell>
</template>
