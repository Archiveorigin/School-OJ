<template>
  <section class="overview-page">
    <div class="overview-hero">
      <div class="hero-copy">
        <span class="eyebrow">HUANGHAI ONLINE JUDGE</span>
        <h1>面向教学全过程的在线评测平台</h1>
        <p>公共题库统一开放，课程、作业与考试按个人身份和所属课程精准组织。</p>
        <div class="hero-actions">
          <el-button type="primary" size="large" @click="router.push('/problems')">浏览题库</el-button>
          <el-button v-if="auth.isAuthed" size="large" @click="router.push('/my/courses')">进入我的课程</el-button>
          <el-button v-else size="large" @click="router.push('/login')">登录平台</el-button>
        </div>
      </div>
      <div class="hero-summary panel">
        <div><strong>{{ problems.length }}</strong><span>公共题目</span></div>
        <div><strong>{{ acceptedCount }}</strong><span>{{ auth.isAuthed ? '我的通过' : '开放访问' }}</span></div>
        <div><strong>{{ courses.length }}</strong><span>{{ auth.isAuthed ? '我的课程' : '课程空间' }}</span></div>
      </div>
    </div>

    <div v-if="loadError" class="panel error-panel">
      <span>{{ loadError }}</span>
      <el-button size="small" @click="load">重新加载</el-button>
    </div>

    <div class="overview-grid">
      <article class="panel feature-card">
        <span class="feature-index">01</span>
        <h2>公共题库</h2>
        <p>无需登录即可查看已发布题目；登录后可提交代码并持续记录解题进度。</p>
        <el-button text type="primary" @click="router.push('/problems')">进入题库 →</el-button>
      </article>
      <article class="panel feature-card">
        <span class="feature-index">02</span>
        <h2>课程空间</h2>
        <p>每门课程拥有独立空间，课程概况、作业与考试以选项栏集中呈现。</p>
        <el-button text type="primary" @click="goPersonal">{{ auth.isAuthed ? '查看我的课程' : '登录后使用' }} →</el-button>
      </article>
      <article class="panel feature-card">
        <span class="feature-index">03</span>
        <h2>教学管理</h2>
        <p>课程班级、预备题库、查重、审计和用户管理统一收纳到独立后台。</p>
        <el-button v-if="canManage" text type="primary" @click="router.push('/admin')">进入后台 →</el-button>
        <span v-else class="muted">按角色授权访问</span>
      </article>
    </div>

    <div v-if="auth.isAuthed" class="personal-section">
      <div class="section-heading">
        <div>
          <span class="eyebrow">PERSONAL WORKSPACE</span>
          <h2>{{ auth.user?.name || '你好' }}，从这里继续</h2>
        </div>
        <el-button @click="router.push('/profile')">个人中心</el-button>
      </div>
      <div class="personal-grid">
        <button class="personal-card" type="button" @click="router.push('/my/courses')">
          <strong>{{ courses.length }}</strong><span>我的课程</span><small>查看课程、班级、作业与考试</small>
        </button>
        <button class="personal-card" type="button" @click="router.push('/submissions')">
          <strong>{{ submissions.length }}</strong><span>近期提交</span><small>追踪代码状态与评测结果</small>
        </button>
        <button class="personal-card" type="button" @click="router.push('/problems')">
          <strong>{{ acceptedCount }}</strong><span>已通过题目</span><small>继续挑战公共题库</small>
        </button>
      </div>
    </div>

    <div class="panel latest-panel">
      <div class="section-heading compact">
        <div>
          <span class="eyebrow">PROBLEM BANK</span>
          <h2>最新公共题目</h2>
        </div>
        <el-button text type="primary" @click="router.push('/problems')">查看全部</el-button>
      </div>
      <el-table :data="latestProblems" @row-click="openProblem">
        <el-table-column prop="display_code" label="题号" width="120" />
        <el-table-column prop="title" label="标题" min-width="240" />
        <el-table-column label="状态" width="130">
          <template #default="{ row }">
            <el-tag v-if="row.progress_status === 'accepted'" type="success" effect="plain">已通过</el-tag>
            <el-tag v-else-if="row.progress_status === 'attempted'" type="warning" effect="plain">已尝试</el-tag>
            <span v-else class="muted">可练习</span>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { client, type Problem, type Submission } from '../api/client'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const problems = ref<Problem[]>([])
const courses = ref<any[]>([])
const submissions = ref<Submission[]>([])
const loadError = ref('')

