<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { FileCheck2, RefreshCw, Search } from 'lucide-vue-next'
import AppShell from '../components/common/AppShell.vue'
import PageHeader from '../components/common/PageHeader.vue'
import EvidenceDrawer from '../components/common/EvidenceDrawer.vue'
import { useAuditStore } from '../stores/audit'
import { errorMessage } from '../api/client'
import type { AuditFilters, AuditLog } from '../types/audit'

const store = useAuditStore()
const drawer = ref(false)
const selected = ref<AuditLog>()
const filters = reactive<AuditFilters>({ entity_type: '', actor_id: '', request_id: '', from: '', to: '' })
const counts = computed(() => store.items.reduce<Record<string, number>>((acc, item) => { acc[item.entity_type] = (acc[item.entity_type] ?? 0) + 1; return acc }, {}))
const entityLabels: Record<string, string> = { process_node: '工艺节点', deviation_scenario: '偏差场景', safeguard: '保护层', coverage_evaluation: '覆盖评估' }
async function refresh() { try { await store.load(filters) } catch (error) { ElMessage.error(errorMessage(error)) } }
function inspect(item: AuditLog) { selected.value = item; drawer.value = true }
function snapshot(value: unknown) { if (typeof value !== 'string') return value ?? {}; try { return JSON.parse(value) } catch { return value } }
onMounted(refresh)
</script>

<template>
  <AppShell>
    <PageHeader eyebrow="IMMUTABLE CHANGE JOURNAL" title="审计中心" description="按实体、操作者、request ID 与时间追溯四类核心实体的写操作和算法运行证据。">
      <el-button :loading="store.loading" @click="refresh"><RefreshCw :size="16" />刷新</el-button>
    </PageHeader>
    <section class="audit-metrics"><div v-for="entity in ['process_node','deviation_scenario','safeguard','coverage_evaluation']" :key="entity"><span>{{ entityLabels[entity] }}</span><strong>{{ counts[entity] ?? 0 }}</strong></div></section>
    <section class="audit-tools">
      <div class="audit-search"><Search :size="16" /><el-input v-model="filters.request_id" placeholder="request ID" clearable /><el-input v-model="filters.actor_id" placeholder="操作者 ID" clearable /></div>
      <div class="audit-search"><el-select v-model="filters.entity_type" placeholder="全部实体" clearable><el-option v-for="(label, value) in entityLabels" :key="value" :label="label" :value="value" /></el-select><el-date-picker v-model="filters.from" type="datetime" placeholder="起始时间" value-format="YYYY-MM-DDTHH:mm:ssZ" /><el-date-picker v-model="filters.to" type="datetime" placeholder="结束时间" value-format="YYYY-MM-DDTHH:mm:ssZ" /><el-button type="primary" @click="refresh">筛选</el-button></div>
    </section>
    <section class="data-section audit-events">
      <el-table v-loading="store.loading" :data="store.items" row-key="id" empty-text="暂无审计记录">
        <el-table-column label="时间" min-width="170"><template #default="{ row }">{{ new Date(row.created_at).toLocaleString('zh-CN') }}</template></el-table-column>
        <el-table-column label="操作者" min-width="150"><template #default="{ row }"><div class="primary-cell"><strong>{{ row.actor_name || `用户 #${row.actor_id}` }}</strong><span>{{ row.actor_role }}</span></div></template></el-table-column>
        <el-table-column label="实体" min-width="145"><template #default="{ row }">{{ entityLabels[row.entity_type] || row.entity_type }} #{{ row.entity_id }}</template></el-table-column>
        <el-table-column prop="action" label="动作" min-width="135"><template #default="{ row }"><code class="audit-action">{{ row.action }}</code></template></el-table-column>
        <el-table-column label="Request ID" min-width="230"><template #default="{ row }"><code>{{ row.request_id }}</code></template></el-table-column>
        <el-table-column label="证据" width="80" fixed="right"><template #default="{ row }"><el-tooltip content="查看变更前后快照"><el-button circle text aria-label="查看证据" @click="inspect(row)"><FileCheck2 :size="17" /></el-button></el-tooltip></template></el-table-column>
      </el-table>
    </section>
    <EvidenceDrawer v-model="drawer" :title="selected ? `${entityLabels[selected.entity_type] || selected.entity_type} #${selected.entity_id}` : '审计证据'" :note="selected ? `${selected.action} · ${selected.request_id}` : ''">
      <template v-if="selected"><div class="snapshot-block"><strong>变更前</strong><pre>{{ JSON.stringify(snapshot(selected.before_snapshot), null, 2) }}</pre></div><div class="snapshot-block"><strong>变更后</strong><pre>{{ JSON.stringify(snapshot(selected.after_snapshot), null, 2) }}</pre></div><div v-if="selected.input_hash || selected.result_summary" class="snapshot-block"><strong>运行元数据</strong><pre>{{ JSON.stringify({ input_hash: selected.input_hash, algorithm_version: selected.algorithm_version, duration_ms: selected.duration_ms, result_summary: snapshot(selected.result_summary) }, null, 2) }}</pre></div></template>
    </EvidenceDrawer>
  </AppShell>
</template>
