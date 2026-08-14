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
            <button
              class="course-add-button"
              type="button"
              aria-label="输入课程邀请码"
              title="输入课程邀请码"
              @click="joinDialogs?.openInvite()"
            >
              <img :src="courseAddIcon" alt="" />
            </button>
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

    <CourseJoinDialogs ref="joinDialogs" @joined="handleJoined" />
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client } from '../../api/client'
import courseAddIcon from '../../assets/course-add.svg'
import CourseJoinDialogs from '../../components/CourseJoinDialogs.vue'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const course = ref<any>(null)
const joinDialogs = ref<InstanceType<typeof CourseJoinDialogs> | null>(null)
const courseID = computed(() => Number(route.params.courseId))
const basePath = computed(() => `/my/courses/${courseID.value}`)
const canManage = computed(() => auth.role === 'teacher' || auth.role === 'admin')

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

function handleJoined(joinedCourseID: number) {
  if (Number.isInteger(joinedCourseID) && joinedCourseID > 0) {
    router.push(`/my/courses/${joinedCourseID}`)
    return
  }
  router.push('/my/courses')
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
.course-add-button { width: 48px; height: 48px; display: grid; place-items: center; color: #083452; border: 1px solid rgba(255,255,255,.72); border-radius: 13px; background: #fff; box-shadow: 0 12px 30px rgba(0,0,0,.18); cursor: pointer; transition: transform .18s ease, box-shadow .18s ease; }
.course-add-button:hover { transform: translateY(-2px); box-shadow: 0 16px 36px rgba(0,0,0,.24); }
.course-add-button:focus-visible { outline: 3px solid #7dd3fc; outline-offset: 3px; }
.course-add-button img { width: 34px; height: 34px; object-fit: contain; }
.course-tabs { position: sticky; top: 0; z-index: 15; display: flex; gap: 8px; padding: 0 max(28px, calc((100vw - 1220px) / 2)); border-bottom: 1px solid var(--border); background: color-mix(in srgb, var(--surface-strong) 92%, transparent); backdrop-filter: blur(16px); }
.course-tabs a { padding: 19px 26px 16px; color: var(--muted); border-bottom: 3px solid transparent; }
.course-tabs a.active { color: var(--accent); border-bottom-color: var(--accent); font-weight: 800; }
.course-view { min-height: calc(100vh - 265px); }
.course-workspace :deep(.sub-hero) { display: none; }
.course-workspace :deep(.sub-content) { padding-top: 24px; }
@media (max-width: 680px) { .course-header-inner { padding: 15px 18px 28px; } .workspace-brand span { display: none; } .course-title-row { align-items: stretch; flex-direction: column; } .course-functions { justify-content: flex-end; } .course-tabs { padding: 0 7px; overflow-x: auto; } .course-tabs a { padding: 16px 20px 13px; white-space: nowrap; } }
</style>
