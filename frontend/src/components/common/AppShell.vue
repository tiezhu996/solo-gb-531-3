<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Boxes, ClipboardCheck, FileSearch, LogOut, Network, ShieldCheck } from 'lucide-vue-next'
import { useAuthStore } from '../../stores/auth'
import SafetyBoundary from './SafetyBoundary.vue'

const router = useRouter()
const auth = useAuthStore()
const links = [
  { to: '/nodes', label: '工艺节点', icon: Boxes },
  { to: '/deviations', label: '偏差分析', icon: FileSearch },
  { to: '/safeguards', label: '保护层台账', icon: ShieldCheck },
  { to: '/coverage', label: '覆盖推演', icon: Network },
  { to: '/audit', label: '审计中心', icon: ClipboardCheck, roles: ['admin', 'safety_reviewer', 'auditor'] },
]
const visibleLinks = computed(() => links.filter((link) => !link.roles || (auth.role && link.roles.includes(auth.role))))
function logout() { auth.logout(); void router.replace('/login') }
onMounted(() => window.addEventListener('hazop-auth-expired', logout))
onUnmounted(() => window.removeEventListener('hazop-auth-expired', logout))
</script>

<template>
  <div class="app-shell">
    <aside class="app-sidebar">
      <div class="brand-block">
        <span class="brand-mark">HZ</span>
        <div><strong>HAZOP Coverage</strong><span>Process safety evidence</span></div>
      </div>
      <nav aria-label="主导航">
        <RouterLink v-for="link in visibleLinks" :key="link.to" :to="link.to" class="nav-link">
          <component :is="link.icon" :size="17" aria-hidden="true" /><span>{{ link.label }}</span>
        </RouterLink>
      </nav>
      <div class="identity-block">
        <span>{{ auth.user?.display_name || auth.user?.username }}</span>
        <strong>{{ auth.role }}</strong>
        <el-tooltip content="退出当前会话" placement="top">
          <el-button text aria-label="退出" @click="logout"><LogOut :size="16" />退出</el-button>
        </el-tooltip>
      </div>
    </aside>
    <main class="app-main">
      <div class="context-strip"><span>OFFLINE REVIEW WORKSPACE</span><strong>NO CONTROL OUTPUT</strong></div>
      <SafetyBoundary />
      <div class="page-wrap"><slot /></div>
    </main>
  </div>
</template>
