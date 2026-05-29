<template>
  <section class="page">
    <div class="page-header">
      <h2>考试</h2>
      <div class="toolbar">
        <el-button v-if="canManage" type="primary" @click="openDialog">新建考试</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>
    <div class="panel">
      <el-table :data="items">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="course_id" label="课程" width="100" />
        <el-table-column prop="title" label="标题" />
        <el-table-column prop="starts_at" label="开始" />
        <el-table-column prop="ends_at" label="结束" />
      </el-table>
    </div>
    <el-dialog v-model="dialogVisible" title="新建考试" width="640px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="课程">
          <el-select v-model="form.course_id" style="width: 100%">
            <el-option
              v-for="course in courses"
              :key="course.id"
              :label="`${course.code} ${course.name}`"
              :value="course.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="标题">
          <el-input v-model="form.title" placeholder="期中考试" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="开始时间">
          <el-date-picker v-model="form.starts_at" type="datetime" style="width: 100%" />
        </el-form-item>
        <el-form-item label="结束时间">
          <el-date-picker v-model="form.ends_at" type="datetime" style="width: 100%" />
        </el-form-item>
        <el-form-item label="题目">
          <el-select v-model="form.problem_ids" multiple style="width: 100%">
            <el-option
              v-for="problem in problems"
              :key="problem.id"
              :label="`${problem.id}. ${problem.title}`"
              :value="problem.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">创建</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import { client } from '../api/client'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const canManage = computed(() => auth.role !== 'student')
const items = ref<any[]>([])
const courses = ref<any[]>([])
const problems = ref<any[]>([])
const dialogVisible = ref(false)
const saving = ref(false)
const form = reactive<any>({
  course_id: undefined,
  title: '',
  description: '',
  starts_at: null,
  ends_at: null,
  problem_ids: []
})

async function load() {
  const [examsRes, coursesRes, problemsRes] = await Promise.all([
    client.get('/exams'),
    client.get('/courses'),
    client.get('/problems')
  ])
  items.value = examsRes.data
  courses.value = coursesRes.data
  problems.value = problemsRes.data
}

function openDialog() {
  form.course_id = courses.value[0]?.id
  dialogVisible.value = true
}

async function submit() {
  if (!form.course_id || !form.title || form.problem_ids.length === 0) {
    ElMessage.error('请选择课程、填写标题并选择题目')
    return
  }
  saving.value = true
  try {
    await client.post('/exams', {
      course_id: form.course_id,
      title: form.title,
      description: form.description,
      starts_at: form.starts_at,
      ends_at: form.ends_at,
      problem_ids: form.problem_ids
    })
    ElMessage.success('考试已创建')
    dialogVisible.value = false
    reset()
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    saving.value = false
  }
}

function reset() {
  form.course_id = undefined
  form.title = ''
  form.description = ''
  form.starts_at = null
  form.ends_at = null
  form.problem_ids = []
}

onMounted(load)
</script>
