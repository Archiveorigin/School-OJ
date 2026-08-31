<template>
  <div class="admin-layout">
    <aside class="admin-sidebar" :class="{ open: mobileOpen }">
      <RouterLink to="/admin" class="admin-brand" @click="mobileOpen = false">
        <span class="admin-brand-mark">
          <img :src="adminLogo" alt="青岛黄海学院" />
        </span>
        <span class="admin-brand-copy">
          <strong>黄海在线</strong>
          <small>教学管理平台</small>
        </span>
      </RouterLink>

      <nav class="admin-navigation" aria-label="教学管理导航">
        <section
          v-for="group in visibleGroups"
          :key="group.label"
          class="admin-nav-group"
        >
          <p>{{ group.label }}</p>
          <RouterLink
            v-for="item in group.items"
            :key="item.path"
            :to="item.path"
            :class="{ active: activeMenu === item.path }"
            :aria-current="activeMenu === item.path ? 'page' : undefined"
            @click="mobileOpen = false"
          >
            <el-icon><component :is="item.icon" /></el-icon>
            <span>{{ item.label }}</span>
          </RouterLink>
        </section>
      </nav>

      <div class="admin-sidebar-footer">
        <button type="button" @click="router.push('/')">
          <el-icon><House /></el-icon>
          <span>返回前台</span>
        </button>
        <button type="button" @click="toggleTheme">
          <el-icon><Setting /></el-icon>
          <span>{{ auth.theme === "dark" ? "浅色模式" : "深色模式" }}</span>
        </button>
        <small>青岛黄海学院 · School OJ</small>
      </div>
    </aside>

    <button
      v-if="mobileOpen"
      class="sidebar-mask"
      type="button"
      aria-label="关闭导航"
      @click="mobileOpen = false"
    ></button>

    <main class="admin-content">
      <header class="admin-topbar">
        <div class="admin-topbar-title">
          <button
            class="mobile-menu"
            type="button"
            aria-label="打开导航"
            @click="mobileOpen = !mobileOpen"
          >
            <el-icon><Menu /></el-icon>
          </button>
          <div class="admin-breadcrumb">
            <span>教学管理中心</span>
            <i aria-hidden="true">/</i>
            <strong>{{ route.meta.title }}</strong>
          </div>
        </div>

        <div class="admin-topbar-actions">
          <div class="admin-date">
            <el-icon><Calendar /></el-icon>
            <span>{{ currentDate }}</span>
          </div>
          <button
            class="notification-button"
            type="button"
            aria-label="个人中心"
            @click="router.push('/profile')"
          >
            <el-icon><Bell /></el-icon>
          </button>
          <el-dropdown trigger="click" @command="handleUserCommand">
            <button class="admin-user" type="button">
              <span class="admin-avatar">
                <img
                  v-if="auth.user?.avatar_url"
                  :src="auth.user.avatar_url"
                  alt=""
                />
                <span v-else>{{ initials }}</span>
              </span>
              <span class="admin-user-copy">
                <strong>{{ auth.user?.name || auth.user?.email }}</strong>
                <small>{{ roleLabel }}</small>
              </span>
              <el-icon><ArrowDown /></el-icon>
            </button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">个人中心</el-dropdown-item>
                <el-dropdown-item command="theme">切换主题</el-dropdown-item>
                <el-dropdown-item divided command="logout"
                  >退出登录</el-dropdown-item
                >
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import type { Component } from "vue";
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  ArrowDown,
  Bell,
  Calendar,
  Collection,
  DataAnalysis,
  Document,
  EditPen,
  House,
  Menu,
  Reading,
  Setting,
  User,
} from "@element-plus/icons-vue";
import { useAuthStore } from "../../stores/auth";

type NavigationItem = {
  path: string;
  label: string;
  icon: Component;
  roles?: string[];
};

const auth = useAuthStore();
const route = useRoute();
const router = useRouter();
const mobileOpen = ref(false);
const adminLogo = "/logo1.png";

const navigationGroups: Array<{ label: string; items: NavigationItem[] }> = [
  {
    label: "工作台",
    items: [{ path: "/admin", label: "教学概览", icon: House }],
  },
  {
    label: "教学组织",
    items: [
      { path: "/admin/courses", label: "课程管理", icon: Reading },
      { path: "/admin/classes", label: "班级管理", icon: User },
      { path: "/admin/exams", label: "考试管理", icon: Collection },
    ],
  },
  {
    label: "内容与评测",
    items: [
      { path: "/admin/prepared-problems", label: "预备题库", icon: EditPen },
      {
        path: "/admin/problem-authors",
        label: "出题管理",
        icon: Document,
        roles: ["admin"],
      },
      { path: "/admin/plagiarism", label: "JPlag 查重", icon: DataAnalysis },
    ],
  },
  {
    label: "系统治理",
    items: [
      {
        path: "/admin/users",
        label: "用户与权限",
        icon: User,
        roles: ["admin"],
      },
      {
        path: "/admin/audit-logs",
        label: "审计日志",
        icon: Setting,
        roles: ["admin"],
      },
    ],
  },
];

