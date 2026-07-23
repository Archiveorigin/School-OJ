<template>
  <section class="course-workspace">
    <div class="course-header">
      <div class="course-heading">
        <el-button text @click="router.push('/my/courses')">← 返回课程列表</el-button>
        <div v-if="course">
          <span>{{ course.code }} · {{ course.term || '学期未设置' }}</span>
          <h1>{{ course.name }}</h1>
          <p>{{ course.college || '所属学院暂未填写' }}</p>
        </div>
        <el-skeleton v-else :rows="2" animated />
      </div>
      <el-button v-if="canManage" plain @click="router.push('/admin/courses')">管理课程</el-button>
    </div>
    <nav class="course-tabs">
      <RouterLink :to="basePath" exact-active-class="active">课程概况</RouterLink>
      <RouterLink :to="`${basePath}/assignments`" active-class="active">作业</RouterLink>
      <RouterLink :to="`${basePath}/exams`" active-class="active">考试</RouterLink>
    </nav>
    <router-view :key="route.fullPath" />
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client } from '../../api/client'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const course = ref<any>(null)
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

watch(courseID, loadCourse)
onMounted(loadCourse)
</script>

<style scoped>
.course-workspace { min-height: calc(100vh - 64px); background: var(--app-bg); }
.course-header { display: flex; align-items: end; justify-content: space-between; gap: 24px; padding: 34px max(28px, calc((100vw - 1180px) / 2)); color: #fff; background: linear-gradient(120deg, #0f172a, #0a5ea6); }
.course-heading { min-width: min(680px, 100%); }
.course-heading :deep(.el-button) { margin: 0 0 18px -14px; color: #bfdbfe; }
.course-heading span, .course-heading p { color: #bfdbfe; }
.course-heading h1 { margin: 6px 0; font-size: 34px; }
.course-heading p { margin: 0; }
.course-tabs { display: flex; gap: 8px; padding: 0 max(28px, calc((100vw - 1180px) / 2)); border-bottom: 1px solid var(--border); background: var(--surface); }
.course-tabs a { padding: 18px 24px 15px; color: var(--muted); border-bottom: 3px solid transparent; }
.course-tabs a.active { color: var(--accent); border-bottom-color: var(--accent); font-weight: 800; }
.course-workspace :deep(.sub-hero) { display: none; }
.course-workspace :deep(.sub-content) { padding-top: 24px; }
@media (max-width: 640px) { .course-header { align-items: stretch; flex-direction: column; padding: 26px 18px; } .course-tabs { padding: 0 8px; overflow-x: auto; } .course-tabs a { white-space: nowrap; } }
</style>
