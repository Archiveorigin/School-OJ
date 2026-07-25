<template>
  <section class="course-workspace">
    <header class="course-header">
      <div class="course-header-inner">
        <div class="workspace-nav">
          <button type="button" class="back-button" @click="router.push('/my/courses')">
            <span aria-hidden="true">←</span> 返回课程列表
          </button>
          <div class="workspace-brand"><img src="/logo1.png" alt="" /><span>黄海在线测试平台</span></div>
        </div>
        <div class="course-title-row">
          <div v-if="course" class="course-heading">
            <span>{{ course.code }} · {{ course.term || '学期未设置' }}</span>
            <h1>{{ course.name }}</h1>
            <p>{{ course.college || '所属学院暂未填写' }}</p>
          </div>
          <el-skeleton v-else :rows="2" animated class="course-heading" />
          <div class="course-functions">
            <span>课程功能</span>
            <el-dropdown trigger="click" placement="bottom-end" @command="handleCourseCommand">
              <button class="qr-menu-button" type="button" aria-label="打开课程扫码功能">
                <img src="/course.jpg" alt="" />
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="qr">扫码加入课程</el-dropdown-item>
                  <el-dropdown-item command="invite">课程邀请码登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-button v-if="canManage" text @click="router.push('/admin/courses')">管理课程</el-button>
          </div>
        </div>
      </div>
    </header>

    <nav class="course-tabs" aria-label="课程功能">
      <RouterLink :to="basePath" exact-active-class="active">课程概况</RouterLink>
      <RouterLink :to="`${basePath}/assignments`" active-class="active">作业</RouterLink>
      <RouterLink :to="`${basePath}/exams`" active-class="active">考试</RouterLink>
    </nav>
    <main class="course-view"><router-view :key="route.fullPath" /></main>

    <el-dialog v-model="qrVisible" title="扫码加入课程" width="420px" align-center class="course-qr-dialog">
      <div class="qr-dialog-content">
        <div class="qr-code-frame">
          <QrcodeVue v-if="qrValue" :value="qrValue" :size="224" level="H" render-as="svg" :margin="2" />
        </div>
        <h3>{{ course?.name || '课程' }}</h3>
        <p>{{ course?.code }} · {{ course?.term || '学期未设置' }}</p>
        <div class="invite-code">
          <span>课程邀请码</span>
          <strong>{{ course?.join_code || '暂未生成' }}</strong>
        </div>
        <small>二维码与本课程邀请码唯一绑定，扫码后将进入课程加入页面。</small>
      </div>
    </el-dialog>

    <el-dialog v-model="inviteVisible" title="课程邀请码登录" width="440px" align-center>
      <div class="invite-dialog-content">
        <p>输入任课教师提供的课程邀请码，也可以使用兼容的班级邀请码。</p>
        <el-input v-model="inviteCode" size="large" clearable placeholder="请输入课程邀请码" @keyup.enter="joinCourse" />
      </div>
      <template #footer>
        <el-button @click="inviteVisible = false">取消</el-button>
        <el-button type="primary" :loading="joining" @click="joinCourse">进入课程</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import QrcodeVue from 'qrcode.vue'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client } from '../../api/client'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const course = ref<any>(null)
const qrVisible = ref(false)
const inviteVisible = ref(false)
const inviteCode = ref('')
const joining = ref(false)
const courseID = computed(() => Number(route.params.courseId))
const basePath = computed(() => `/my/courses/${courseID.value}`)
const canManage = computed(() => auth.role === 'teacher' || auth.role === 'admin')
const qrValue = computed(() => {
  const code = course.value?.join_code || course.value?.code
  if (!code) return ''
  const url = new URL('/my/courses', window.location.origin)
  url.searchParams.set('join_code', code)
  url.searchParams.set('course', String(course.value?.code || courseID.value))
  return url.toString()
})

