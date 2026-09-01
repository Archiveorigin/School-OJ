<template>
  <section class="teaching-overview" v-loading="loading">
    <header class="overview-heading">
      <div>
        <span>TEACHING OVERVIEW</span>
        <h1>教学概览</h1>
        <p>聚焦正在进行的教学任务、近期考试和需要处理的事项。</p>
      </div>
      <div class="heading-actions">
        <time>{{ currentDate }}</time
        ><el-button @click="load">刷新</el-button
        ><el-button type="primary" @click="router.push('/admin/exams/new')">新建考试</el-button>
      </div>
    </header>

    <section class="metric-grid" aria-label="教学数据概览">
      <article>
        <span class="metric-mark blue"
          ><el-icon><Reading /></el-icon
        ></span>
        <div>
          <small>课程</small><strong>{{ courses.length }}</strong>
          <p>当前教学课程</p>
        </div>
      </article>
      <article>
        <span class="metric-mark cyan"
          ><el-icon><User /></el-icon
        ></span>
        <div>
          <small>班级</small><strong>{{ classes.length }}</strong>
          <p>纳入管理的班级</p>
        </div>
      </article>
      <article>
        <span class="metric-mark green"
          ><el-icon><VideoPlay /></el-icon
        ></span>
        <div>
          <small>进行中考试</small><strong>{{ runningExams.length }}</strong>
          <p>需要持续关注</p>
        </div>
      </article>
      <article>
        <span class="metric-mark amber"
          ><el-icon><Calendar /></el-icon
        ></span>
        <div>
          <small>待开始考试</small><strong>{{ upcomingExams.length }}</strong>
          <p>等待发布或开考</p>
        </div>
      </article>
    </section>

    <div class="overview-grid">
      <section class="overview-card schedule-card">
        <header class="card-heading">
          <div>
            <h2>近期考试</h2>
            <p>优先展示进行中和即将开始的考试。</p>
          </div>
          <el-button text type="primary" @click="router.push('/admin/exams')"
            >考试管理 <el-icon><Right /></el-icon
          ></el-button>
        </header>
        <div v-if="schedule.length" class="schedule-list">
          <button
            v-for="exam in schedule"
            :key="exam.id"
            type="button"
            @click="router.push(`/exams/${exam.id}`)"
          >
            <span class="exam-date"
              ><strong>{{ dayText(exam.starts_at) }}</strong
              ><small>{{ monthText(exam.starts_at) }}</small></span
            >
            <span class="exam-main"
              ><strong>{{ exam.title }}</strong
              ><small>{{ examScope(exam) }}</small></span
            >
            <span class="exam-time">{{ timeText(exam) }}</span>
            <em :class="`status-${examStatus(exam)}`">{{ statusText(exam) }}</em>
          </button>
        </div>
        <el-empty v-else :image-size="72" description="暂无近期考试安排" />
      </section>

      <section class="overview-card pending-card">
        <header class="card-heading">
          <div>
            <h2>待处理</h2>
            <p>
              {{
                auth.role === 'admin'
                  ? '按职责拆分权限申请与题目工单。'
                  : '当前账号可处理的教学事项。'
              }}
            </p>
          </div>
        </header>
        <template v-if="auth.role === 'admin'">
          <button
            class="pending-row"
            type="button"
            @click="router.push('/admin/users/permissions')"
          >
            <span class="pending-icon permission"
              ><el-icon><UserFilled /></el-icon></span
            ><span><strong>出题权限申请</strong><small>在用户与权限中审核申请</small></span
            ><em>{{ pendingApplications.length }}</em
            ><el-icon><Right /></el-icon>
          </button>
          <button class="pending-row" type="button" @click="router.push('/admin/problem-authors')">
            <span class="pending-icon ticket"
              ><el-icon><Document /></el-icon></span
            ><span><strong>题目修改工单</strong><small>只处理题目数据覆盖操作</small></span
            ><em>{{ pendingTickets.length }}</em
            ><el-icon><Right /></el-icon>
          </button>
          <p v-if="!pendingApplications.length && !pendingTickets.length" class="all-clear">
            <el-icon><CircleCheckFilled /></el-icon>当前没有待处理事项
          </p>
        </template>
        <template v-else>
          <button class="pending-row" type="button" @click="router.push('/admin/exams')">
            <span class="pending-icon permission"
              ><el-icon><Calendar /></el-icon></span
            ><span><strong>考试安排</strong><small>查看和维护课程考试</small></span
            ><em>{{ runningExams.length + upcomingExams.length }}</em
            ><el-icon><Right /></el-icon>
          </button>
          <button
            class="pending-row"
            type="button"
            @click="router.push('/admin/prepared-problems')"
          >
            <span class="pending-icon ticket"
              ><el-icon><EditPen /></el-icon></span
            ><span><strong>预备题库</strong><small>整理后续教学题目</small></span
            ><el-icon><Right /></el-icon>
          </button>
        </template>
      </section>
    </div>

    <div class="secondary-grid">
      <section class="overview-card quick-card">
        <header class="card-heading">
          <div>
            <h2>快捷入口</h2>
            <p>直接进入常用管理任务。</p>
          </div>
        </header>
        <div class="quick-links">
          <button
            v-for="item in quickLinks"
            :key="item.path"
            type="button"
            @click="router.push(item.path)"
          >
            <el-icon><component :is="item.icon" /></el-icon
            ><span
              ><strong>{{ item.label }}</strong
              ><small>{{ item.description }}</small></span
            ><el-icon><Right /></el-icon>
          </button>
        </div>
      </section>

      <section class="overview-card activity-card">
        <header class="card-heading">
          <div>
            <h2>{{ auth.role === 'admin' ? '最近动态' : '考试状态' }}</h2>
            <p>
              {{ auth.role === 'admin' ? '系统内最近的管理操作。' : '近期考试的简要状态。' }}
            </p>
          </div>
          <el-button
            v-if="auth.role === 'admin'"
            text
            type="primary"
            @click="router.push('/admin/audit-logs')"
            >审计日志</el-button
          >
        </header>
        <div v-if="activityItems.length" class="activity-list">
          <article v-for="item in activityItems" :key="item.key">
            <i></i
            ><span
              ><strong>{{ item.title }}</strong
              ><small>{{ item.description }}</small></span
            ><time>{{ item.time }}</time>
          </article>
        </div>
        <el-empty v-else :image-size="64" description="暂无管理动态" />
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import {
  Calendar,
  CircleCheckFilled,
  Collection,
  Document,
  EditPen,
  Reading,
  Right,
  User,
  UserFilled,
  VideoPlay
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { client } from '../../api/client'
import { apiErrorMessage } from '../../api/errors'
import { useAuthStore } from '../../stores/auth'

type ExamItem = {
  id: number
  title: string
  course_code?: string
  course_name?: string
  class_name?: string
  starts_at?: string | null
  ends_at?: string | null
}
type AuditItem = {
  id: number
  actor_name?: string
  action?: string
  resource_label?: string
  created_at?: string
}

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const exams = ref<ExamItem[]>([])
const courses = ref<any[]>([])
const classes = ref<any[]>([])
const pendingApplications = ref<any[]>([])
const pendingTickets = ref<any[]>([])
const auditLogs = ref<AuditItem[]>([])
const currentDate = new Intl.DateTimeFormat('zh-CN', {
  year: 'numeric',
  month: 'long',
  day: 'numeric',
  weekday: 'short'
}).format(new Date())

const runningExams = computed(() => exams.value.filter((exam) => examStatus(exam) === 'running'))
const upcomingExams = computed(() => exams.value.filter((exam) => examStatus(exam) === 'upcoming'))
const schedule = computed(() =>
  [...runningExams.value, ...upcomingExams.value]
    .sort((a, b) => dateValue(a.starts_at) - dateValue(b.starts_at))
    .slice(0, 6)
)
const quickLinks = computed(() => [
  {
    path: '/admin/courses',
    label: '课程管理',
    description: '课程与教学范围',
    icon: Reading
  },
  {
    path: '/admin/classes',
    label: '班级管理',
    description: '学生组织与班级',
    icon: User
  },
  {
    path: '/admin/exams',
    label: '考试管理',
    description: '发布与查看考试',
    icon: Collection
  },
  {
    path: auth.role === 'admin' ? '/admin/problem-authors' : '/admin/prepared-problems',
    label: auth.role === 'admin' ? '工单管理' : '预备题库',
    description: auth.role === 'admin' ? '审核题目数据修改' : '整理教学题目',
    icon: auth.role === 'admin' ? Document : EditPen
  }
])
const activityItems = computed(() =>
  auth.role === 'admin'
    ? auditLogs.value.slice(0, 5).map((item) => ({
        key: `audit-${item.id}`,
        title: item.actor_name || '系统',
        description: auditLabel(item),
        time: shortDateTime(item.created_at)
      }))
    : schedule.value.slice(0, 5).map((exam) => ({
        key: `exam-${exam.id}`,
        title: exam.title,
        description: statusText(exam),
        time: shortDateTime(exam.starts_at)
      }))
)

function dateValue(value?: string | null) {
  const time = value ? new Date(value).getTime() : Number.MAX_SAFE_INTEGER
  return Number.isFinite(time) ? time : Number.MAX_SAFE_INTEGER
}
function examStatus(exam: ExamItem): 'running' | 'upcoming' | 'closed' {
  const now = Date.now()
  if (exam.ends_at && dateValue(exam.ends_at) <= now) return 'closed'
  if (exam.starts_at && dateValue(exam.starts_at) > now) return 'upcoming'
  return 'running'
}
function statusText(exam: ExamItem) {
  return examStatus(exam) === 'running'
    ? '进行中'
    : examStatus(exam) === 'upcoming'
      ? '待开始'
      : '已结束'
}
function dayText(value?: string | null) {
  return value ? String(new Date(value).getDate()).padStart(2, '0') : '--'
}
function monthText(value?: string | null) {
  return value ? `${new Date(value).getMonth() + 1}月` : '待定'
}
function examScope(exam: ExamItem) {
  return (
    [exam.course_code, exam.course_name, exam.class_name || '全课程'].filter(Boolean).join(' · ') ||
    '未设置课程范围'
  )
}
function timeText(exam: ExamItem) {
  if (!exam.starts_at) return '时间待定'
  const start = new Date(exam.starts_at).toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit'
  })
  const end = exam.ends_at
    ? new Date(exam.ends_at).toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit'
      })
    : ''
  return end ? `${start}–${end}` : start
}
function shortDateTime(value?: string | null) {
  return value
    ? new Date(value).toLocaleString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      })
    : '-'
}
function auditLabel(item: AuditItem) {
  const labels: Record<string, string> = {
    'exam.create': '创建了考试',
    'exam.delete': '删除了考试',
    'course.create': '创建了课程',
    'course.update': '更新了课程',
    'class.create': '创建了班级',
    'problem_change_ticket.apply': '执行了题目修改工单',
    'author_application.review': '处理了出题权限申请',
    'permission.update': '更新了用户权限'
  }
  return `${labels[item.action || ''] || item.action || '执行了管理操作'}${item.resource_label ? ` · ${item.resource_label}` : ''}`
}

