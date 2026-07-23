<template>
  <div class="admin-layout">
    <aside class="admin-sidebar" :class="{ open: mobileOpen }">
      <RouterLink to="/admin" class="admin-brand" @click="mobileOpen = false">
        <img src="/logo1.png" alt="" />
        <div><strong>教学管理后台</strong><span>School OJ Console</span></div>
      </RouterLink>
      <el-menu router :default-active="activeMenu" class="admin-menu" @select="mobileOpen = false">
        <el-menu-item v-for="item in visibleItems" :key="item.path" :index="item.path">
          <span class="menu-dot" aria-hidden="true"></span><span>{{ item.label }}</span>
        </el-menu-item>
      </el-menu>
      <div class="admin-sidebar-footer">
        <el-button plain @click="router.push('/')">返回前台</el-button>
        <el-button text @click="toggleTheme">{{ auth.theme === 'dark' ? '浅色' : '深色' }}</el-button>
      </div>
    </aside>
    <div v-if="mobileOpen" class="sidebar-mask" @click="mobileOpen = false"></div>
    <main class="admin-content">
      <header class="admin-topbar">
        <div class="admin-topbar-title">
          <button class="mobile-menu" type="button" @click="mobileOpen = !mobileOpen">☰</button>
          <div><span class="muted">管理后台</span><h2>{{ route.meta.title }}</h2></div>
        </div>
        <div class="admin-user">
          <span>{{ auth.user?.name }}</span>
          <button type="button" @click="router.push('/profile')">{{ initials }}</button>
        </div>
      </header>
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const mobileOpen = ref(false)
const items = [
  { path: '/admin', label: '管理概览' },
  { path: '/admin/courses', label: '课程管理' },
  { path: '/admin/classes', label: '班级管理' },
  { path: '/admin/prepared-problems', label: '预备题库' },
  { path: '/admin/plagiarism', label: 'JPlag 查重' },
  { path: '/admin/audit-logs', label: '审计日志', roles: ['admin'] },
  { path: '/admin/users', label: '用户管理', roles: ['admin'] }
]
const visibleItems = computed(() => items.filter((item) => !item.roles || item.roles.includes(auth.role || '')))
const activeMenu = computed(() => String(route.meta.adminMenu || '/admin'))
const initials = computed(() => (auth.user?.name || auth.user?.email || 'U').slice(0, 1).toUpperCase())

function toggleTheme() {
  auth.toggleTheme()
}
</script>

<style scoped>
.admin-layout { min-height: 100vh; display: grid; grid-template-columns: 250px minmax(0, 1fr); background: var(--app-bg); }
.admin-sidebar { position: sticky; top: 0; z-index: 30; height: 100vh; display: flex; flex-direction: column; color: #e2e8f0; background: #07182a; border-right: 1px solid rgba(255,255,255,.08); }
.admin-brand { display: flex; align-items: center; gap: 12px; padding: 24px 20px; color: #fff; border-bottom: 1px solid rgba(255,255,255,.08); }
.admin-brand img { width: 42px; height: 42px; object-fit: cover; border-radius: 10px; background: #fff; }
.admin-brand div { display: grid; gap: 3px; }
.admin-brand span { color: #7dd3fc; font-size: 11px; letter-spacing: .08em; }
.admin-menu { flex: 1; padding: 16px 10px; border-right: 0; background: transparent; }
.admin-menu :deep(.el-menu-item) { gap: 12px; height: 48px; margin: 4px 0; color: #a9bdd2; border-radius: 9px; }
.admin-menu :deep(.el-menu-item:hover), .admin-menu :deep(.el-menu-item.is-active) { color: #fff; background: rgba(14, 165, 233, .16); }
.menu-dot { width: 7px; height: 7px; flex: 0 0 auto; border: 1px solid #38bdf8; border-radius: 50%; }
.admin-menu :deep(.el-menu-item.is-active) .menu-dot { background: #38bdf8; box-shadow: 0 0 0 4px rgba(56,189,248,.12); }
.admin-sidebar-footer { display: flex; padding: 18px; border-top: 1px solid rgba(255,255,255,.08); }
.admin-content { min-width: 0; }
.admin-topbar { position: sticky; top: 0; z-index: 20; display: flex; align-items: center; justify-content: space-between; min-height: 72px; padding: 0 28px; border-bottom: 1px solid var(--border); background: var(--glass); backdrop-filter: blur(16px); }
.admin-topbar-title { display: flex; align-items: center; gap: 14px; }
.admin-topbar-title span { font-size: 12px; }
.admin-topbar-title h2 { margin: 2px 0 0; font-size: 20px; }
.admin-user { display: flex; align-items: center; gap: 12px; }
.admin-user button { width: 38px; height: 38px; color: #fff; border: 0; border-radius: 50%; background: linear-gradient(135deg, #0a5ea6, #0f766e); cursor: pointer; }
.mobile-menu { display: none; font-size: 20px; border: 0; background: transparent; color: var(--text); }
.sidebar-mask { display: none; }
.admin-content :deep(.page) { max-width: none; }
@media (max-width: 820px) { .admin-layout { grid-template-columns: 1fr; } .admin-sidebar { position: fixed; left: -260px; transition: left .2s ease; } .admin-sidebar.open { left: 0; } .sidebar-mask { position: fixed; inset: 0; z-index: 25; display: block; background: rgba(2, 6, 23, .55); } .mobile-menu { display: inline-grid; } .admin-topbar { padding: 0 16px; } }
</style>
