<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { KeyRound, ShieldCheck } from 'lucide-vue-next'
import { useAuthStore } from '../stores/auth'
import { errorMessage } from '../api/client'
import SafetyBoundary from '../components/common/SafetyBoundary.vue'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const loading = ref(false)
const form = reactive({ username: 'engineer', password: 'engineer123' })
const accounts = [
  { label: '工艺工程师', username: 'engineer', password: 'engineer123' },
  { label: '安全复核员', username: 'reviewer', password: 'reviewer123' },
  { label: '审计员', username: 'auditor', password: 'auditor123' },
  { label: '管理员', username: 'admin', password: 'admin123' },
]
async function submit() {
  loading.value = true
  try {
    await auth.login(form)
    await router.replace(typeof route.query.redirect === 'string' ? route.query.redirect : '/nodes')
  } catch (error) { ElMessage.error(errorMessage(error)) }
  finally { loading.value = false }
}
function select(account: typeof accounts[number]) { form.username = account.username; form.password = account.password }
</script>

<template>
  <div class="login-shell">
    <section class="login-identity">
      <div class="brand-block large"><span class="brand-mark">HZ</span><div><strong>HAZOP Coverage</strong><span>Process safety evidence</span></div></div>
      <div class="worksheet-mark">
        <p class="eyebrow">OFFLINE SAFEGUARD REVIEW</p>
        <h1>保护层覆盖分析</h1>
        <p>把节点边界、偏差路径、独立保护层与不可变评估快照放在同一条证据链上。</p>
      </div>
      <SafetyBoundary />
    </section>
    <section class="login-panel">
      <div class="login-form-wrap">
        <ShieldCheck :size="28" />
        <p class="eyebrow">AUTHORIZED REVIEWERS</p>
        <h2>进入离线评审工作台</h2>
        <el-alert v-if="route.query.expired" title="会话已过期，请重新登录" type="warning" :closable="false" />
        <el-form label-position="top" @submit.prevent="submit">
          <el-form-item label="账号"><el-input v-model="form.username" autocomplete="username" /></el-form-item>
          <el-form-item label="密码"><el-input v-model="form.password" type="password" show-password autocomplete="current-password" /></el-form-item>
          <el-button type="primary" native-type="submit" :loading="loading"><KeyRound :size="16" />登录</el-button>
        </el-form>
        <div class="account-switch"><span>演示身份：</span><el-button v-for="account in accounts" :key="account.username" text size="small" @click="select(account)">{{ account.label }}</el-button></div>
      </div>
    </section>
  </div>
</template>
