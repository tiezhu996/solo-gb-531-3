<script setup lang="ts">
import { ShieldAlert } from 'lucide-vue-next'
</script>

<template>
  <div class="safety-boundary" role="note">
    <ShieldAlert :size="19" aria-hidden="true" />
    <div>
      <strong>离线辅助边界</strong>
      <span>结果不能替代持证工艺安全人员判断；系统不连接 PLC、SIS、阀门或生产设备，也不能下发控制指令。</span>
    </div>
  </div>
</template>
