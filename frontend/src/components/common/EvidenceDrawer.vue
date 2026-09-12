<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { FileCheck2, X } from 'lucide-vue-next'

const props = withDefaults(defineProps<{ modelValue: boolean; title?: string; note?: string; evidence?: unknown }>(), { title: '证据详情', note: '' })
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const open = ref(props.modelValue)
watch(() => props.modelValue, (value) => { open.value = value })
watch(open, (value) => emit('update:modelValue', value))
const rendered = computed(() => typeof props.evidence === 'string' ? props.evidence : JSON.stringify(props.evidence ?? {}, null, 2))
</script>

<template>
  <el-drawer v-model="open" size="min(520px, 92vw)" direction="rtl" :with-header="false" class="evidence-drawer">
    <header class="drawer-header">
      <div><FileCheck2 :size="19" /><div><p class="eyebrow">EVIDENCE CHAIN</p><h2>{{ title }}</h2></div></div>
      <el-tooltip content="关闭证据详情"><el-button circle text aria-label="关闭" @click="open = false"><X :size="18" /></el-button></el-tooltip>
    </header>
    <p v-if="note" class="drawer-note">{{ note }}</p>
    <slot><pre class="evidence-json">{{ rendered }}</pre></slot>
    <footer>证据仅用于离线评审与审计追溯，不构成设备控制指令。</footer>
  </el-drawer>
</template>
