import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
const AuditLogs = () => import('../views/AuditLogs.vue')
const AssignmentDetail = () => import('../views/AssignmentDetail.vue')
const Assignments = () => import('../views/Assignments.vue')
const ClassList = () => import('../views/ClassList.vue')
const CourseList = () => import('../views/CourseList.vue')
const Courses = () => import('../views/Courses.vue')
const Dashboard = () => import('../views/Dashboard.vue')
const ExamCreate = () => import('../views/ExamCreate.vue')
const ExamDetail = () => import('../views/ExamDetail.vue')
const ExamRankings = () => import('../views/ExamRankings.vue')
const ExamProblems = () => import('../views/exam/ExamProblems.vue')
const ExamRecords = () => import('../views/exam/ExamRecords.vue')
const ExamSubmit = () => import('../views/exam/ExamSubmit.vue')
const Exams = () => import('../views/Exams.vue')
// leaderboard disabled: const Leaderboard = () => import('../views/Leaderboard.vue')
const CourseStudents = () => import('../views/CourseStudents.vue')
const Login = () => import('../views/Login.vue')
const Plagiarism = () => import('../views/Plagiarism.vue')
const Problems = () => import('../views/Problems.vue')
const PreparedProblems = () => import('../views/PreparedProblems.vue')
const Profile = () => import('../views/Profile.vue')
const Register = () => import('../views/Register.vue')
const ForgotPassword = () => import('../views/ForgotPassword.vue')
const Submissions = () => import('../views/Submissions.vue')
const Users = () => import('../views/Users.vue')

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: Login, meta: { public: true } },
    { path: '/register', component: Register, meta: { public: true } },
    { path: '/forgot-password', component: ForgotPassword, meta: { public: true } },
    { path: '/', component: Dashboard, meta: { title: '概览' } },
    { path: '/profile', component: Profile, meta: { title: 'Profile' } },
    { path: '/courses', component: Courses, meta: { title: '课程班级' } },
    { path: '/courses/list', component: CourseList, meta: { title: '课程列表', activeMenu: '/courses' } },
    { path: '/classes', component: ClassList, meta: { title: '班级列表', activeMenu: '/courses' } },
    { path: '/courses/:id/students', component: CourseStudents, meta: { roles: ['admin', 'teacher'], title: '课程学生', activeMenu: '/courses' } },
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
    // leaderboard disabled: { path: '/leaderboard', component: Leaderboard, meta: { title: '排行榜' } },
    { path: '/admin/exam-rankings', component: ExamRankings, meta: { roles: ['admin', 'teacher'], title: '考试实时榜' } },
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
