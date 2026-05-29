<template>
  <router-view v-if="$route.path === '/login'" />
  <el-container v-else class="shell">
    <el-aside width="220px" class="aside">
      <div class="brand">School OJ</div>
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
        <span class="muted">{{ auth.user?.name }} / {{ auth.user?.role }}</span>
        <el-button @click="logout">退出</el-button>
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'

const auth = useAuthStore()
const router = useRouter()

function logout() {
  auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.shell {
  min-height: 100vh;
}

.aside {
  background: #ffffff;
  border-right: 1px solid #d9dee8;
}

.brand {
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 20px;
  font-weight: 700;
  border-bottom: 1px solid #d9dee8;
}

.nav {
  border-right: 0;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  background: #ffffff;
  border-bottom: 1px solid #d9dee8;
}
</style>
