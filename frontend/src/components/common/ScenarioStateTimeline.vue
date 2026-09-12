<script setup lang="ts">
import { computed } from 'vue'
import { Check, RotateCcw } from 'lucide-vue-next'
import type { ScenarioState } from '../../types/deviation-scenario'

const props = defineProps<{ state: ScenarioState }>()
const stages: { value: ScenarioState; label: string }[] = [
  { value: 'draft', label: '草稿' },
  { value: 'analyzed', label: '已分析' },
  { value: 'verified', label: '已复核' },
  { value: 'accepted', label: '已接受' },
]
const activeIndex = computed(() => props.state === 'rework' ? 1 : stages.findIndex((item) => item.value === props.state))
</script>

<template>
  <div class="scenario-timeline" :class="{ rework: state === 'rework' }" aria-label="偏差场景状态流">
    <div v-for="(stage, index) in stages" :key="stage.value" class="scenario-stage" :class="{ active: stage.value === state, passed: index < activeIndex }">
      <span class="stage-node"><Check v-if="index < activeIndex" :size="12" /><span v-else>{{ index + 1 }}</span></span>
      <strong>{{ stage.label }}</strong>
    </div>
    <div v-if="state === 'rework'" class="rework-marker"><RotateCcw :size="14" />退回修订</div>
  </div>
</template>