const visibleGroups = computed(() =>
  navigationGroups
    .map((group) => ({
      ...group,
      items: group.items.filter(
        (item) => !item.roles || item.roles.includes(auth.role || ""),
      ),
    }))
    .filter((group) => group.items.length),
);
const activeMenu = computed(() => String(route.meta.adminMenu || "/admin"));
const initials = computed(() =>
  (auth.user?.name || auth.user?.email || "U").trim().slice(0, 1).toUpperCase(),
);
const roleLabel = computed(() =>
  auth.role === "admin" ? "系统管理员" : "教师",
);
const currentDate = new Intl.DateTimeFormat("zh-CN", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  weekday: "short",
}).format(new Date());

function toggleTheme() {
  auth.toggleTheme();
}

function handleUserCommand(command: string) {
  if (command === "profile") {
    router.push("/profile");
    return;
  }
  if (command === "theme") {
    toggleTheme();
    return;
  }
  if (command === "logout") {
    auth.logout();
    router.push("/login");
  }
}
</script>

<style scoped>
.admin-layout {
  --admin-blue: #0a3d86;
  --admin-blue-strong: #072f69;
  --admin-accent: #135ecb;
  --admin-red: #c92c2c;
  --admin-paper: #ffffff;
  --admin-ink: #132a4d;
  --admin-muted: #66748a;
  --admin-line: #dce4ef;
  min-height: 100vh;
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  color: var(--admin-ink);
  background: #f7f9fc;
}

.admin-sidebar {
  position: sticky;
  top: 0;
  z-index: 30;
  height: 100vh;
  display: flex;
  min-width: 0;
  flex-direction: column;
  border-right: 1px solid var(--admin-line);
  background: var(--admin-paper);
}

.admin-brand {
  display: flex;
  min-height: 172px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 12px;
  padding: 24px 18px;
  color: #fff;
  text-align: center;
  background: var(--admin-blue);
}

.admin-brand:hover {
  color: #fff;
}

.admin-brand-mark {
  width: 78px;
  height: 78px;
  display: grid;
  place-items: center;
  flex: 0 0 auto;
  border-radius: 50%;
  background: #fff;
}

.admin-brand-mark img {
  width: 72px;
  height: 72px;
  object-fit: contain;
  border-radius: 50%;
}

.admin-brand-copy {
  display: grid;
  gap: 3px;
}

.admin-brand-copy strong {
  font-family: "STSong", "Songti SC", "Noto Serif SC", serif;
  font-size: 21px;
  letter-spacing: 0.08em;
}

.admin-brand-copy small {
  color: rgba(255, 255, 255, 0.78);
  font-size: 10px;
  letter-spacing: 0.08em;
}

.admin-navigation {
  flex: 1;
  overflow-y: auto;
  padding: 18px 12px;
}

.admin-nav-group + .admin-nav-group {
  margin-top: 18px;
}

.admin-nav-group p {
  margin: 0 12px 7px;
  color: #98a4b5;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.12em;
}

.admin-nav-group a {
  position: relative;
  display: flex;
  min-height: 44px;
  align-items: center;
  gap: 12px;
  margin: 2px 0;
  padding: 0 13px;
  color: #465a78;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 600;
  transition:
    background 0.18s ease,
    color 0.18s ease;
}

.admin-nav-group a:hover,
.admin-nav-group a.active {
  color: var(--admin-accent);
  background: #eef4fd;
}

.admin-nav-group a.active::before {
  position: absolute;
  left: -12px;
  width: 4px;
  height: 28px;
  border-radius: 0 4px 4px 0;
  background: var(--admin-accent);
  content: "";
}

.admin-nav-group .el-icon {
  font-size: 18px;
}

.admin-sidebar-footer {
  display: grid;
  gap: 2px;
  padding: 12px 12px 18px;
  border-top: 1px solid var(--admin-line);
}

.admin-sidebar-footer button {
  display: flex;
  min-height: 38px;
  align-items: center;
  gap: 10px;
  padding: 0 12px;
  border: 0;
  color: var(--admin-muted);
  background: transparent;
  cursor: pointer;
}

.admin-sidebar-footer button:hover {
  color: var(--admin-accent);
}

