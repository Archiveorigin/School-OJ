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
import Submissions from '../views/Submissions.vue'
import Users from '../views/Users.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: Login },
    { path: '/', component: Dashboard },
    { path: '/courses', component: Courses },
    { path: '/problems', component: Problems },
    { path: '/assignments', component: Assignments },
    { path: '/exams', component: Exams },
    { path: '/submissions', component: Submissions },
    { path: '/leaderboard', component: Leaderboard },
    { path: '/plagiarism', component: Plagiarism },
    { path: '/audit-logs', component: AuditLogs },
    { path: '/users', component: Users }
  ]
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.path !== '/login' && !auth.isAuthed) {
    return '/login'
  }
  if (to.path === '/login' && auth.isAuthed) {
    return '/'
  }
})

export default router
