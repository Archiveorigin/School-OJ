<template>
  <section class="teaching-overview" v-loading="loading">
    <AdminPageHeader title="教学概览" eyebrow="TEACHING OVERVIEW">
      <div class="heading-actions">
        <time>{{ currentDate }}</time
        ><el-button @click="load">刷新</el-button
        ><el-button type="primary" @click="router.push('/admin/exams/new')">新建考试</el-button>
      </div>
    </AdminPageHeader>

    <div class="overview-grid">
      <section class="overview-card schedule-card">
        <AdminSectionHeader title="近期考试">
          <el-button text type="primary" @click="router.push('/admin/exams')"
            >考试管理 <el-icon><Right /></el-icon
          ></el-button>
        </AdminSectionHeader>
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
        <AdminSectionHeader title="待处理" />
        <template v-if="auth.can('user:manage')">
          <button
            class="pending-row"
            type="button"
            @click="router.push('/admin/users/permissions')"
          >
            <span class="pending-icon permission"
              ><el-icon><UserFilled /></el-icon></span
            ><span><strong>出题权限申请</strong></span
            ><em>{{ pendingApplications.length }}</em
            ><el-icon><Right /></el-icon>
          </button>
          <button class="pending-row" type="button" @click="router.push('/admin/problem-authors')">
            <span class="pending-icon ticket"
              ><el-icon><Document /></el-icon></span
            ><span><strong>题目修改工单</strong></span
            ><em>{{ pendingTicketCount }}</em
            ><el-icon><Right /></el-icon>
          </button>
          <p v-if="!pendingApplications.length && !pendingTicketCount" class="all-clear">
            <el-icon><CircleCheckFilled /></el-icon>当前没有待处理事项
          </p>
        </template>
        <template v-else>
          <button class="pending-row" type="button" @click="router.push('/admin/exams')">
            <span class="pending-icon permission"
              ><el-icon><Calendar /></el-icon></span
            ><span><strong>考试安排</strong></span
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
            ><span><strong>预备题库</strong></span
            ><el-icon><Right /></el-icon>
          </button>
        </template>
      </section>
    </div>

    <div class="secondary-grid">
      <section class="overview-card quick-card">
        <AdminSectionHeader title="快捷入口" />
        <div class="quick-links">
          <button
            v-for="item in quickLinks"
            :key="item.path"
            type="button"
            @click="router.push(item.path)"
          >
            <el-icon><component :is="item.icon" /></el-icon
            ><span><strong>{{ item.label }}</strong></span
            ><el-icon><Right /></el-icon>
          </button>
        </div>
      </section>

      <section class="overview-card activity-card">
        <AdminSectionHeader :title="auth.can('user:manage') ? '最近动态' : '考试状态'">
          <el-button
            v-if="auth.can('user:manage')"
            text
            type="primary"
            @click="router.push('/admin/audit-logs')"
            >审计日志</el-button
          >
        </AdminSectionHeader>
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
  UserFilled
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { generatedApi } from '../../api/client'
import { apiErrorMessage } from '../../api/errors'
import AdminPageHeader from '../../components/admin/AdminPageHeader.vue'
import AdminSectionHeader from '../../components/admin/AdminSectionHeader.vue'
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
const pendingApplications = ref<any[]>([])
const pendingTicketCount = ref(0)
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
    icon: Reading
  },
  {
    path: '/admin/classes',
    label: '班级管理',
    icon: User
  },
  {
    path: '/admin/exams',
    label: '考试管理',
    icon: Collection
  },
  {
    path: auth.can('user:manage') ? '/admin/problem-authors' : '/admin/prepared-problems',
    label: auth.can('user:manage') ? '工单管理' : '预备题库',
    icon: auth.can('user:manage') ? Document : EditPen
  }
])
const activityItems = computed(() =>
  auth.can('user:manage')
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
    const examResponse = await generatedApi.listExams<any>()
    exams.value = Array.isArray(examResponse.data) ? examResponse.data : []
    if (auth.can('user:manage')) {
      const [applicationResult, ticketResult, auditResult] = await Promise.allSettled([
        generatedApi.listAuthorApplications<any>({ query: { status: 'pending' } }),
        generatedApi.listProblemChangeTickets<any>({ query: { status: 'pending' } }),
        generatedApi.listAuditLogs<any>()
      ])
      pendingApplications.value =
        applicationResult.status === 'fulfilled' && Array.isArray(applicationResult.value.data)
          ? applicationResult.value.data
          : []
      pendingTicketCount.value = ticketResult.status === 'fulfilled' ? ticketResult.value.data.total || 0 : 0
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
  color: var(--admin-ink);
}
.heading-actions {
  display: flex;
  align-items: center;
  gap: 9px;
}
.heading-actions time {
  margin-right: 4px;
  color: var(--admin-muted);
  font-size: 12px;
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
  border: 1px solid var(--admin-line);
  border-radius: var(--radius-md);
  background: var(--admin-paper);
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
  border-bottom: 1px solid var(--admin-line-soft);
  color: var(--admin-ink);
  text-align: left;
  background: var(--admin-paper);
  cursor: pointer;
}
.schedule-list button:hover {
  background: var(--admin-hover);
}
.exam-date {
  display: grid;
  place-items: center;
  align-self: stretch;
  align-content: center;
  border-right: 1px solid var(--admin-line);
}
.exam-date strong {
  font-size: 20px;
}
.exam-date small {
  color: var(--admin-muted);
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
  color: var(--admin-muted);
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
  color: var(--color-success);
  background: var(--color-success-soft);
}
.status-upcoming {
  color: var(--color-warning);
  background: var(--color-warning-soft);
}
.status-closed {
  color: var(--admin-muted);
  background: var(--color-neutral-soft);
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
  border-bottom: 1px solid var(--admin-line-soft);
  color: var(--admin-ink);
  text-align: left;
  background: var(--admin-paper);
  cursor: pointer;
}
.pending-row:hover {
  background: var(--admin-hover);
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
  color: var(--admin-accent);
  background: var(--color-primary-soft);
}
.pending-icon.ticket {
  color: var(--color-warning);
  background: var(--color-warning-soft);
}
.pending-row em {
  min-width: 26px;
  padding: 4px 7px;
  color: var(--color-on-emphasis);
  border-radius: 999px;
  background: var(--admin-red);
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
  color: var(--color-success);
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
  border-right: 1px solid var(--admin-line-soft);
  border-bottom: 1px solid var(--admin-line-soft);
  color: var(--admin-ink);
  text-align: left;
  background: var(--admin-paper);
  cursor: pointer;
}
.quick-links button:hover {
  background: var(--admin-hover);
}
.quick-links button > .el-icon:first-child {
  color: var(--admin-accent);
  font-size: 20px;
}
.activity-list article {
  display: grid;
  grid-template-columns: 9px minmax(0, 1fr) auto;
  align-items: center;
  gap: 11px;
  padding: 12px 18px;
  border-bottom: 1px solid var(--admin-line-soft);
}
.activity-list i {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--admin-accent);
}
.activity-list span {
  display: grid;
  gap: 3px;
}
.activity-list small,
.activity-list time {
  color: var(--admin-muted);
  font-size: 11px;
}
@media (max-width: 1080px) {
  .overview-grid,
  .secondary-grid {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 680px) {
  .teaching-overview {
    padding: 18px 12px 32px;
  }
  .heading-actions {
    flex-wrap: wrap;
  }
  .heading-actions time {
    width: 100%;
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
