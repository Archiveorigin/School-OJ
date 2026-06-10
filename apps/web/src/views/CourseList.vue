<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h2>{{ canManage ? '课程列表' : '我的课程' }}</h2>
        <p class="muted">{{ canManage ? '管理员和教师可查看所有可管理课程。' : '这里仅显示你所属班级对应的课程。' }}</p>
      </div>
      <div class="toolbar">
        <el-button v-if="canManage" type="primary" @click="openCourseDialog">新建课程</el-button>
        <el-button @click="router.push('/courses')">返回入口</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>

    <div class="panel">
      <el-table :data="pagedCourses" v-loading="loading">
        <el-table-column v-if="canManage" prop="code" label="课程代码" width="150" />
        <el-table-column prop="name" label="课程名称" min-width="180" />
        <el-table-column prop="term" label="学期" width="140" />
        <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip />
        <el-table-column label="班级" width="100">
          <template #default="{ row }">{{ courseClassCount(row.id) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="220">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="openClasses(row)">查看班级</el-button>
            <el-button v-if="canManage" size="small" @click="openClassDialog(row.id)">加班级</el-button>
          </template>
        </el-table-column>
      </el-table>
      <ListPagination v-model:page="page" v-model:page-size="pageSize" :total="courses.length" />
    </div>

    <el-dialog v-model="courseDialogVisible" title="新建课程" width="560px">
      <el-form :model="courseForm" label-width="90px">
        <el-form-item label="课程代码">
          <el-input v-model="courseForm.code" placeholder="CS101-2026" />
        </el-form-item>
        <el-form-item label="课程名称">
          <el-input v-model="courseForm.name" placeholder="程序设计基础" />
        </el-form-item>
        <el-form-item label="学期">
          <el-input v-model="courseForm.term" placeholder="2026 春" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="courseForm.description" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="courseDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingCourse" @click="submitCourse">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="classDialogVisible" title="新建班级" width="520px">
      <el-form :model="classForm" label-width="90px">
        <el-form-item label="课程">
          <el-select v-model="classForm.course_id" style="width: 100%">
            <el-option v-for="course in courses" :key="course.id" :label="courseOptionLabel(course)" :value="course.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="班级名称">
          <el-input v-model="classForm.name" placeholder="计科一班" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="classDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingClass" @click="submitClass">创建</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { client } from '../api/client'
import ListPagination from '../components/ListPagination.vue'
import { useAuthStore } from '../stores/auth'
import { useClassroomStore } from '../stores/classroom'

type Course = {
  id: number
  code?: string
  name: string
  term?: string
  description?: string
}

const auth = useAuthStore()
const classroom = useClassroomStore()
const router = useRouter()
const canManage = computed(() => auth.role === 'admin' || auth.role === 'teacher')
const loading = ref(false)
const courses = ref<Course[]>([])
const page = ref(1)
const pageSize = ref(10)
const courseDialogVisible = ref(false)
const classDialogVisible = ref(false)
const savingCourse = ref(false)
const savingClass = ref(false)
const courseForm = reactive({ code: '', name: '', term: '2026 春', description: '' })
const classForm = reactive({ course_id: 0, name: '' })
const pagedCourses = computed(() => courses.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))

function list<T>(value: T[] | null | undefined) {
  return Array.isArray(value) ? value : []
}

async function load() {
  loading.value = true
  try {
    const [coursesRes] = await Promise.all([client.get('/courses'), classroom.load({ force: true })])
    courses.value = list(coursesRes.data)
    clampPage()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

function courseClassCount(courseID: number) {
  return classroom.classes.filter((item) => item.course_id === courseID).length
}

function courseOptionLabel(course: Course) {
  return course.code ? `${course.code} ${course.name}` : course.name
}

function openClasses(course: Course) {
  router.push({ path: '/classes', query: { course_id: course.id } })
}

function openCourseDialog() {
  courseDialogVisible.value = true
}

function openClassDialog(courseID: number) {
  classForm.course_id = courseID
  classForm.name = ''
  classDialogVisible.value = true
}

async function submitCourse() {
  if (!courseForm.code || !courseForm.name) {
    ElMessage.error('请填写课程代码和课程名称')
    return
  }
  savingCourse.value = true
  try {
    await client.post('/courses', { ...courseForm })
    ElMessage.success('课程已创建')
    courseDialogVisible.value = false
    courseForm.code = ''
    courseForm.name = ''
    courseForm.term = '2026 春'
    courseForm.description = ''
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    savingCourse.value = false
  }
}

async function submitClass() {
  if (!classForm.course_id || !classForm.name) {
    ElMessage.error('请选择课程并填写班级名称')
    return
  }
  savingClass.value = true
  try {
    await client.post(`/courses/${classForm.course_id}/classes`, { name: classForm.name })
    ElMessage.success('班级已创建')
    classDialogVisible.value = false
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    savingClass.value = false
  }
}

watch(pageSize, clampPage)

onMounted(load)

function clampPage() {
  const maxPage = Math.max(1, Math.ceil(courses.value.length / pageSize.value))
  if (page.value > maxPage) page.value = maxPage
  if (page.value < 1) page.value = 1
}
</script>

<style scoped>
.page-header p {
  margin: 6px 0 0;
}
</style>