.admin-sidebar-footer small {
  padding: 10px 12px 0;
  color: #a0aaba;
  font-size: 10px;
  line-height: 1.5;
}

.admin-content {
  min-width: 0;
}

.admin-topbar {
  position: sticky;
  top: 0;
  z-index: 24;
  display: flex;
  min-height: 64px;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 0 28px;
  border-bottom: 1px solid var(--admin-line);
  background: rgba(255, 255, 255, 0.96);
  backdrop-filter: blur(16px);
}

.admin-topbar-title,
.admin-topbar-actions,
.admin-date,
.admin-user {
  display: flex;
  align-items: center;
}

.admin-breadcrumb {
  display: flex;
  align-items: center;
  gap: 11px;
  font-size: 14px;
}

.admin-breadcrumb span {
  color: var(--admin-muted);
}

.admin-breadcrumb i {
  color: #c0c8d4;
  font-style: normal;
}

.admin-breadcrumb strong {
  color: var(--admin-ink);
}

.admin-topbar-actions {
  gap: 15px;
}

.admin-date {
  gap: 8px;
  padding-right: 16px;
  color: #40536f;
  border-right: 1px solid var(--admin-line);
  font-size: 13px;
}

.notification-button {
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  border: 0;
  color: #40536f;
  background: transparent;
  cursor: pointer;
}

.admin-user {
  gap: 9px;
  padding: 0;
  border: 0;
  color: var(--admin-ink);
  background: transparent;
  cursor: pointer;
}

.admin-avatar {
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  overflow: hidden;
  color: #fff;
  border-radius: 50%;
  background: var(--admin-blue);
  font-weight: 800;
}

.admin-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.admin-user-copy {
  display: grid;
  gap: 2px;
  text-align: left;
}

.admin-user-copy strong {
  max-width: 130px;
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.admin-user-copy small {
  color: var(--admin-muted);
  font-size: 11px;
}

.mobile-menu,
.sidebar-mask {
  display: none;
}

.admin-content :deep(.page) {
  max-width: none;
  min-height: calc(100vh - 64px);
  padding: 26px 30px 34px;
}

.admin-content :deep(.sub-page) {
  padding: 0;
}

.admin-content :deep(.sub-hero) {
  border-bottom: 1px solid var(--admin-line);
  background: #fff;
}

.admin-content :deep(.sub-hero-inner) {
  max-width: none;
  padding: 28px 30px;
}

.admin-content :deep(.sub-hero-title),
.admin-content :deep(.sub-hero-stat-val) {
  color: var(--admin-ink);
}

.admin-content :deep(.sub-hero-sub),
.admin-content :deep(.sub-hero-stat-label) {
  color: var(--admin-muted);
}

.admin-content :deep(.sub-content) {
  max-width: none;
  padding: 24px 30px 34px;
}

.admin-content :deep(.panel) {
  border-color: var(--admin-line);
  border-radius: 5px;
  background: #fff;
  box-shadow: none;
  backdrop-filter: none;
}

.admin-content :deep(.panel:hover) {
  border-color: #b9c8dd;
  box-shadow: none;
}

.admin-content :deep(.el-button--primary) {
  --el-button-bg-color: var(--admin-accent);
  --el-button-border-color: var(--admin-accent);
  --el-button-hover-bg-color: #0f52b3;
  --el-button-hover-border-color: #0f52b3;
}

@media (max-width: 900px) {
  .admin-layout {
    grid-template-columns: 1fr;
  }

  .admin-sidebar {
    position: fixed;
    left: -230px;
    width: 220px;
    transition: left 0.2s ease;
  }

  .admin-sidebar.open {
    left: 0;
  }

  .sidebar-mask {
    position: fixed;
    inset: 0;
    z-index: 25;
    display: block;
    border: 0;
    background: rgba(8, 25, 50, 0.48);
  }

  .mobile-menu {
    width: 44px;
    height: 44px;
    display: grid;
    place-items: center;
    margin-right: 10px;
    border: 0;
    color: var(--admin-ink);
    background: transparent;
  }
}

@media (max-width: 680px) {
  .admin-topbar {
    min-height: 58px;
    padding: 0 14px;
  }

  .admin-breadcrumb span,
  .admin-breadcrumb i,
  .admin-date,
  .notification-button,
  .admin-user-copy,
  .admin-user > .el-icon {
    display: none;
  }

  .admin-content :deep(.page),
  .admin-content :deep(.sub-content) {
    padding: 18px 14px 26px;
  }

  .admin-content :deep(.sub-hero-inner) {
    padding: 22px 14px;
  }
}
</style>
