<template>
  <router-view v-if="authPage" />
  <el-container v-else direction="vertical" class="shell" :class="{ 'exam-shell': studentExamWorkspace, 'admin-shell': adminWorkspace, 'course-shell': courseWorkspace }">
    <el-header v-if="!studentExamWorkspace && !adminWorkspace && !courseWorkspace" class="topbar" height="auto">
      <AppSidebar :active-menu="activeMenu" :role="auth.role" :authenticated="auth.isAuthed" />
      <div class="topbar-actions">
        <el-dropdown v-if="auth.isAuthed" trigger="click" @command="handleCommand">
          <button class="avatar-button" type="button">
            <img v-if="auth.user?.avatar_url" :src="auth.user.avatar_url" alt="" />
            <span v-else>{{ initials }}</span>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">个人中心</el-dropdown-item>
              <el-dropdown-item command="update-profile">更新个人资料</el-dropdown-item>
              <el-dropdown-item v-if="['problem_setter', 'teacher', 'admin'].includes(auth.role || '')" command="author">创建题目</el-dropdown-item>
              <el-dropdown-item v-if="auth.role === 'teacher' || auth.role === 'admin'" command="admin">后台管理</el-dropdown-item>
              <el-dropdown-item command="theme">
                {{ auth.theme === 'dark' ? '切换明亮模式' : '切换暗黑模式' }}
              </el-dropdown-item>
              <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button v-else type="primary" @click="goLogin">登录</el-button>
      </div>
    </el-header>
    <el-main class="main-content" :class="{ 'exam-main-content': studentExamWorkspace, 'admin-main-content': adminWorkspace, 'course-main-content': courseWorkspace }">
      <router-view />
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { ElMessageBox } from 'element-plus'
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppSidebar from './components/AppSidebar.vue'
import { useAuthStore } from './stores/auth'
import { useExamLockStore } from './stores/examLock'

const auth = useAuthStore()
const examLock = useExamLockStore()
const router = useRouter()
const route = useRoute()
let activeExamPromptOpen = false
let lastPromptedExamId: number | undefined

const authPage = computed(() => ['/login', '/register', '/forgot-password'].includes(route.path))
const initials = computed(() => (auth.user?.name || auth.user?.email || 'U').trim().slice(0, 1).toUpperCase())
const activeMenu = computed(() => String(route.meta.activeMenu || route.path))
const currentExamRouteId = computed(() => {
  const value = route.params.id
  return typeof value === 'string' ? Number(value) : undefined
})
const studentExamWorkspace = computed(() => auth.role === 'student' && Boolean(currentExamRouteId.value) && route.path.startsWith('/exams/'))
const adminWorkspace = computed(() => route.path === '/admin' || route.path.startsWith('/admin/'))
const courseWorkspace = computed(() => /^\/my\/courses\/[^/]+/.test(route.path))
const activeExamRoot = computed(() => (examLock.examId ? `/exams/${examLock.examId}` : ''))
const inActiveExam = computed(() => Boolean(activeExamRoot.value) && (route.path === activeExamRoot.value || route.path.startsWith(`${activeExamRoot.value}/`)))

function logout() {
  auth.logout()
  examLock.unlock()
  router.push('/login')
}

function goLogin() {
  router.push({ path: '/login', query: { redirect: route.fullPath } })
}

function handleCommand(command: string) {
  if (command === 'profile') {
    router.push('/profile')
    return
  }
  if (command === 'update-profile') {
    router.push({ path: '/profile', query: { edit: '1' } })
    return
  }
  if (command === 'admin') {
    router.push('/admin')
    return
  }
  if (command === 'author') {
    router.push('/problems/create')
    return
  }
  if (command === 'theme') {
    auth.toggleTheme()
    return
  }
  if (command === 'logout') {
    logout()
  }
}

async function maybeShowActiveExamPrompt() {
  if (!auth.isAuthed || auth.role !== 'student' || authPage.value) return
  try {
    await examLock.syncActiveExam()
  } catch {
    return
  }
  if (!examLock.locked || !examLock.examId) {
    lastPromptedExamId = undefined
    return
  }
  if (inActiveExam.value) {
    lastPromptedExamId = undefined
    return
  }
  if (activeExamPromptOpen || lastPromptedExamId === examLock.examId) return
  const examId = examLock.examId
  activeExamPromptOpen = true
  lastPromptedExamId = examId
  try {
    await ElMessageBox.confirm(examLock.message, '正在进行的考试', {
      type: 'warning',
      confirmButtonText: '返回考试',
      cancelButtonText: '继续浏览',
      distinguishCancelAndClose: true,
      customClass: 'active-exam-dialog'
    })
    if (examLock.examId === examId) router.push(`/exams/${examId}/problems`)
  } catch {
    // The student may continue browsing; the reminder is intentionally not a lock.
  } finally {
    activeExamPromptOpen = false
  }
}

watch(
  () => auth.isAuthed,
  (authed) => {
    if (!authed) {
      examLock.unlock()
      lastPromptedExamId = undefined
    }
  }
)

watch(
  () => [auth.isAuthed, auth.role, route.fullPath],
  () => {
    void maybeShowActiveExamPrompt()
  },
  { immediate: true }
)
</script>

<style scoped>
.shell {
  min-height: 100vh;
  background:
    linear-gradient(180deg, rgba(10, 94, 166, 0.05), transparent 320px),
    var(--app-bg);
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 64px;
  padding: 0 18px;
  background: var(--glass);
  border-bottom: 1px solid var(--border);
  backdrop-filter: blur(18px);
}

.topbar-actions {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  flex: 0 0 auto;
}

.avatar-button {
  width: 40px;
  height: 40px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: linear-gradient(135deg, var(--accent), #14b8a6);
  color: #fff;
  font-weight: 800;
  cursor: pointer;
  overflow: hidden;
  transition: transform 0.18s ease, box-shadow 0.18s ease;
}

.avatar-button:hover {
  transform: translateY(-1px);
  box-shadow: 0 12px 28px rgba(10, 94, 166, 0.18);
}

.avatar-button img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.main-content {
  padding: 0;
}

.exam-shell {
  min-height: 100vh;
}

.admin-shell,
.admin-main-content,
.course-shell,
.course-main-content {
  min-height: 100vh;
}

.admin-main-content,
.course-main-content {
  padding: 0;
}

.exam-main-content :deep(.page) {
  min-height: 100vh;
}

:global(.active-exam-dialog) {
  border: 2px solid #f59e0b;
  box-shadow: 0 28px 80px rgba(146, 64, 14, 0.28);
}

:global(.active-exam-dialog .el-message-box__title) {
  color: #92400e;
  font-weight: 800;
}

:global(.active-exam-dialog .el-message-box__message) {
  color: var(--text);
  font-size: 16px;
  line-height: 1.7;
}

@media (max-width: 760px) {
  .topbar {
    align-items: flex-start;
    flex-direction: column;
    padding: 10px 12px;
  }

}
</style>
