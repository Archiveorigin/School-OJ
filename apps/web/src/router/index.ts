import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import AuditLogs from '../views/AuditLogs.vue'
import AssignmentDetail from '../views/AssignmentDetail.vue'
import Assignments from '../views/Assignments.vue'
import Courses from '../views/Courses.vue'
import Dashboard from '../views/Dashboard.vue'
import ExamCreate from '../views/ExamCreate.vue'
import ExamDetail from '../views/ExamDetail.vue'
import ExamProblems from '../views/exam/ExamProblems.vue'
import ExamRecords from '../views/exam/ExamRecords.vue'
import ExamSubmit from '../views/exam/ExamSubmit.vue'
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
    { path: '/login', component: Login, meta: { public: true } },
    { path: '/register', component: Register, meta: { public: true } },
    { path: '/forgot-password', component: ForgotPassword, meta: { public: true } },
    { path: '/', component: Dashboard, meta: { title: '概览' } },
    { path: '/profile', component: Profile, meta: { title: 'Profile' } },
    { path: '/courses', component: Courses, meta: { title: '课程班级' } },
    { path: '/problems', component: Problems, meta: { title: '题库' } },
    { path: '/prepared-problems', component: PreparedProblems, meta: { roles: ['admin', 'teacher'], title: '预备题库' } },
    { path: '/assignments', component: Assignments, meta: { title: '作业' } },
    { path: '/assignments/:id', component: AssignmentDetail, meta: { title: '作业', activeMenu: '/assignments' } },
    { path: '/exams', component: Exams, meta: { title: '考试' } },
    { path: '/exams/new', component: ExamCreate, meta: { roles: ['admin', 'teacher'], title: '新建考试', activeMenu: '/exams' } },
    {
      path: '/exams/:id',
      component: ExamDetail,
      redirect: (to) => `/exams/${to.params.id}/problems`,
      meta: { title: '考试', activeMenu: '/exams' },
      children: [
        { path: 'problems', component: ExamProblems, meta: { title: '考试', activeMenu: '/exams' } },
        { path: 'submit', component: ExamSubmit, meta: { title: '考试', activeMenu: '/exams' } },
        { path: 'records', component: ExamRecords, meta: { title: '考试', activeMenu: '/exams' } }
      ]
    },
    { path: '/submissions', component: Submissions, meta: { title: '提交' } },
    { path: '/leaderboard', component: Leaderboard, meta: { title: '排行榜' } },
    { path: '/plagiarism', component: Plagiarism, meta: { roles: ['admin', 'teacher'], title: 'JPlag 查重' } },
    { path: '/audit-logs', component: AuditLogs, meta: { roles: ['admin'], title: '审计日志' } },
    { path: '/users', component: Users, meta: { roles: ['admin'], title: '用户管理' } }
  ]
})

const publicPaths = ['/login', '/register', '/forgot-password']

router.beforeEach(async (to, from) => {
  const auth = useAuthStore()
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
})

export default router
