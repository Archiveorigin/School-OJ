<template>
  <section class="page dash-page">
    <!-- Hero Banner -->
    <div class="dash-hero" :style="{ backgroundImage: `url(${heroBg})` }">
      <div class="dash-hero-overlay">
        <div class="dash-hero-inner">
          <div class="dash-hero-text">
            <h1 class="dash-hero-title">{{ roleTitle }}</h1>
            <p class="dash-hero-sub">{{ heroSubtitle }}</p>
          </div>
          <div class="dash-hero-stats" v-if="!showNoClassState">
            <div class="dash-hero-stat" v-for="s in heroStats" :key="s.label">
              <span class="dash-hero-stat-val">{{ s.value }}</span>
              <span class="dash-hero-stat-label">{{ s.label }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Error State -->
    <div v-if="loadError" class="panel dash-error">
      <strong>概览加载失败</strong>
      <span>{{ loadError }}</span>
      <el-button size="small" @click="refresh">重试</el-button>
    </div>

    <!-- No-class State -->
    <div v-if="showNoClassState" class="dash-no-class-wrap">
      <div class="dash-no-class-card">
        <div class="dash-no-class-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" width="48" height="48">
            <path d="M12 2L2 7l10 5 10-5-10-5z"/>
            <path d="M2 17l10 5 10-5"/>
            <path d="M2 12l10 5 10-5"/>
          </svg>
        </div>
        <h3>加入课程，开始学习</h3>
        <p class="muted">通过教师提供的课程邀请码或班级邀请码加入，即可访问题库、作业与考试。</p>
        <div class="dash-join-row">
          <el-input v-model="joinClassCode" placeholder="输入邀请码（课程或班级）" class="dash-join-input" size="large" />
          <el-button type="primary" size="large" :loading="joining" @click="joinClass">加入</el-button>
        </div>
      </div>
    </div>

    <!-- Main Content (when class/course available) -->
    <template v-if="!showNoClassState">
      <!-- Quick Stats -->
      <div class="dash-stats-row">
        <div class="dash-stat-card" v-for="s in stats" :key="s.label">
          <div class="dash-stat-icon" :class="s.iconClass" v-html="s.icon"></div>
          <div class="dash-stat-body">
            <strong>{{ s.value }}</strong>
            <span class="muted">{{ s.label }}</span>
          </div>
        </div>
      </div>

      <!-- Main Grid -->
      <div class="dash-main-grid">
        <!-- Left: Role Panel + Quick Info -->
        <div class="dash-main-left">
          <div class="panel dash-panel-flush">
            <div class="section-title">
              <h3>{{ rolePanelTitle }}</h3>
            </div>
            <div class="dash-quick-grid">
              <div v-for="card in roleCards" :key="card.label" class="dash-quick-card">
                <span class="dash-quick-val">{{ card.value }}</span>
                <span class="dash-quick-label muted">{{ card.label }}</span>
              </div>
            </div>
            <div class="dash-actions-row">
              <el-button v-for="btn in roleActionButtons" :key="btn.label" :type="btn.type || 'default'" size="small" @click="btn.action">
                {{ btn.label }}
              </el-button>
            </div>
          </div>
        </div>

        <!-- Right: Fortune Card -->
        <div class="dash-main-right">
          <div class="panel dash-fortune">
            <div class="dash-fortune-header">
              <span class="dash-fortune-badge">{{ fortune.badge }}</span>
              <span class="dash-fortune-date">{{ dateStr }}</span>
            </div>
            <strong class="dash-fortune-title">{{ fortune.title }}</strong>
            <p class="dash-fortune-tip muted">{{ fortune.tip }}</p>
            <div class="dash-fortune-tags">
              <el-tag type="success" effect="plain" size="small">宜 {{ fortune.good }}</el-tag>
              <el-tag type="info" effect="plain" size="small">推荐 {{ fortune.lang }}</el-tag>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- Bottom Feature Section -->
    <div class="dash-bottom" :style="{ backgroundImage: `url(${bottomBg})` }" v-if="!showNoClassState">
      <div class="dash-bottom-overlay">
        <h3 class="dash-bottom-heading">快速导航</h3>
        <p class="dash-bottom-sub">探索更多功能</p>
        <div class="dash-bottom-grid">
          <div class="dash-feature-card" v-for="card in featureCards" :key="card.label" @click="card.action">
            <div class="dash-feature-icon" v-html="card.icon"></div>
            <div class="dash-feature-body">
              <span class="dash-feature-label">{{ card.label }}</span>
              <span class="dash-feature-desc muted">{{ card.desc }}</span>
            </div>
            <svg class="dash-feature-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { client, type Submission } from '../api/client'
import { useAuthStore } from '../stores/auth'
import { useClassroomStore } from '../stores/classroom'

const courses = ref<any[]>([])
const problems = ref<any[]>([])
const submissions = ref<Submission[]>([])
const assignments = ref<any[]>([])
const exams = ref<any[]>([])
const loading = ref(false)
const loadError = ref('')
const joinClassCode = ref('')
const joining = ref(false)
const auth = useAuthStore()
const classroom = useClassroomStore()
const router = useRouter()

// Background images (WebP with JPEG fallback via server-side)
const heroBg = '/bg-hero.webp'
const bottomBg = '/bg-cards.webp'

const showNoClassState = computed(() => auth.role === 'student' && classroom.loaded && classroom.classes.length === 0)

const dateStr = computed(() => {
  const d = new Date()
  const w = ['日', '一', '二', '三', '四', '五', '六'][d.getDay()]
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()} 星期${w}`
})

const stats = computed(() => [
  {
    label: auth.role === 'student' ? '已加入课程' : '课程',
    value: courses.value.length,
    icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="22" height="22"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>',
    iconClass: 'dash-icon-courses'
  },
  {
    label: '当前题目',
    value: problems.value.length,
    icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="22" height="22"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>',
    iconClass: 'dash-icon-problems'
  },
  {
    label: '近期提交',
    value: submissions.value.length,
    icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="22" height="22"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>',
    iconClass: 'dash-icon-submissions'
  },
  {
    label: '已通过',
    value: submissions.value.filter((s) => s.status === 'accepted').length,
    icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="22" height="22"><polyline points="20 6 9 17 4 12"/></svg>',
    iconClass: 'dash-icon-accepted'
  }
])

const heroStats = computed(() => [
  { label: '课程', value: courses.value.length },
  { label: '题目', value: problems.value.length },
  { label: '提交', value: submissions.value.length },
  { label: 'AC', value: submissions.value.filter((s) => s.status === 'accepted').length }
])

const roleTitle = computed(() => {
  if (auth.role === 'admin') return '欢迎回来，管理员'
  if (auth.role === 'teacher') return '教学工作台'
  return '学习工作台'
})

const heroSubtitle = computed(() => {
  if (classroom.loading && !classroom.loaded) return '正在加载...'
  const item = classroom.activeClass
  if (item) return `${item.course_code} ${item.course_name} · ${item.class_name}`
  if (auth.role === 'teacher') return '选择班级以查看课程数据'
  if (auth.role === 'admin') return '系统管理概览'
  return '加入课程以开始学习'
})

const rolePanelTitle = computed(() => {
  if (auth.role === 'admin') return '系统运行概况'
  if (auth.role === 'teacher') return '教学安排'
  return '我的学习'
})

const roleCards = computed(() => [
  { label: auth.role === 'student' ? '我的班级' : '可管理班级', value: classroom.classes.length },
  { label: '作业', value: assignments.value.length },
  { label: '考试', value: exams.value.length }
])

const roleActionButtons = computed(() => {
  const common: { label: string; action: () => void; type?: string }[] = [
    { label: '题库', action: () => router.push('/problems') },
    { label: '全部提交', action: () => router.push('/submissions') }
  ]
  if (auth.role === 'student') {
    return [
      ...common,
      { label: '作业', action: () => router.push('/assignments') },
      { label: '考试', action: () => router.push('/exams') }
    ]
  }
  return [
    ...common,
    { label: '课程管理', action: () => router.push('/courses/list'), type: 'primary' },
    { label: '班级管理', action: () => router.push('/classes') }
  ]
})

const featureCards = computed(() => {
  const cards = [
    {
      label: '题库',
      desc: '浏览全部题目与预备题库',
      icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="22" height="22"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>',
      action: () => router.push('/problems')
    },
    {
      label: '考试',
      desc: '查看进行中的考试与成绩',
      icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="22" height="22"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>',
      action: () => router.push('/exams')
    },
    {
      label: '作业',
      desc: '查看已布置的作业任务',
      icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="22" height="22"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>',
      action: () => router.push('/assignments')
    }
  ]
  if (auth.role === 'teacher' || auth.role === 'admin') {
    cards.push({
      label: '课程学生',
      desc: '查看课程下的学生名单',
      icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="22" height="22"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>',
      action: () => router.push('/courses/list')
    })
  }
  return cards.slice(0, 4)
})

const fortune = computed(() => {
  const seed = `${new Date().toISOString().slice(0, 10)}-${auth.user?.id || 0}`
  const sum = Array.from(seed).reduce((acc, char) => acc + char.charCodeAt(0), 0)
  const options = [
    { badge: '轻盈', title: '适合补齐一个小缺口', tip: '把最小的待办先合并掉，今天的节奏会更顺。', good: '写测试', lang: 'Go' },
    { badge: '清晰', title: '适合读题和拆解边界', tip: '先确认输入输出和极端数据，再动手会更稳。', good: '整理题面', lang: 'Python' },
    { badge: '高效', title: '适合完成一次提交闭环', tip: '从编译到 AC 的反馈链路会给你明确方向。', good: '调试样例', lang: 'C++' },
    { badge: '专注', title: '适合复盘排行榜变化', tip: '看一眼自己和同学的差距，再定下一题。', good: '复盘错题', lang: 'Java' }
  ]
  return options[sum % options.length]
})

function list<T>(value: T[] | null | undefined) {
  return Array.isArray(value) ? value : []
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const params = classroom.activeClassId ? { class_id: classroom.activeClassId } : {}
    const skipClassScoped = showNoClassState.value
    const [c, p, s, a, e] = await Promise.all([
      client.get('/courses'),
      skipClassScoped ? Promise.resolve({ data: [] }) : client.get('/problems', { params }),
      client.get('/submissions'),
      skipClassScoped ? Promise.resolve({ data: [] }) : client.get('/assignments', { params }),
      skipClassScoped ? Promise.resolve({ data: [] }) : client.get('/exams', { params })
    ])
    courses.value = list(c.data)
    problems.value = list(p.data)
    submissions.value = list(s.data)
    assignments.value = list(a.data)
    exams.value = list(e.data)
  } catch (err: any) {
    loadError.value = err.response?.data?.error || err.message || '未知错误'
  } finally {
    loading.value = false
  }
}

async function refresh() {
  await classroom.load({ force: true })
  await load()
}

async function joinClass() {
  const code = joinClassCode.value.trim()
  if (!code) {
    ElMessage.error('请填写邀请码')
    return
  }
  joining.value = true
  try {
    const upperCode = code.toUpperCase()
    if (upperCode.startsWith('R')) {
      const { data } = await client.post('/courses/join', { join_code: upperCode })
      ElMessage.success('已加入课程')
    } else {
      const { data } = await client.post('/classes/join', { join_code: upperCode })
      ElMessage.success('已加入班级')
      classroom.setActive(data.class_id)
    }
    joinClassCode.value = ''
    await classroom.load({ force: true })
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    joining.value = false
  }
}

watch(
  () => classroom.activeClassId,
  () => {
    if (classroom.loaded) load()
  }
)

onMounted(async () => {
  await classroom.load()
  await load()
})
</script>

<style scoped>
.dash-page {
  padding: 0;
  overflow-x: hidden;
}

/* ====== HERO BANNER ====== */
.dash-hero {
  position: relative;
  height: clamp(240px, 34vw, 380px);
  background-size: cover;
  background-position: center;
}

.dash-hero-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    135deg,
    rgba(15, 23, 42, 0.84) 0%,
    rgba(15, 23, 42, 0.55) 45%,
    rgba(10, 94, 166, 0.3) 100%
  );
  display: flex;
  align-items: center;
}

.dash-hero-inner {
  max-width: 1200px;
  width: 100%;
  margin: 0 auto;
  padding: 32px 36px 36px;
  display: flex;
  flex-direction: column;
  gap: 28px;
}

.dash-hero-text {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.dash-hero-title {
  margin: 0;
  font-size: clamp(22px, 3.2vw, 34px);
  font-weight: 700;
  color: #f8fafc;
  text-shadow: 0 2px 12px rgba(0, 0, 0, 0.35);
}

.dash-hero-sub {
  margin: 0;
  font-size: 15px;
  color: rgba(248, 250, 252, 0.7);
}

.dash-hero-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
}

.dash-hero-stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 12px 20px;
  background: rgba(255, 255, 255, 0.11);
  border: 1px solid rgba(255, 255, 255, 0.17);
  border-radius: 12px;
  backdrop-filter: blur(10px);
  min-width: 80px;
  text-align: center;
  transition: background 0.2s;
}

.dash-hero-stat:hover {
  background: rgba(255, 255, 255, 0.18);
}

.dash-hero-stat-val {
  font-size: 22px;
  font-weight: 700;
  color: #fff;
}

.dash-hero-stat-label {
  font-size: 12px;
  color: rgba(248, 250, 252, 0.6);
}

/* ====== ERROR STATE ====== */
.dash-error {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 16px 20px 0;
  color: #b42318;
}

/* ====== NO-CLASS STATE ====== */
.dash-no-class-wrap {
  max-width: 600px;
  margin: -48px auto 0;
  padding: 0 20px;
  position: relative;
  z-index: 2;
}

.dash-no-class-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  padding: 40px 32px;
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.12);
  backdrop-filter: blur(18px);
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

:root.dark .dash-no-class-card {
  box-shadow: 0 18px 48px rgba(0, 0, 0, 0.3);
}

.dash-no-class-icon {
  width: 80px;
  height: 80px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: linear-gradient(135deg, color-mix(in srgb, var(--accent) 20%, transparent), color-mix(in srgb, #14b8a6 20%, transparent));
  color: var(--accent);
  margin-bottom: 4px;
}

.dash-no-class-card h3 {
  margin: 0;
  font-size: 20px;
  color: var(--text);
}

.dash-no-class-card > p {
  margin: 0 0 8px;
  max-width: 380px;
  line-height: 1.6;
}

.dash-join-row {
  display: flex;
  gap: 12px;
  width: 100%;
  max-width: 420px;
}

.dash-join-input {
  flex: 1;
}

/* ====== STATS ROW ====== */
.dash-stats-row {
  max-width: 1200px;
  margin: -36px auto 0;
  padding: 0 20px;
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
  position: relative;
  z-index: 2;
}

.dash-stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 18px 20px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: 0 4px 16px rgba(15, 23, 42, 0.06);
  backdrop-filter: blur(12px);
  cursor: default;
  transition: transform 0.2s, box-shadow 0.2s, border-color 0.2s;
}

.dash-stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.1);
  border-color: color-mix(in srgb, var(--accent) 40%, var(--border));
}

.dash-stat-icon {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  border-radius: 12px;
  color: #fff;
}

.dash-icon-courses {
  background: linear-gradient(135deg, #0a5ea6, #4aa3ff);
}

.dash-icon-problems {
  background: linear-gradient(135deg, #6366f1, #818cf8);
}

.dash-icon-submissions {
  background: linear-gradient(135deg, #f59e0b, #fbbf24);
}

.dash-icon-accepted {
  background: linear-gradient(135deg, #16a34a, #4ade80);
}

.dash-stat-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.dash-stat-body strong {
  font-size: 24px;
  color: var(--text);
}

.dash-stat-body span {
  font-size: 13px;
}

/* ====== MAIN GRID ====== */
.dash-main-grid {
  max-width: 1200px;
  margin: 20px auto 0;
  padding: 0 20px;
  display: grid;
  grid-template-columns: 1fr;
  gap: 16px;
}

.dash-panel-flush {
  padding: 20px 24px;
}

.dash-quick-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-bottom: 16px;
}

.dash-quick-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: color-mix(in srgb, var(--surface-strong) 72%, transparent);
  transition: border-color 0.2s;
}

.dash-quick-card:hover {
  border-color: color-mix(in srgb, var(--accent) 35%, var(--border));
}

.dash-quick-val {
  font-size: 26px;
  font-weight: 700;
  color: var(--text);
}

.dash-quick-label {
  font-size: 13px;
}

.dash-actions-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

/* ====== FORTUNE CARD ====== */
.dash-fortune {
  padding: 20px 24px;
  position: relative;
  overflow: hidden;
}

.dash-fortune::before {
  content: '';
  position: absolute;
  top: -40px;
  right: -40px;
  width: 140px;
  height: 140px;
  border-radius: 50%;
  background: radial-gradient(circle, color-mix(in srgb, var(--accent) 12%, transparent), transparent 70%);
  pointer-events: none;
}

.dash-fortune-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.dash-fortune-badge {
  font-size: 13px;
  font-weight: 600;
  padding: 4px 12px;
  border-radius: 20px;
  background: color-mix(in srgb, var(--accent) 14%, transparent);
  color: var(--accent);
}

.dash-fortune-date {
  font-size: 12px;
  color: var(--muted);
}

.dash-fortune-title {
  display: block;
  margin-bottom: 8px;
  font-size: 20px;
  color: var(--text);
}

.dash-fortune-tip {
  margin: 0 0 14px;
  line-height: 1.55;
}

.dash-fortune-tags {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

/* ====== BOTTOM FEATURE SECTION ====== */
.dash-bottom {
  margin-top: 32px;
  background-size: cover;
  background-position: center;
  background-attachment: fixed;
}

.dash-bottom-overlay {
  padding: 48px 20px 56px;
  background: linear-gradient(
    180deg,
    rgba(15, 23, 42, 0.9) 0%,
    rgba(15, 23, 42, 0.78) 100%
  );
  backdrop-filter: blur(3px);
}

.dash-bottom-heading {
  max-width: 1200px;
  margin: 0 auto 8px;
  font-size: 22px;
  font-weight: 700;
  color: #f8fafc;
  text-align: center;
}

.dash-bottom-sub {
  max-width: 1200px;
  margin: 0 auto 28px;
  font-size: 14px;
  color: rgba(248, 250, 252, 0.6);
  text-align: center;
}

.dash-bottom-grid {
  max-width: 1200px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 14px;
}

.dash-feature-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 18px 22px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 12px;
  backdrop-filter: blur(8px);
  cursor: pointer;
  transition: background 0.2s, border-color 0.2s, transform 0.2s;
}

.dash-feature-card:hover {
  background: rgba(255, 255, 255, 0.15);
  border-color: rgba(74, 163, 255, 0.4);
  transform: translateX(4px);
}

.dash-feature-icon {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  display: grid;
  place-items: center;
  border-radius: 10px;
  background: rgba(74, 163, 255, 0.2);
  color: #83c4ff;
}

.dash-feature-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.dash-feature-label {
  font-size: 15px;
  font-weight: 600;
  color: #f8fafc;
}

.dash-feature-desc {
  font-size: 13px;
  color: rgba(248, 250, 252, 0.55);
}

.dash-feature-arrow {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  color: rgba(248, 250, 252, 0.3);
  transition: transform 0.2s, color 0.2s;
}

.dash-feature-card:hover .dash-feature-arrow {
  transform: translateX(3px);
  color: #fff;
}

/* ====== RESPONSIVE ====== */
@media (min-width: 901px) {
  .dash-main-grid {
    grid-template-columns: 1.2fr 0.8fr;
  }
}

@media (max-width: 900px) {
  .dash-hero-inner {
    padding: 28px 20px 28px;
    gap: 20px;
  }

  .dash-stats-row {
    grid-template-columns: repeat(2, 1fr);
    margin-top: -28px;
  }

  .dash-bottom-grid {
    grid-template-columns: 1fr;
  }

  .dash-quick-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 600px) {
  .dash-hero {
    height: auto;
    min-height: 220px;
  }

  .dash-hero-inner {
    padding: 24px 16px 24px;
    gap: 16px;
  }

  .dash-hero-stats {
    gap: 8px;
  }

  .dash-hero-stat {
    padding: 8px 14px;
  }

  .dash-hero-stat-val {
    font-size: 18px;
  }

  .dash-stats-row {
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    margin-top: -24px;
    padding: 0 12px;
  }

  .dash-stat-card {
    padding: 14px;
    gap: 10px;
  }

  .dash-stat-icon {
    width: 40px;
    height: 40px;
    border-radius: 10px;
  }

  .dash-stat-body strong {
    font-size: 20px;
  }

  .dash-main-grid {
    padding: 0 12px;
    gap: 12px;
  }

  .dash-quick-grid {
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
  }

  .dash-quick-card {
    padding: 10px 12px;
  }

  .dash-quick-val {
    font-size: 22px;
  }

  .dash-bottom-overlay {
    padding: 36px 16px 40px;
  }

  .dash-join-row {
    flex-direction: column;
    align-items: stretch;
  }

  .dash-no-class-card {
    padding: 28px 20px;
  }
}
</style>
