<template>
  <section class="overview-page">
    <div class="overview-hero">
      <div class="hero-copy">
        <span class="eyebrow">HUANGHAI ONLINE ASSESSMENT</span>
        <h1>黄海在线测试平台</h1>
        <p>以开放题库承载知识积累，以课程空间连接教学过程，让每一次练习、作业与考试都沉淀为清晰的学习轨迹。</p>
        <div class="hero-actions">
          <el-button type="primary" size="large" @click="router.push('/problems')">进入公共题库</el-button>
          <el-button v-if="auth.isAuthed" size="large" @click="router.push('/my/courses')">查看我的课程</el-button>
          <el-button v-else size="large" @click="router.push('/login')">登录平台</el-button>
        </div>
      </div>
      <aside class="academic-summary" aria-label="平台概览">
        <div class="summary-emblem">
          <img src="/logo1.png" alt="" />
          <span>笃学 · 求真 · 知行合一</span>
        </div>
        <div class="summary-data">
          <div><strong>{{ problems.length }}</strong><span>开放题目</span></div>
          <div><strong>{{ acceptedCount }}</strong><span>{{ auth.isAuthed ? '已通过题目' : '学术资源' }}</span></div>
          <div><strong>{{ courses.length }}</strong><span>{{ auth.isAuthed ? '在学课程' : '课程空间' }}</span></div>
        </div>
      </aside>
    </div>

    <div v-if="loadError" class="panel error-panel">
      <span>{{ loadError }}</span>
      <el-button size="small" @click="load">重新加载</el-button>
    </div>

    <section class="academic-overview">
      <div class="section-heading">
        <div>
          <span class="eyebrow">ACADEMIC JOURNEY</span>
          <h2>从知识研习到能力验证</h2>
          <p>围绕教学、实践与评价建立连贯的在线学习空间。</p>
        </div>
      </div>
      <div class="academic-grid">
        <article class="academic-card">
          <span class="academic-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20V3H6.5A2.5 2.5 0 0 0 4 5.5v14Z"/><path d="M8 7h8M8 11h6"/></svg>
          </span>
          <div>
            <h3>公共题库</h3>
            <p>以规范题目、运行限制与评测反馈构成开放的专业训练资源。</p>
            <el-button text type="primary" @click="router.push('/problems')">开始研习 →</el-button>
          </div>
        </article>
        <article class="academic-card">
          <span class="academic-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24"><path d="M9 3h6M10 3v5l-5.5 9.2A2.5 2.5 0 0 0 6.7 21h10.6a2.5 2.5 0 0 0 2.2-3.8L14 8V3"/><path d="M7.5 16h9"/></svg>
          </span>
          <div>
            <h3>课程研习</h3>
            <p>在独立课程空间中组织课程信息、作业实践与阶段性考试。</p>
            <el-button text type="primary" @click="goCourses">{{ auth.isAuthed ? '进入课程空间' : '登录后使用' }} →</el-button>
          </div>
        </article>
        <article class="academic-card">
          <span class="academic-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24"><path d="m3 10 9-5 9 5-9 5-9-5Z"/><path d="M7 13.5V18c3 2 7 2 10 0v-4.5M21 10v6"/></svg>
          </span>
          <div>
            <h3>教学评价</h3>
            <p>通过可信评测、学习活跃度与课程数据支持教学反馈和能力成长。</p>
            <el-button v-if="canManage" text type="primary" @click="router.push('/admin')">进入教学管理 →</el-button>
            <span v-else class="academic-note">依角色授权访问</span>
          </div>
        </article>
      </div>
    </section>

    <section class="campus-banner">
      <div>
        <span class="eyebrow">KNOWLEDGE · PRACTICE · GROWTH</span>
        <h2>让严谨的思考，在每一次代码运行中得到验证</h2>
        <p>面向学院课程教学的在线测试环境，记录过程，尊重探索，关注真实能力的形成。</p>
      </div>
      <el-button v-if="auth.isAuthed" size="large" @click="router.push('/profile')">查看学习活跃度</el-button>
    </section>

    <section class="panel latest-panel">
      <div class="section-heading compact">
        <div>
          <span class="eyebrow">OPEN PROBLEM BANK</span>
          <h2>最新公共题目</h2>
        </div>
        <el-button text type="primary" @click="router.push('/problems')">查看全部</el-button>
      </div>
      <el-table :data="latestProblems" @row-click="openProblem">
        <el-table-column prop="display_code" label="题号" width="120" />
        <el-table-column prop="title" label="题目" min-width="240" />
        <el-table-column label="学习状态" width="130">
          <template #default="{ row }">
            <el-tag v-if="row.progress_status === 'accepted'" type="success" effect="plain">已通过</el-tag>
            <el-tag v-else-if="row.progress_status === 'attempted'" type="warning" effect="plain">已尝试</el-tag>
            <span v-else class="muted">可研习</span>
          </template>
        </el-table-column>
      </el-table>
    </section>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { client, type Problem } from '../api/client'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const problems = ref<Problem[]>([])
