<template>
  <router-view v-if="publicPage" />
  <el-container v-else direction="vertical" class="shell" :class="{ 'exam-shell': studentExamWorkspace }">
    <el-header v-if="!studentExamWorkspace" class="topbar" height="auto">
      <AppSidebar :active-menu="activeMenu" :role="auth.role" />
      <div class="topbar-actions">
        <el-select
          v-if="auth.isAuthed && classroom.classes.length"
          :model-value="classroom.activeClassId"
          class="class-switch"
          filterable
          @change="setClass"
        >
          <el-option
            v-for="item in classroom.classes"
            :key="item.class_id"
            :label="classOptionLabel(item)"
            :value="item.class_id"
          />
        </el-select>
        <el-dropdown trigger="click" @command="handleCommand">
          <button class="avatar-button" type="button">
            <img v-if="auth.user?.avatar_url" :src="auth.user.avatar_url" alt="" />
            <span v-else>{{ initials }}</span>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">Profile</el-dropdown-item>
              <el-dropdown-item command="theme">
                {{ auth.theme === 'dark' ? '切换明亮模式' : '切换暗黑模式' }}
              </el-dropdown-item>
              <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </el-header>
    <el-main class="main-content" :class="{ 'exam-main-content': studentExamWorkspace }">
      <router-view />
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { ElMessageBox } from 'element-plus'
import { computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppSidebar from './components/AppSidebar.vue'
import { useAuthStore } from './stores/auth'
import { useClassroomStore } from './stores/classroom'
import { useExamLockStore } from './stores/examLock'
import type { ClassContext } from './api/client'

const auth = useAuthStore()
const classroom = useClassroomStore()
const examLock = useExamLockStore()
const router = useRouter()
const route = useRoute()
let activeExamPromptOpen = false
let lastPromptedExamId: number | undefined

const publicPage = computed(() => ['/login', '/register', '/forgot-password'].includes(route.path))
const initials = computed(() => (auth.user?.name || auth.user?.email || 'U').trim().slice(0, 1).toUpperCase())
const activeMenu = computed(() => String(route.meta.activeMenu || route.path))
const currentExamRouteId = computed(() => {
  const value = route.params.id
  return typeof value === 'string' ? Number(value) : undefined
})
const studentExamWorkspace = computed(() => auth.role === 'student' && Boolean(currentExamRouteId.value) && route.path.startsWith('/exams/'))
const activeExamRoot = computed(() => (examLock.examId ? `/exams/${examLock.examId}` : ''))
const inActiveExam = computed(() => Boolean(activeExamRoot.value) && (route.path === activeExamRoot.value || route.path.startsWith(`${activeExamRoot.value}/`)))

function logout() {
  auth.logout()
  examLock.unlock()
  classroom.clear()
  router.push('/login')
}

function setClass(value: number) {
  classroom.setActive(value)
}

function classOptionLabel(item: ClassContext) {
  return auth.role === 'student' ? item.class_name : `${item.course_code} / ${item.class_name}`
}

function handleCommand(command: string) {
  if (command === 'profile') {
    router.push('/profile')
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
  if (!auth.isAuthed || auth.role !== 'student' || publicPage.value) return
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

onMounted(() => {
  examLock.hydrate()
  if (auth.isAuthed) classroom.load()
})

watch(
  () => auth.isAuthed,
  (authed) => {
    if (authed) classroom.load()
    else {
      classroom.clear()
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

.class-switch {
  width: 240px;
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

  .class-switch {
    width: min(240px, calc(100vw - 82px));
  }
}
</style>