const canManage = computed(() => auth.role === 'teacher' || auth.role === 'admin')
const acceptedCount = computed(() => problems.value.filter((item) => item.progress_status === 'accepted').length)
const latestProblems = computed(() => problems.value.slice(0, 8))

async function load() {
  loadError.value = ''
  try {
    problems.value = (await client.get('/problems')).data || []
    if (!auth.isAuthed) return
    const [courseRes, submissionRes] = await Promise.all([
      client.get('/courses'),
      client.get('/submissions', { params: { limit: 20 } })
    ])
    courses.value = courseRes.data || []
    submissions.value = submissionRes.data || []
  } catch (err: any) {
    loadError.value = err.response?.data?.error || err.message || '概览加载失败'
  }
}

function goPersonal() {
  router.push(auth.isAuthed ? '/my/courses' : '/login')
}

function openProblem(problem: Problem) {
  router.push(`/problems/${problem.id}`)
}

onMounted(load)
</script>

<style scoped>
.overview-page { max-width: 1240px; margin: 0 auto; padding: 42px 28px 72px; }
.overview-hero { min-height: 390px; display: grid; grid-template-columns: minmax(0, 1.5fr) minmax(280px, .7fr); align-items: center; gap: 48px; padding: 52px; border-radius: 28px; color: #fff; background: linear-gradient(120deg, rgba(5, 34, 65, .96), rgba(10, 94, 166, .88)), url('/bg-hero.webp') center/cover; box-shadow: 0 28px 70px rgba(10, 50, 88, .22); }
.eyebrow { color: #38bdf8; font-size: 12px; font-weight: 800; letter-spacing: .16em; }
.hero-copy h1 { max-width: 760px; margin: 14px 0; font-size: clamp(34px, 5vw, 58px); line-height: 1.08; }
.hero-copy p { max-width: 680px; margin: 0; color: #dbeafe; font-size: 17px; line-height: 1.8; }
.hero-actions { display: flex; gap: 12px; margin-top: 28px; }
.hero-summary { display: grid; gap: 1px; padding: 0; overflow: hidden; border-color: rgba(255,255,255,.24); background: rgba(255,255,255,.12); backdrop-filter: blur(14px); }
.hero-summary div { display: flex; align-items: baseline; justify-content: space-between; gap: 16px; padding: 24px; border-bottom: 1px solid rgba(255,255,255,.14); }
.hero-summary div:last-child { border-bottom: 0; }
.hero-summary strong { color: #fff; font-size: 32px; }
.hero-summary span { color: #dbeafe; }
.error-panel { display: flex; justify-content: space-between; gap: 16px; margin-top: 20px; }
.overview-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 18px; margin-top: 24px; }
.feature-card { position: relative; min-height: 210px; padding: 28px; }
.feature-index { color: var(--accent); font-size: 12px; font-weight: 800; letter-spacing: .12em; }
.feature-card h2 { margin: 18px 0 8px; }
.feature-card p { min-height: 66px; color: var(--muted); line-height: 1.7; }
.personal-section, .latest-panel { margin-top: 48px; }
.section-heading { display: flex; align-items: end; justify-content: space-between; gap: 18px; margin-bottom: 18px; }
.section-heading h2 { margin: 8px 0 0; font-size: 28px; }
.section-heading.compact { align-items: center; margin: 0 0 14px; }
.personal-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
.personal-card { display: grid; gap: 6px; padding: 24px; text-align: left; color: var(--text); border: 1px solid var(--border); border-radius: 16px; background: var(--surface); cursor: pointer; transition: transform .18s ease, border-color .18s ease; }
.personal-card:hover { transform: translateY(-3px); border-color: var(--accent); }
.personal-card strong { color: var(--accent); font-size: 32px; }
.personal-card span { font-size: 17px; font-weight: 700; }
.personal-card small { color: var(--muted); }
.latest-panel { padding: 24px; }
.latest-panel :deep(.el-table__row) { cursor: pointer; }
@media (max-width: 900px) { .overview-hero { grid-template-columns: 1fr; padding: 34px; } .overview-grid, .personal-grid { grid-template-columns: 1fr; } }
@media (max-width: 560px) { .overview-page { padding: 22px 14px 48px; } .overview-hero { padding: 28px 22px; border-radius: 20px; } .hero-actions { align-items: stretch; flex-direction: column; } }
</style>
