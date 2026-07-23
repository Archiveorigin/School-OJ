<template>
  <section class="page course-list-page">
    <div class="course-list-hero">
      <div>
        <span class="eyebrow">PERSONAL COURSES</span>
        <h1>我的课程</h1>
        <p>选择课程进入独立空间，集中查看课程信息、作业与考试。</p>
      </div>
      <div class="course-count"><strong>{{ courses.length }}</strong><span>门课程</span></div>
    </div>

    <div v-if="auth.role === 'student'" class="panel join-panel">
      <div><strong>加入新课程</strong><p class="muted">输入教师提供的课程邀请码或班级邀请码。</p></div>
      <div class="join-actions">
        <el-input v-model="joinCode" placeholder="课程或班级邀请码" clearable @keyup.enter="joinCourse" />
        <el-button type="primary" :loading="joining" @click="joinCourse">加入</el-button>
      </div>
    </div>

    <div class="panel course-table-panel">
      <div class="table-tools">
        <el-input v-model="keyword" clearable placeholder="搜索课程号、课程名、学期或学院" />
        <el-button :loading="loading" @click="load">刷新</el-button>
      </div>
      <el-table :data="filteredCourses" v-loading="loading" @row-click="openCourse">
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
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { client } from '../../api/client'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const courses = ref<any[]>([])
const keyword = ref('')
const joinCode = ref('')
const loading = ref(false)
const joining = ref(false)

const filteredCourses = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) return courses.value
  return courses.value.filter((course) => [course.code, course.name, course.term, course.college].some((value) => String(value || '').toLowerCase().includes(text)))
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

async function joinCourse() {
  const code = joinCode.value.trim()
  if (!code) return
  joining.value = true
  try {
    try {
      await client.post('/courses/join', { join_code: code })
    } catch (courseError: any) {
      if (courseError.response?.status !== 404) throw courseError
      await client.post('/classes/join', { join_code: code })
    }
    joinCode.value = ''
    ElMessage.success('已加入课程')
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || '邀请码无效')
  } finally {
    joining.value = false
  }
}

function openCourse(course: any) {
  router.push(`/my/courses/${course.id}`)
}

onMounted(load)
</script>

<style scoped>
.course-list-page { max-width: 1180px; margin: 0 auto; padding: 36px 28px 68px; }
.course-list-hero { display: flex; align-items: end; justify-content: space-between; gap: 24px; padding: 38px 42px; color: #fff; border-radius: 24px; background: linear-gradient(120deg, #082f49, #0a5ea6 70%, #0f766e); }
.eyebrow { color: #7dd3fc; font-size: 12px; font-weight: 800; letter-spacing: .16em; }
.course-list-hero h1 { margin: 10px 0 6px; font-size: 38px; }
.course-list-hero p { margin: 0; color: #dbeafe; }
.course-count { display: grid; min-width: 110px; text-align: right; }
.course-count strong { font-size: 44px; line-height: 1; }
.course-count span { color: #dbeafe; }
.join-panel { display: flex; align-items: center; justify-content: space-between; gap: 24px; margin-top: 20px; padding: 22px 26px; }
.join-panel p { margin: 6px 0 0; }
.join-actions, .table-tools { display: flex; gap: 10px; }
.join-actions { width: min(460px, 100%); }
.course-table-panel { margin-top: 20px; padding: 22px; }
.table-tools { justify-content: flex-end; margin-bottom: 18px; }
.table-tools .el-input { width: min(420px, 100%); }
.course-table-panel :deep(.el-table__row) { cursor: pointer; }
@media (max-width: 680px) { .course-list-page { padding: 20px 14px 48px; } .course-list-hero, .join-panel { align-items: stretch; flex-direction: column; } .course-count { text-align: left; } .join-actions, .table-tools { width: 100%; } }
</style>