const courses = ref<any[]>([])
const loadError = ref('')

const canManage = computed(() => auth.role === 'teacher' || auth.role === 'admin')
const acceptedCount = computed(() => problems.value.filter((item) => item.progress_status === 'accepted').length)
const latestProblems = computed(() => problems.value.slice(0, 8))

async function load() {
  loadError.value = ''
  try {
    problems.value = (await client.get('/problems')).data || []
    if (auth.isAuthed) courses.value = (await client.get('/courses')).data || []
  } catch (err: any) {
    loadError.value = err.response?.data?.error || err.message || '概览加载失败'
  }
}

function goCourses() {
  router.push(auth.isAuthed ? '/my/courses' : '/login')
}

function openProblem(problem: Problem) {
  router.push(`/problems/${problem.display_code || problem.id}`)
}

onMounted(load)
</script>

<style scoped>
.overview-page { max-width: 1320px; margin: 0 auto; padding: 28px 28px 72px; }
.overview-hero { position: relative; min-height: 470px; display: grid; grid-template-columns: minmax(0, 1.45fr) minmax(300px, .62fr); align-items: center; gap: 54px; padding: 58px 64px; overflow: hidden; border-radius: 30px; color: #fff; background: linear-gradient(112deg, rgba(3, 30, 55, .96), rgba(7, 73, 122, .86) 56%, rgba(10, 94, 166, .62)), url('/bg-hero.webp') center/cover; box-shadow: 0 30px 80px rgba(8, 47, 73, .25); }
.overview-hero::before { content: ''; position: absolute; inset: 22px; pointer-events: none; border: 1px solid rgba(255,255,255,.18); border-radius: 21px; }
.overview-hero::after { content: '黄海学院 · 在线测试'; position: absolute; right: 34px; bottom: 28px; color: rgba(255,255,255,.24); font-family: Georgia, 'Times New Roman', serif; font-size: 12px; letter-spacing: .3em; }
.hero-copy, .academic-summary { position: relative; z-index: 1; }
.eyebrow { color: #67e8f9; font-size: 11px; font-weight: 800; letter-spacing: .2em; }
.hero-copy h1 { margin: 15px 0 18px; font-family: 'Noto Serif SC', 'Songti SC', SimSun, serif; font-size: clamp(40px, 5.6vw, 68px); font-weight: 700; letter-spacing: .04em; line-height: 1.08; }
.hero-copy p { max-width: 720px; margin: 0; color: #dbeafe; font-size: 17px; line-height: 1.9; }
.hero-actions { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 32px; }
.academic-summary { overflow: hidden; border: 1px solid rgba(255,255,255,.24); border-radius: 20px; background: rgba(3, 28, 49, .44); box-shadow: 0 18px 50px rgba(0,0,0,.16); backdrop-filter: blur(15px); }
.summary-emblem { display: grid; place-items: center; gap: 10px; padding: 26px; border-bottom: 1px solid rgba(255,255,255,.14); }
.summary-emblem img { width: 72px; height: 72px; padding: 5px; object-fit: contain; border-radius: 18px; background: rgba(255,255,255,.94); }
.summary-emblem span { color: #bae6fd; font-family: 'Noto Serif SC', SimSun, serif; letter-spacing: .14em; }
.summary-data { display: grid; grid-template-columns: repeat(3, 1fr); }
.summary-data div { display: grid; gap: 4px; padding: 20px 10px; text-align: center; border-right: 1px solid rgba(255,255,255,.12); }
.summary-data div:last-child { border-right: 0; }
.summary-data strong { font-size: 28px; }
.summary-data span { color: #bfdbfe; font-size: 12px; }
.error-panel { display: flex; justify-content: space-between; gap: 16px; margin-top: 20px; padding: 18px 22px; }
.academic-overview { padding: 60px 6px 18px; }
.section-heading { display: flex; align-items: end; justify-content: space-between; gap: 18px; margin-bottom: 22px; }
.section-heading h2 { margin: 8px 0 5px; font-family: 'Noto Serif SC', 'Songti SC', SimSun, serif; font-size: 30px; }
.section-heading p { margin: 0; color: var(--muted); }
.academic-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 18px; }
.academic-card { display: flex; gap: 20px; min-height: 220px; padding: 28px; border: 1px solid var(--border); border-top: 3px solid color-mix(in srgb, var(--accent) 78%, #fff); border-radius: 8px 8px 18px 18px; background: linear-gradient(155deg, var(--surface-strong), var(--surface)); box-shadow: 0 15px 38px rgba(15, 50, 78, .07); }
.academic-icon { width: 52px; height: 52px; display: grid; place-items: center; flex: 0 0 auto; color: var(--accent); border: 1px solid color-mix(in srgb, var(--accent) 26%, var(--border)); border-radius: 50%; background: color-mix(in srgb, var(--accent) 9%, var(--surface)); }
.academic-icon svg { width: 27px; fill: none; stroke: currentColor; stroke-width: 1.5; stroke-linecap: round; stroke-linejoin: round; }
.academic-card h3 { margin: 3px 0 10px; font-family: 'Noto Serif SC', SimSun, serif; font-size: 21px; }
.academic-card p { min-height: 76px; margin: 0 0 8px; color: var(--muted); line-height: 1.8; }
.academic-note { display: inline-block; padding: 10px 0; color: var(--muted); font-size: 13px; }
.campus-banner { min-height: 260px; display: flex; align-items: center; justify-content: space-between; gap: 32px; margin-top: 42px; padding: 46px 54px; color: #fff; border-radius: 22px; background: linear-gradient(100deg, rgba(4, 30, 55, .94), rgba(4, 62, 94, .76)), url('/bg-cards.webp') center/cover; }
.campus-banner h2 { max-width: 720px; margin: 12px 0; font-family: 'Noto Serif SC', SimSun, serif; font-size: clamp(27px, 3.5vw, 40px); }
.campus-banner p { max-width: 760px; margin: 0; color: #dbeafe; line-height: 1.8; }
.latest-panel { margin-top: 28px; padding: 26px; }
.section-heading.compact { align-items: center; margin: 0 0 16px; }
.latest-panel :deep(.el-table__row) { cursor: pointer; }
@media (max-width: 980px) { .overview-hero { grid-template-columns: 1fr; padding: 48px 42px; } .academic-grid { grid-template-columns: 1fr; } .academic-card p { min-height: 0; } }
@media (max-width: 680px) { .overview-page { padding: 16px 13px 48px; } .overview-hero { min-height: auto; gap: 32px; padding: 42px 24px; border-radius: 20px; } .overview-hero::before { inset: 10px; } .summary-data { grid-template-columns: 1fr; } .summary-data div { border-right: 0; border-bottom: 1px solid rgba(255,255,255,.12); } .summary-data div:last-child { border-bottom: 0; } .hero-actions, .campus-banner { align-items: stretch; flex-direction: column; } .academic-card { flex-direction: column; } .campus-banner { padding: 34px 24px; } }
</style>
