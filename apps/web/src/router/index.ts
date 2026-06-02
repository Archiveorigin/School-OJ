import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
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

router.beforeEach(async (to) => {
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
