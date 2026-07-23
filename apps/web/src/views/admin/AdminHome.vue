<template>
  <section class="admin-home">
    <div class="admin-welcome">
      <div><span>CONTROL CENTER</span><h1>教学与系统管理</h1><p>课程相关配置和管理工具已统一归集到这里。</p></div>
      <el-button type="primary" @click="router.push('/')">返回前台</el-button>
    </div>
    <div class="admin-card-grid">
      <button v-for="item in cards" :key="item.path" type="button" class="panel admin-card" @click="router.push(item.path)">
        <span>{{ item.mark }}</span><strong>{{ item.label }}</strong><small>{{ item.description }}</small>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const cards = computed(() => [
  { path: '/admin/courses', label: '课程管理', description: '维护课程资料、成员与课程邀请码', mark: 'COURSE' },
  { path: '/admin/classes', label: '班级管理', description: '维护班级、学生名单和班级邀请码', mark: 'CLASS' },
  { path: '/admin/prepared-problems', label: '预备题库', description: '管理未公开题目并发布到公共题库', mark: 'PREP' },
  { path: '/admin/plagiarism', label: 'JPlag 查重', description: '按课程任务创建代码相似度分析', mark: 'CHECK' },
  ...(auth.role === 'admin' ? [
    { path: '/admin/audit-logs', label: '审计日志', description: '查看关键操作与资源变更记录', mark: 'AUDIT' },
    { path: '/admin/users', label: '用户管理', description: '管理平台账号、角色和登录凭据', mark: 'USER' }
  ] : [])
])
</script>

<style scoped>
.admin-home { padding: 32px; }
.admin-welcome { display: flex; align-items: end; justify-content: space-between; gap: 24px; padding: 36px 40px; color: #fff; border-radius: 20px; background: linear-gradient(120deg, #0f172a, #0a5ea6); }
.admin-welcome span { color: #7dd3fc; font-size: 11px; font-weight: 800; letter-spacing: .16em; }
.admin-welcome h1 { margin: 10px 0 6px; font-size: 34px; }
.admin-welcome p { margin: 0; color: #bfdbfe; }
.admin-card-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 18px; margin-top: 22px; }
.admin-card { display: grid; gap: 11px; min-height: 170px; padding: 24px; text-align: left; color: var(--text); cursor: pointer; }
.admin-card:hover { border-color: var(--accent); transform: translateY(-2px); }
.admin-card span { color: var(--accent); font-size: 10px; font-weight: 800; letter-spacing: .14em; }
.admin-card strong { font-size: 20px; }
.admin-card small { color: var(--muted); line-height: 1.6; }
@media (max-width: 1000px) { .admin-card-grid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 620px) { .admin-home { padding: 18px 14px; } .admin-welcome { align-items: stretch; flex-direction: column; padding: 26px 22px; } .admin-card-grid { grid-template-columns: 1fr; } }
</style>
