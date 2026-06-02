<template>
  <router-view v-if="publicPage" />
  <el-container v-else class="shell">
    <AppSidebar :active-menu="activeMenu" :role="auth.role" />
    <el-container>
      <el-header class="topbar">
        <div class="topbar-title">
          <span class="topbar-mark"></span>
          <span>{{ pageTitle }}</span>
        </div>
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
              :label="`${item.course_code} / ${item.class_name}`"
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
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppSidebar from './components/AppSidebar.vue'
import { useAuthStore } from './stores/auth'
import { useClassroomStore } from './stores/classroom'
import { useExamLockStore } from './stores/examLock'

const auth = useAuthStore()
const classroom = useClassroomStore()
const examLock = useExamLockStore()
const router = useRouter()
const route = useRoute()

const publicPage = computed(() => ['/login', '/register', '/forgot-password'].includes(route.path))
const initials = computed(() => (auth.user?.name || auth.user?.email || 'U').trim().slice(0, 1).toUpperCase())
const activeMenu = computed(() => String(route.meta.activeMenu || route.path))
const pageTitle = computed(() => {
  return String(route.meta.title || '黄海在线题测平台')
})

function logout() {
  auth.logout()
  classroom.clear()
  router.push('/login')
}

function setClass(value: number) {
  if (examLock.locked) {
    ElMessage.warning(examLock.message)
    return
  }
  classroom.setActive(value)
}

function handleCommand(command: string) {
  if (examLock.locked && command !== 'theme') {
    ElMessage.warning(examLock.message)
    return
  }
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

onMounted(() => {
  examLock.hydrate()
  if (auth.isAuthed) classroom.load()
})

watch(
  () => auth.isAuthed,
  (authed) => {
    if (authed) classroom.load()
    else classroom.clear()
  }
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
  background: var(--glass);
  border-bottom: 1px solid var(--border);
  backdrop-filter: blur(18px);
}

.topbar-title {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  color: var(--text);
  font-weight: 700;
}

.topbar-actions {
  display: inline-flex;
  align-items: center;
  gap: 12px;
}

.class-switch {
  width: 240px;
}

.topbar-mark {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--accent);
  box-shadow: 0 0 0 5px rgba(10, 94, 166, 0.12);
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

@media (max-width: 760px) {
  .shell {
    display: block;
  }

  .topbar {
    min-height: 58px;
  }

  .class-switch {
    width: min(240px, 48vw);
  }
}
</style>
