import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const AdminHome = () => import('../views/admin/AdminHome.vue')
const AdminLayout = () => import('../views/admin/AdminLayout.vue')
const AssignmentDetail = () => import('../views/AssignmentDetail.vue')
const Assignments = () => import('../views/Assignments.vue')
const AuditLogs = () => import('../views/AuditLogs.vue')
const ClassList = () => import('../views/ClassList.vue')
const CourseList = () => import('../views/CourseList.vue')
const CourseOverview = () => import('../views/courses/CourseOverview.vue')
const CourseWorkspace = () => import('../views/courses/CourseWorkspace.vue')
const CourseStudents = () => import('../views/CourseStudents.vue')
const Dashboard = () => import('../views/Dashboard.vue')
const ExamCreate = () => import('../views/ExamCreate.vue')
const ExamDetail = () => import('../views/ExamDetail.vue')
const ExamProblems = () => import('../views/exam/ExamProblems.vue')
const ExamOverview = () => import('../views/exam/ExamOverview.vue')
const ExamRanking = () => import('../views/exam/ExamRanking.vue')
const ExamRecords = () => import('../views/exam/ExamRecords.vue')
const ExamSubmit = () => import('../views/exam/ExamSubmit.vue')
const Exams = () => import('../views/Exams.vue')
const ForgotPassword = () => import('../views/ForgotPassword.vue')
const Login = () => import('../views/Login.vue')
const MyCourses = () => import('../views/courses/MyCourses.vue')
const Plagiarism = () => import('../views/Plagiarism.vue')
const ProblemDetail = () => import('../views/problems/ProblemDetail.vue')
const ProblemCreate = () => import('../views/problems/ProblemCreate.vue')
const ProblemAuthorManagement = () => import('../views/admin/ProblemAuthorManagement.vue')
const ProblemSubmissions = () => import('../views/problems/ProblemSubmissions.vue')
const Problems = () => import('../views/Problems.vue')
const ProblemsLayout = () => import('../views/problems/ProblemsLayout.vue')
const PreparedProblems = () => import('../views/PreparedProblems.vue')
const Profile = () => import('../views/Profile.vue')
const Register = () => import('../views/Register.vue')
const Submissions = () => import('../views/Submissions.vue')
const TeamContests = () => import('../views/teams/TeamContests.vue')
const TeamContestDetail = () => import('../views/teams/TeamContestDetail.vue')
const TeamList = () => import('../views/teams/TeamList.vue')
const TeamMembers = () => import('../views/teams/TeamMembers.vue')
const TeamProblemSetDetail = () => import('../views/teams/TeamProblemSetDetail.vue')
const TeamProblemSets = () => import('../views/teams/TeamProblemSets.vue')
const TeamWorkspace = () => import('../views/teams/TeamWorkspace.vue')
const Users = () => import('../views/Users.vue')