async function load() {
  loading.value = true
  try {
    const [examResponse, courseResponse, classResponse] = await Promise.all([
      client.get('/exams'),
      client.get('/courses'),
      client.get('/classes')
    ])
    exams.value = Array.isArray(examResponse.data) ? examResponse.data : []
    courses.value = Array.isArray(courseResponse.data) ? courseResponse.data : []
    classes.value = Array.isArray(classResponse.data) ? classResponse.data : []
    if (auth.role === 'admin') {
      const [applicationResult, ticketResult, auditResult] = await Promise.allSettled([
        client.get('/author-applications', { params: { status: 'pending' } }),
        client.get('/problem-change-tickets', {
          params: { status: 'pending' }
        }),
        client.get('/audit-logs')
      ])
      pendingApplications.value =
        applicationResult.status === 'fulfilled' && Array.isArray(applicationResult.value.data)
          ? applicationResult.value.data
          : []
      pendingTickets.value =
        ticketResult.status === 'fulfilled' && Array.isArray(ticketResult.value.data)
          ? ticketResult.value.data
          : []
      auditLogs.value =
        auditResult.status === 'fulfilled' && Array.isArray(auditResult.value.data)
          ? auditResult.value.data
          : []
    }
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, '加载教学概览失败'))
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.teaching-overview {
  min-height: calc(100vh - 64px);
  padding: 26px 28px 40px;
  color: #173765;
}
.overview-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 18px;
}
.overview-heading > div:first-child > span {
  color: #135ecb;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.14em;
}
.overview-heading h1 {
  margin: 6px 0 4px;
  font-size: 30px;
}
.overview-heading p {
  margin: 0;
  color: #6f7f96;
}
.heading-actions {
  display: flex;
  align-items: center;
  gap: 9px;
}
.heading-actions time {
  margin-right: 4px;
  color: #7c899c;
  font-size: 12px;
}
.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 14px;
}
.metric-grid article {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
  padding: 17px;
  border: 1px solid #dce4ef;
  border-radius: 7px;
  background: #fff;
}
.metric-mark {
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  flex: 0 0 auto;
  border-radius: 8px;
  font-size: 20px;
}
.metric-mark.blue {
  color: #135ecb;
  background: #eaf2ff;
}
.metric-mark.cyan {
  color: #087d9b;
  background: #e5f7fb;
}
.metric-mark.green {
  color: #08734c;
  background: #e4f7ef;
}
.metric-mark.amber {
  color: #a85b00;
  background: #fff3dd;
}
.metric-grid small,
.metric-grid strong,
.metric-grid p {
  display: block;
}
.metric-grid small {
  color: #66748a;
}
.metric-grid strong {
  margin: 2px 0;
  color: #173765;
  font-size: 26px;
}
.metric-grid p {
  margin: 0;
  color: #98a3b3;
  font-size: 11px;
}
.overview-grid,
.secondary-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(340px, 0.65fr);
  gap: 14px;
}
.secondary-grid {
  grid-template-columns: 1fr 1fr;
  margin-top: 14px;
}
.overview-card {
  min-width: 0;
  overflow: hidden;
  border: 1px solid #dce4ef;
  border-radius: 7px;
  background: #fff;
}
.card-heading {
  display: flex;
  min-height: 70px;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 14px 18px;
  border-bottom: 1px solid #e4eaf2;
}
.card-heading h2 {
  margin: 0 0 4px;
  font-size: 17px;
}
.card-heading p {
  margin: 0;
  color: #7d899a;
  font-size: 12px;
}
.schedule-list button {
  width: 100%;
  display: grid;
  grid-template-columns: 50px minmax(0, 1fr) 110px 70px;
  align-items: center;
  gap: 13px;
  min-height: 72px;
  padding: 10px 16px;
  border: 0;
  border-bottom: 1px solid #e9edf3;
  color: #173765;
  text-align: left;
  background: #fff;
  cursor: pointer;
}
.schedule-list button:hover {
  background: #f6f9fd;
}
.exam-date {
  display: grid;
  place-items: center;
  align-self: stretch;
  align-content: center;
  border-right: 1px solid #e2e8f0;
}
.exam-date strong {
  font-size: 20px;
}
.exam-date small {
  color: #8794a7;
}
.exam-main {
  display: grid;
  gap: 5px;
  min-width: 0;
}
.exam-main strong,
.exam-main small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.exam-main small,
.exam-time {
  color: #6f7f96;
  font-size: 12px;
}
.schedule-list em {
  padding: 4px 7px;
  border-radius: 4px;
  text-align: center;
  font-size: 11px;
  font-style: normal;
  font-weight: 700;
}
.status-running {
  color: #08734c;
  background: #e4f7ef;
}
.status-upcoming {
  color: #a85b00;
  background: #fff3dd;
}
.status-closed {
  color: #6c7685;
  background: #eef1f5;
}
.pending-card {
  align-self: start;
}
.pending-row {
  width: 100%;
  display: grid;
  grid-template-columns: 40px minmax(0, 1fr) auto 18px;
  align-items: center;
  gap: 11px;
  padding: 15px 18px;
  border: 0;
  border-bottom: 1px solid #e9edf3;
  color: #173765;
  text-align: left;
  background: #fff;
  cursor: pointer;
}
.pending-row:hover {
  background: #f6f9fd;
}
.pending-icon {
  width: 38px;
  height: 38px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  font-size: 18px;
}
.pending-icon.permission {
  color: #135ecb;
  background: #eaf2ff;
}
.pending-icon.ticket {
  color: #a85b00;
  background: #fff3dd;
}
.pending-row > span:nth-child(2) {
  display: grid;
  gap: 3px;
}
.pending-row small {
  color: #7c899c;
}
.pending-row em {
  min-width: 26px;
  padding: 4px 7px;
  color: #fff;
  border-radius: 999px;
  background: #c92c2c;
  text-align: center;
  font-style: normal;
  font-weight: 800;
}
.all-clear {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  padding: 16px 18px;
  color: #08734c;
}
.quick-links {
  display: grid;
  grid-template-columns: 1fr 1fr;
}
.quick-links button {
  display: grid;
  grid-template-columns: 30px 1fr 18px;
  align-items: center;
  gap: 10px;
  min-height: 78px;
  padding: 13px 16px;
  border: 0;
  border-right: 1px solid #e9edf3;
  border-bottom: 1px solid #e9edf3;
  color: #173765;
  text-align: left;
  background: #fff;
  cursor: pointer;
}
.quick-links button:hover {
  background: #f6f9fd;
}
.quick-links button > .el-icon:first-child {
  color: #135ecb;
  font-size: 20px;
}
.quick-links span {
  display: grid;
  gap: 4px;
}
.quick-links small {
  color: #7c899c;
}
.activity-list article {
  display: grid;
  grid-template-columns: 9px minmax(0, 1fr) auto;
  align-items: center;
  gap: 11px;
  padding: 12px 18px;
  border-bottom: 1px solid #e9edf3;
}
.activity-list i {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #135ecb;
}
.activity-list span {
  display: grid;
  gap: 3px;
}
.activity-list small,
.activity-list time {
  color: #7c899c;
  font-size: 11px;
}
@media (max-width: 1080px) {
  .metric-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .overview-grid,
  .secondary-grid {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 680px) {
  .teaching-overview {
    padding: 18px 12px 32px;
  }
  .overview-heading {
    align-items: stretch;
    flex-direction: column;
  }
  .heading-actions {
    flex-wrap: wrap;
  }
  .heading-actions time {
    width: 100%;
  }
  .metric-grid {
    grid-template-columns: 1fr;
  }
  .schedule-list button {
    grid-template-columns: 46px 1fr auto;
  }
  .exam-time {
    display: none;
  }
  .quick-links {
    grid-template-columns: 1fr;
  }
}
</style>
