import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import type { UserRole } from '../types/auth'

export function useAuth() {
  const store = useAuthStore()
  const router = useRouter()
  const canEdit = computed(() => store.hasRole('admin', 'process_engineer'))
  const canReview = computed(() => store.hasRole('admin', 'safety_reviewer'))
  const isAuditor = computed(() => store.hasRole('auditor'))
  const allowed = (...roles: UserRole[]) => store.hasRole(...roles)
  const expire = () => { store.logout(); void router.replace({ name: 'login', query: { expired: '1' } }) }
  return { user: computed(() => store.user), role: computed(() => store.role), canEdit, canReview, isAuditor, allowed, logout: expire }
}
