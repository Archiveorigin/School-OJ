<template>
  <section class="page course-list-page">
    <div class="course-list-hero">
      <div>
        <span class="eyebrow">PERSONAL COURSES</span>
        <h1>我的课程</h1>
        <p>选择课程进入独立空间，集中查看课程信息、作业与考试。</p>
      </div>
      <div class="hero-side">
        <div class="course-count"><strong>{{ courses.length }}</strong><span>门课程</span></div>
        <div v-if="auth.role === 'student'" class="join-shortcuts">
          <img src="/course.jpg" alt="加入新课程" />
          <div class="join-shortcut-copy">
            <strong>加入新课程</strong>
            <div class="join-shortcut-actions">
              <el-button class="scan-button" @click="joinDialogs?.openScanner()">扫码加入</el-button>
              <el-button class="invite-button" @click="joinDialogs?.openInvite()">邀请码加入</el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="panel course-table-panel">
      <div class="table-tools">
        <el-input v-model="keyword" clearable placeholder="搜索课程号、课程名、学期或学院" />
        <el-button :loading="loading" @click="load">刷新</el-button>
      </div>
      <el-table :data="pagedCourses" v-loading="loading" @row-click="openCourse">
        <el-table-column prop="code" label="课程号" width="160" />
        <el-table-column prop="name" label="课程名" min-width="220" />
        <el-table-column prop="term" label="学期" width="150" />
        <el-table-column label="所属学院" min-width="190">
          <template #default="{ row }">{{ row.college || '暂未填写' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="110" align="right">
          <template #default="{ row }"><el-button size="small" type="primary" @click.stop="openCourse(row)">进入</el-button></template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && !filteredCourses.length" description="暂无课程" />
      <ListPagination v-model:page="page" v-model:page-size="pageSize" :total="filteredCourses.length" />
    </div>

    <CourseJoinDialogs ref="joinDialogs" @joined="handleJoined" />
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client } from '../../api/client'
import CourseJoinDialogs from '../../components/CourseJoinDialogs.vue'
import ListPagination from '../../components/ListPagination.vue'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const courses = ref<any[]>([])
const keyword = ref('')
const loading = ref(false)
const page = ref(1)
const pageSize = ref(10)
const joinDialogs = ref<InstanceType<typeof CourseJoinDialogs> | null>(null)

const filteredCourses = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) return courses.value
  return courses.value.filter((course) => [course.code, course.name, course.term, course.college].some((value) => String(value || '').toLowerCase().includes(text)))
})
const pagedCourses = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredCourses.value.slice(start, start + pageSize.value)
})

async function load() {
  loading.value = true
  try {
    courses.value = (await client.get('/courses')).data || []
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

async function handleJoined() {
  await load()
  if (route.query.join_code) {
    const query = { ...route.query }
    delete query.join_code
    delete query.course
    await router.replace({ query })
  }
}

function openCourse(course: any) {
  router.push(`/my/courses/${course.id}`)
}

watch([keyword, pageSize], () => { page.value = 1 })
onMounted(() => {
  const queryCode = typeof route.query.join_code === 'string' ? route.query.join_code : ''
  if (queryCode) joinDialogs.value?.openInvite(queryCode)
  void load()
})
</script>

<style scoped>
.course-list-page { max-width: 1180px; margin: 0 auto; padding: 36px 28px 68px; }
.course-list-hero { display: flex; align-items: end; justify-content: space-between; gap: 24px; padding: 38px 42px; color: #fff; border-radius: 24px; background: linear-gradient(120deg, #082f49, #0a5ea6 70%, #0f766e); }
.eyebrow { color: #7dd3fc; font-size: 12px; font-weight: 800; letter-spacing: .16em; }
.course-list-hero h1 { margin: 10px 0 6px; font-size: 38px; }
.course-list-hero p { margin: 0; color: #dbeafe; }
.hero-side { display: flex; align-items: center; gap: 22px; }
.course-count { display: grid; min-width: 110px; text-align: right; }
.course-count strong { font-size: 44px; line-height: 1; }
.course-count span { color: #dbeafe; }
.join-shortcuts { display: flex; align-items: center; gap: 13px; padding: 10px 12px; border: 1px solid rgba(255,255,255,.28); border-radius: 16px; background: rgba(255,255,255,.1); backdrop-filter: blur(10px); }
.join-shortcuts img { width: 54px; height: 61px; object-fit: cover; border-radius: 9px; background: #fff; }
.join-shortcut-copy { display: grid; gap: 8px; }
.join-shortcut-actions, .table-tools { display: flex; gap: 8px; }
.join-shortcut-actions :deep(.el-button) { margin: 0; color: #fff; border-color: rgba(255,255,255,.54); background: rgba(255,255,255,.12); }
.join-shortcut-actions :deep(.el-button:hover) { color: #083452; background: #fff; }
.join-shortcut-actions .invite-button { border-color: #7dd3fc; background: rgba(14,165,233,.32); }
.course-table-panel { margin-top: 20px; padding: 22px; }
.table-tools { justify-content: flex-end; margin-bottom: 18px; }
.table-tools .el-input { width: min(420px, 100%); }
.course-table-panel :deep(.el-table__row) { cursor: pointer; }
@media (max-width: 860px) { .course-list-hero { align-items: stretch; flex-direction: column; } .hero-side { justify-content: space-between; } }
@media (max-width: 680px) { .course-list-page { padding: 20px 14px 48px; } .hero-side { align-items: stretch; flex-direction: column; } .course-count { text-align: left; } .join-shortcuts { align-items: stretch; } .join-shortcuts img { width: 48px; height: 54px; } .join-shortcut-actions, .table-tools { width: 100%; } .join-shortcut-actions { flex-wrap: wrap; } }
</style>