const teacherRoles = ['admin', 'teacher']

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: Login, meta: { public: true } },
    { path: '/register', component: Register, meta: { public: true } },
    { path: '/forgot-password', component: ForgotPassword, meta: { public: true } },
    { path: '/', component: Dashboard, meta: { public: true, title: '概览', activeMenu: '/' } },
    {
      path: '/problems',
      component: ProblemsLayout,
      meta: { public: true, title: '题库', activeMenu: '/problems' },
      children: [
        { path: '', name: 'problem-list', component: Problems },
        { path: 'create', name: 'problem-create', component: ProblemCreate, meta: { public: false, requiresAuthor: true, title: '创建题目', activeMenu: '/problems/create' } },
        { path: ':id/submissions', name: 'problem-submissions', component: ProblemSubmissions, meta: { public: false, title: '题目提交记录' } },
        { path: ':id', name: 'problem-detail', component: ProblemDetail, meta: { title: '题目详情' } }
      ]
    },
    { path: '/profile', component: Profile, meta: { title: '个人中心' } },
    { path: '/teams', component: TeamList, meta: { title: '团队', activeMenu: '/teams' } },
    { path: '/teams/:teamId/problem-sets/:setId', component: TeamProblemSetDetail, meta: { title: '团队题单', activeMenu: '/teams' } },
    {
      path: '/teams/:teamSlug',
      component: TeamWorkspace,
      redirect: (to) => `/teams/${to.params.teamSlug}/contests`,
      meta: { title: '团队空间', activeMenu: '/teams' },
      children: [
        { path: 'contests', component: TeamContests, meta: { title: '团队比赛' } },
        { path: 'contests/:contestId', component: TeamContestDetail, meta: { title: '团队比赛详情' } },
        { path: 'problem-sets', component: TeamProblemSets, meta: { title: '团队题单' } },
        { path: 'members', component: TeamMembers, meta: { title: '团队成员' } }
      ]
    },
    { path: '/my/courses', component: MyCourses, meta: { title: '我的课程', activeMenu: '/my/courses' } },
    {
      path: '/my/courses/:courseId',
      component: CourseWorkspace,
      meta: { title: '课程空间', activeMenu: '/my/courses' },
      children: [
        { path: '', component: CourseOverview, meta: { title: '课程概况' } },
        { path: 'assignments', component: Assignments, meta: { title: '课程作业' } },
        { path: 'exams', component: Exams, meta: { title: '课程考试' } }
      ]
    },
    { path: '/courses', redirect: '/my/courses' },
    { path: '/courses/list', redirect: '/admin/courses' },
    { path: '/classes', redirect: '/admin/classes' },
    { path: '/assignments', redirect: '/my/courses' },
    { path: '/assignments/:id', component: AssignmentDetail, meta: { title: '作业' } },
    { path: '/exams', redirect: '/my/courses' },
    { path: '/exams/new', component: ExamCreate, meta: { roles: teacherRoles, title: '新建考试' } },
    {
      path: '/exams/:id',
      component: ExamDetail,
      redirect: (to) => `/exams/${to.params.id}/overview`,
      meta: { title: '考试' },
      children: [
        { path: 'overview', component: ExamOverview, meta: { title: '题目概览' } },
        { path: 'problems', component: ExamProblems, meta: { title: '考试题目' } },
        { path: 'submit', component: ExamSubmit, meta: { title: '提交代码' } },
        { path: 'records', component: ExamRecords, meta: { title: '提交记录' } },
        { path: 'ranking', component: ExamRanking, meta: { title: '实时榜单' } }
      ]
    },
    { path: '/submissions', component: Submissions, meta: { title: '我的提交' } },
    {
      path: '/admin',
      component: AdminLayout,
      meta: { roles: teacherRoles, title: '后台管理' },
      children: [
        { path: '', component: AdminHome, meta: { title: '管理概览', adminMenu: '/admin' } },
        { path: 'courses', component: CourseList, meta: { title: '课程管理', adminMenu: '/admin/courses' } },
        { path: 'courses/:id/students', component: CourseStudents, meta: { title: '课程学生', adminMenu: '/admin/courses' } },
        { path: 'classes', component: ClassList, meta: { title: '班级管理', adminMenu: '/admin/classes' } },
        { path: 'prepared-problems', component: PreparedProblems, meta: { title: '预备题库', adminMenu: '/admin/prepared-problems' } },
        { path: 'problem-authors', component: ProblemAuthorManagement, meta: { roles: ['admin'], title: '出题管理', adminMenu: '/admin/problem-authors' } },
        { path: 'plagiarism', component: Plagiarism, meta: { title: 'JPlag 查重', adminMenu: '/admin/plagiarism' } },
        { path: 'audit-logs', component: AuditLogs, meta: { roles: ['admin'], title: '审计日志', adminMenu: '/admin/audit-logs' } },
        { path: 'users', component: Users, meta: { roles: ['admin'], title: '用户管理', adminMenu: '/admin/users' } }
      ]
    },
    { path: '/prepared-problems', redirect: '/admin/prepared-problems' },
    { path: '/plagiarism', redirect: '/admin/plagiarism' },
    { path: '/audit-logs', redirect: '/admin/audit-logs' },
    { path: '/users', redirect: '/admin/users' },
    { path: '/problem-authors', redirect: '/admin/problem-authors' },
    { path: '/admin/exam-rankings', redirect: '/my/courses' }
  ]
})

const authPaths = ['/login', '/register', '/forgot-password']

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (auth.isAuthed && !auth.hydrated) await auth.hydrate()
  if (!to.meta.public && !auth.isAuthed) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (authPaths.includes(to.path) && auth.isAuthed) return '/'
  const roles = to.meta.roles as string[] | undefined
  if (roles && (!auth.user || !roles.includes(auth.user.role))) return '/'
  if (to.meta.requiresAuthor && (!auth.user || !(auth.user.can_author || auth.user.role === 'admin'))) return '/'
})

export default router
