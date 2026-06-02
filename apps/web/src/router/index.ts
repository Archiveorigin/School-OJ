import { createRouter, createWebHistory } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import { useExamLockStore } from '../stores/examLock'
import AuditLogs from '../views/AuditLogs.vue'
import Assignments from '../views/Assignments.vue'
import Courses from '../views/Courses.vue'
import Dashboard from '../views/Dashboard.vue'
import Exams from '../views/Exams.vue'
import Leaderboard from '../views/Leaderboard.vue'
import Login from '../views/Login.vue'
import Plagiarism from '../views/Plagiarism.vue'
import Problems from '../views/Problems.vue'
import PreparedProblems from '../views/PreparedProblems.vue'
import Profile from '../views/Profile.vue'
import Register from '../views/Register.vue'
import ForgotPassword from '../views/ForgotPassword.vue'
import Submissions from '../views/Submissions.vue'
import Users from '../views/Users.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: Login },
    { path: '/register', component: Register },
    { path: '/forgot-password', component: ForgotPassword },
    { path: '/', component: Dashboard },
    { path: '/profile', component: Profile },
    { path: '/courses', component: Courses },
    { path: '/problems', component: Problems },
    { path: '/prepared-problems', component: PreparedProblems, meta: { roles: ['admin', 'teacher'] } },
    { path: '/assignments', component: Assignments },
    { path: '/exams', component: Exams },
    { path: '/submissions', component: Submissions },
    { path: '/leaderboard', component: Leaderboard },
    { path: '/plagiarism', component: Plagiarism, meta: { roles: ['admin', 'teacher'] } },
    { path: '/audit-logs', component: AuditLogs, meta: { roles: ['admin'] } },
    { path: '/users', component: Users, meta: { roles: ['admin'] } }
  ]
})

const publicPaths = ['/login', '/register', '/forgot-password']

router.beforeEach(async (to, from) => {
  const auth = useAuthStore()
  const examLock = useExamLockStore()
  examLock.hydrate()
  if (auth.isAuthed && !auth.hydrated) {
    await auth.hydrate()
  }
  if (!publicPaths.includes(to.path) && !auth.isAuthed) {
    return '/login'
  }
  if (publicPaths.includes(to.path) && auth.isAuthed) {
    return '/'
  }
  const roles = to.meta.roles as string[] | undefined
  if (roles && (!auth.user || !roles.includes(auth.user.role))) {
    return '/'
  }
  if (examLock.locked && to.path !== '/exams') {
    ElMessage.warning(examLock.message)
    return { path: '/exams', query: examLock.examId ? { locked_exam_id: String(examLock.examId) } : undefined }
  }
  if (examLock.locked && to.path === '/exams' && examLock.examId && to.query.locked_exam_id !== String(examLock.examId)) {
    return { path: '/exams', query: { ...to.query, locked_exam_id: String(examLock.examId) } }
  }
})

export default router
