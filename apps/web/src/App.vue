<template>
  <router-view v-if="publicPage" />
  <el-container v-else class="shell">
    <el-aside width="220px" class="aside">
      <div class="brand">
        <img class="brand-logo" src="/logo.jpg" alt="黄海在线题测平台" />
        <span>黄海在线题测平台</span>
      </div>
      <el-menu router :default-active="$route.path" class="nav">
        <el-menu-item index="/">概览</el-menu-item>
        <el-menu-item index="/courses">课程班级</el-menu-item>
        <el-menu-item index="/problems">题库</el-menu-item>
        <el-menu-item index="/assignments">作业</el-menu-item>
        <el-menu-item index="/exams">考试</el-menu-item>
        <el-menu-item index="/submissions">提交</el-menu-item>
        <el-menu-item index="/leaderboard">排行榜</el-menu-item>
        <el-menu-item v-if="auth.role !== 'student'" index="/plagiarism">JPlag 查重</el-menu-item>
        <el-menu-item v-if="auth.role === 'admin'" index="/audit-logs">审计日志</el-menu-item>
        <el-menu-item v-if="auth.role === 'admin'" index="/users">用户</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="topbar">
        <div class="topbar-title">
          <span class="topbar-mark"></span>
          <span>{{ pageTitle }}</span>
        </div>
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
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const publicPage = computed(() => ['/login', '/register', '/forgot-password'].includes(route.path))
const initials = computed(() => (auth.user?.name || auth.user?.email || 'U').trim().slice(0, 1).toUpperCase())
const pageTitle = computed(() => {
  const map: Record<string, string> = {
    '/': '概览',
    '/courses': '课程班级',
    '/problems': '题库',
    '/assignments': '作业',
    '/exams': '考试',
    '/submissions': '提交',
    '/leaderboard': '排行榜',
    '/plagiarism': 'JPlag 查重',
    '/audit-logs': '审计日志',
    '/users': '用户管理',
    '/profile': 'Profile'
  }
  return map[route.path] || '黄海在线题测平台'
})

function logout() {
  auth.logout()
  router.push('/login')
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
</script>

<style scoped>
.shell {
  min-height: 100vh;
  background:
    linear-gradient(180deg, rgba(10, 94, 166, 0.05), transparent 320px),
    var(--app-bg);
}

.aside {
  background: var(--surface-strong);
  border-right: 1px solid var(--border);
  box-shadow: 8px 0 28px rgba(15, 23, 42, 0.04);
}

.brand {
  min-height: 64px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 16px;
  font-weight: 800;
  line-height: 1.2;
  color: var(--text);
  border-bottom: 1px solid var(--border);
}

.brand-logo {
  width: 36px;
  height: 36px;
  object-fit: cover;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 8px 20px rgba(10, 94, 166, 0.15);
}

.nav {
  border-right: 0;
  background: transparent;
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
</style>
