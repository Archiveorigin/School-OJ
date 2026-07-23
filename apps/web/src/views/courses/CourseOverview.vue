<template>
  <section class="course-overview">
    <div class="stat-grid">
      <div class="panel stat-card"><strong>{{ classes.length }}</strong><span>相关班级</span></div>
      <div class="panel stat-card"><strong>{{ assignments.length }}</strong><span>课程作业</span></div>
      <div class="panel stat-card"><strong>{{ exams.length }}</strong><span>课程考试</span></div>
    </div>
    <div class="overview-columns">
      <div class="panel content-panel">
        <div class="section-title"><h3>班级信息</h3><el-button v-if="canManage" text type="primary" @click="router.push('/admin/classes')">管理班级</el-button></div>
        <el-table :data="classes" v-loading="loading">
          <el-table-column prop="name" label="班级" min-width="180" />
          <el-table-column prop="term" label="学期" width="140" />
          <el-table-column v-if="canManage" prop="join_code" label="邀请码" width="150" />
        </el-table>
        <el-empty v-if="!loading && !classes.length" description="暂无班级信息" />
      </div>
      <div class="panel content-panel quick-panel">
        <div class="section-title"><h3>快速进入</h3></div>
        <button type="button" @click="router.push(`${basePath}/assignments`)"><strong>课程作业</strong><span>查看任务、截止时间和完成情况 →</span></button>
        <button type="button" @click="router.push(`${basePath}/exams`)"><strong>课程考试</strong><span>进入考试、提交记录和实时榜单 →</span></button>
        <button type="button" @click="router.push('/problems')"><strong>公共题库</strong><span>浏览全站公开题目 →</span></button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client } from '../../api/client'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const courseID = computed(() => Number(route.params.courseId))
const basePath = computed(() => `/my/courses/${courseID.value}`)
const canManage = computed(() => auth.role === 'teacher' || auth.role === 'admin')
const classes = ref<any[]>([])
const assignments = ref<any[]>([])
const exams = ref<any[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const [classRes, assignmentRes, examRes] = await Promise.all([
      client.get('/classes', { params: { course_id: courseID.value } }),
      client.get('/assignments', { params: { course_id: courseID.value } }),
      client.get('/exams', { params: { course_id: courseID.value } })
    ])
    classes.value = classRes.data || []
    assignments.value = assignmentRes.data || []
    exams.value = examRes.data || []
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.course-overview { max-width: 1180px; margin: 0 auto; padding: 28px; }
.stat-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
.stat-card { display: grid; gap: 6px; padding: 24px; }
.stat-card strong { color: var(--accent); font-size: 34px; }
.stat-card span { color: var(--muted); }
.overview-columns { display: grid; grid-template-columns: minmax(0, 1.45fr) minmax(280px, .7fr); gap: 18px; margin-top: 18px; }
.content-panel { padding: 24px; }
.quick-panel { display: flex; flex-direction: column; gap: 10px; }
.quick-panel button { display: grid; gap: 6px; padding: 18px; text-align: left; color: var(--text); border: 1px solid var(--border); border-radius: 12px; background: var(--surface-strong); cursor: pointer; }
.quick-panel button:hover { border-color: var(--accent); }
.quick-panel span { color: var(--muted); }
@media (max-width: 800px) { .stat-grid, .overview-columns { grid-template-columns: 1fr; } .course-overview { padding: 18px 14px; } }
</style>
