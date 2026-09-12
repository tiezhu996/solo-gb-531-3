<script setup lang="ts">
import { computed } from 'vue'
import { riskLevel } from '../../utils/risk'

const props = defineProps<{ rank?: string; likelihood?: number; severity?: number; score?: number }>()
const numeric = computed(() => props.score ?? ((props.likelihood ?? 0) * (props.severity ?? 0)))
const normalized = computed(() => (props.rank ?? '').toLowerCase())
const level = computed(() => riskLevel(normalized.value, numeric.value))
const label = computed(() => props.rank || (level.value === 'high' ? '高风险' : level.value === 'medium' ? '中风险' : '低风险'))
</script>

<template>
  <span class="risk-badge" :class="`risk-${level}`"><span class="risk-symbol" aria-hidden="true" />{{ label }}</span>
</template>
