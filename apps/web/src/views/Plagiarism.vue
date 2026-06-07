<template>
  <section class="page">
    <div class="page-header">
      <h2>JPlag 查重</h2>
      <div class="toolbar">
        <el-select v-model="form.course_id" placeholder="课程" class="course-select" @change="form.assignment_id = undefined">
          <el-option v-for="course in courses" :key="course.id" :label="courseLabel(course)" :value="course.id" />
        </el-select>
        <el-select v-model="form.assignment_id" placeholder="作业" clearable class="work-select">
          <el-option v-for="assignment in assignmentOptions" :key="assignment.id" :label="assignment.title" :value="assignment.id" />
        </el-select>
        <el-select v-model="form.language" style="width: 120px">
          <el-option label="C++" value="cpp" />
          <el-option label="C" value="c" />
          <el-option label="Python" value="python" />
          <el-option label="Java" value="java" />
        </el-select>
        <el-button type="primary" @click="create">启动查重</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>
    <div class="panel">
      <el-table :data="jobs">
        <el-table-column label="课程" min-width="160">
          <template #default="{ row }">{{ courseLabel(row) }}</template>
        </el-table-column>
        <el-table-column label="任务" min-width="160">
          <template #default="{ row }">{{ row.assignment_title || row.exam_title || '-' }}</template>
        </el-table-column>
        <el-table-column prop="language" label="语言" width="100" />
        <el-table-column prop="status" label="状态" width="120" />
        <el-table-column prop="message" label="消息" />
        <el-table-column label="报告" width="120">
          <template #default="{ row }">{{ row.report_object ? '已生成' : '-' }}</template>
        </el-table-column>
        <el-table-column label="时间" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
      </el-table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import { client } from '../api/client'
import { formatDateTime } from '../features/time'

const jobs = ref<any[]>([])
const courses = ref<any[]>([])
const assignments = ref<any[]>([])
const form = reactive<{ course_id?: number; assignment_id?: number; language: string }>({ course_id: undefined, assignment_id: undefined, language: 'cpp' })
const assignmentOptions = computed(() => assignments.value.filter((item) => !form.course_id || item.course_id === form.course_id))

async function load() {
  const [jobsRes, coursesRes, assignmentsRes] = await Promise.all([
    client.get('/plagiarism/jobs'),
    client.get('/courses'),
    client.get('/assignments')
  ])
  jobs.value = jobsRes.data
  courses.value = coursesRes.data
  assignments.value = assignmentsRes.data
  if (!form.course_id) {
    form.course_id = courses.value[0]?.id
  }
}

async function create() {
  if (!form.course_id) {
    ElMessage.error('请选择课程')
    return
  }
  await client.post('/plagiarism/jobs', {
    course_id: form.course_id,
    assignment_id: form.assignment_id,
    language: form.language
  })
  ElMessage.success('已启动')
  load()
}

function courseLabel(row: any) {
  return [row.course_code || row.code, row.course_name || row.name].filter(Boolean).join(' ') || '-'
}

onMounted(load)
</script>

<style scoped>
.course-select {
  width: 220px;
}

.work-select {
  width: 220px;
}
</style>