async function loadCourse() {
  try {
    const items = (await client.get('/courses')).data || []
    course.value = items.find((item: any) => item.id === courseID.value) || null
    if (!course.value) {
      ElMessage.warning('课程不存在或当前账号无权访问')
      router.replace('/my/courses')
    }
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

function handleCourseCommand(command: string) {
  if (command === 'qr') {
    qrVisible.value = true
    return
  }
  if (command === 'invite') {
    inviteCode.value = ''
    inviteVisible.value = true
  }
}

async function joinCourse() {
  const code = inviteCode.value.trim()
  if (!code) {
    ElMessage.warning('请输入课程邀请码')
    return
  }
  joining.value = true
  try {
    try {
      await client.post('/courses/join', { join_code: code })
    } catch (courseError: any) {
      if (courseError.response?.status !== 404) throw courseError
      await client.post('/classes/join', { join_code: code })
    }
    const items = (await client.get('/courses')).data || []
    const joined = items.find((item: any) => item.join_code === code)
    inviteVisible.value = false
    inviteCode.value = ''
    ElMessage.success('课程登录成功')
    if (joined) router.push(`/my/courses/${joined.id}`)
    else router.push('/my/courses')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || '邀请码无效')
  } finally {
    joining.value = false
  }
}

watch(courseID, loadCourse)
onMounted(loadCourse)
</script>

<style scoped>
.course-workspace { min-height: 100vh; background: var(--app-bg); }
.course-header { color: #fff; background: radial-gradient(circle at 82% 22%, rgba(56,189,248,.24), transparent 28%), linear-gradient(120deg, #061a2e, #074f85 66%, #0f766e); }
.course-header-inner { max-width: 1280px; margin: 0 auto; padding: 18px 34px 34px; }
.workspace-nav { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding-bottom: 30px; }
.back-button { display: inline-flex; align-items: center; gap: 8px; padding: 8px 0; color: #bfdbfe; border: 0; background: transparent; cursor: pointer; }
.workspace-brand { display: flex; align-items: center; gap: 9px; color: rgba(255,255,255,.82); font-size: 13px; letter-spacing: .06em; }
.workspace-brand img { width: 30px; height: 30px; padding: 2px; object-fit: contain; border-radius: 8px; background: #fff; }
.course-title-row { display: flex; align-items: end; justify-content: space-between; gap: 28px; }
.course-heading { min-width: min(680px, 100%); }
.course-heading span, .course-heading p { color: #bae6fd; }
.course-heading h1 { margin: 7px 0; font-family: 'Noto Serif SC', 'Songti SC', SimSun, serif; font-size: clamp(30px, 4vw, 44px); }
.course-heading p { margin: 0; }
.course-functions { display: flex; align-items: center; gap: 10px; padding-bottom: 2px; }
.course-functions > span { color: #bae6fd; font-size: 12px; }
.course-functions :deep(.el-button) { color: #dbeafe; }
.qr-menu-button { width: 48px; height: 48px; display: grid; place-items: center; color: #083452; border: 1px solid rgba(255,255,255,.72); border-radius: 13px; background: #fff; box-shadow: 0 12px 30px rgba(0,0,0,.18); cursor: pointer; transition: transform .18s ease, box-shadow .18s ease; }
.qr-menu-button:hover { transform: translateY(-2px); box-shadow: 0 16px 36px rgba(0,0,0,.24); }
.qr-menu-button img { width: 38px; height: 42px; object-fit: cover; border-radius: 6px; }
.course-tabs { position: sticky; top: 0; z-index: 15; display: flex; gap: 8px; padding: 0 max(28px, calc((100vw - 1220px) / 2)); border-bottom: 1px solid var(--border); background: color-mix(in srgb, var(--surface-strong) 92%, transparent); backdrop-filter: blur(16px); }
.course-tabs a { padding: 19px 26px 16px; color: var(--muted); border-bottom: 3px solid transparent; }
.course-tabs a.active { color: var(--accent); border-bottom-color: var(--accent); font-weight: 800; }
.course-view { min-height: calc(100vh - 265px); }
.course-workspace :deep(.sub-hero) { display: none; }
.course-workspace :deep(.sub-content) { padding-top: 24px; }
.qr-dialog-content { display: grid; justify-items: center; text-align: center; }
.qr-code-frame { display: grid; place-items: center; padding: 14px; border: 1px solid var(--border); border-radius: 18px; background: #fff; box-shadow: 0 16px 38px rgba(15, 50, 78, .12); }
.qr-dialog-content h3 { margin: 20px 0 5px; }
.qr-dialog-content p { margin: 0; color: var(--muted); }
.invite-code { min-width: 250px; display: flex; align-items: center; justify-content: space-between; gap: 18px; margin: 18px 0 12px; padding: 13px 16px; border-radius: 10px; background: var(--app-bg); }
.invite-code span, .qr-dialog-content small { color: var(--muted); }
.invite-code strong { color: var(--accent); letter-spacing: .1em; }
.qr-dialog-content small { line-height: 1.6; }
.invite-dialog-content p { margin: 0 0 16px; color: var(--muted); line-height: 1.7; }
@media (max-width: 680px) { .course-header-inner { padding: 15px 18px 28px; } .workspace-brand span { display: none; } .course-title-row { align-items: stretch; flex-direction: column; } .course-functions { justify-content: flex-end; } .course-tabs { padding: 0 7px; overflow-x: auto; } .course-tabs a { padding: 16px 20px 13px; white-space: nowrap; } }
</style>
