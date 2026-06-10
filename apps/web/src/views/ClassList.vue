<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h2>{{ canManage ? '班级列表' : '我的班级' }}</h2>
        <p class="muted">{{ canManage ? '查看课程下的班级，并进入当前班级工作区。' : '这里仅显示你已加入的班级。' }}</p>
      </div>
      <div class="toolbar">
        <el-button v-if="canManage" type="primary" @click="openClassDialog()">新建班级</el-button>
        <el-button @click="router.push('/courses')">返回入口</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>

    <div v-if="auth.role === 'student'" class="panel join-strip">
      <el-input v-model="joinClassCode" placeholder="班级邀请码" class="join-code-input" />
      <el-button type="primary" :loading="joining" @click="joinClass">加入班级</el-button>
      <span class="muted">输入教师提供的邀请码后，即可看到该班级题库、作业、考试。</span>
    </div>

    <div class="panel">
      <div v-if="selectedCourseName" class="filtered-course">
        <span class="muted">当前课程：</span>
        <strong>{{ selectedCourseName }}</strong>
        <el-button text type="primary" @click="clearCourseFilter">查看全部班级</el-button>
      </div>
      <el-table :data="pagedClasses" v-loading="loading">
        <el-table-column v-if="canManage" prop="class_id" label="班级 ID" width="100" />
        <el-table-column prop="class_name" label="班级名称" min-width="180" />
        <el-table-column label="所属课程" min-width="200">
          <template #default="{ row }">{{ classCourseLabel(row) }}</template>
        </el-table-column>
        <el-table-column prop="term" label="学期" width="140" />
        <el-table-column label="操作" width="210">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="activate(row.class_id)">切换</el-button>
            <el-button
              v-if="auth.role === 'student'"
              size="small"
              type="danger"
              plain
              :loading="leavingClassId === row.class_id"
              @click="leaveClass(row)"
            >
              退出
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <ListPagination v-model:page="page" v-model:page-size="pageSize" :total="filteredClasses.length" />
    </div>

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
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client, type ClassContext } from '../api/client'
import ListPagination from '../components/ListPagination.vue'
import { useAuthStore } from '../stores/auth'
import { useClassroomStore } from '../stores/classroom'

type Course = {
  id: number
  code?: string
  name: string
}

const auth = useAuthStore()
const classroom = useClassroomStore()
const route = useRoute()
const router = useRouter()
const canManage = computed(() => auth.role === 'admin' || auth.role === 'teacher')
const loading = ref(false)
const courses = ref<Course[]>([])
const page = ref(1)
const pageSize = ref(10)
const joinClassCode = ref('')
const joining = ref(false)
const leavingClassId = ref<number>()
const classDialogVisible = ref(false)
const savingClass = ref(false)
const classForm = reactive({ course_id: 0, name: '' })
const courseIDFilter = computed(() => Number(route.query.course_id || 0))
const filteredClasses = computed(() => {
  if (!courseIDFilter.value) return classroom.classes
  return classroom.classes.filter((item) => item.course_id === courseIDFilter.value)
})
const selectedCourseName = computed(() => {
  if (!courseIDFilter.value) return ''
  return courses.value.find((course) => course.id === courseIDFilter.value)?.name || filteredClasses.value[0]?.course_name || ''
})
const pagedClasses = computed(() => filteredClasses.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))

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

function courseOptionLabel(course: Course) {
  return course.code ? `${course.code} ${course.name}` : course.name
}

function classCourseLabel(row: ClassContext) {
  if (auth.role === 'student') return row.course_name
  return row.course_code ? `${row.course_code} / ${row.course_name}` : row.course_name
}

function openClassDialog(courseID = courseIDFilter.value) {
  classForm.course_id = courseID || courses.value[0]?.id || 0
  classForm.name = ''
  classDialogVisible.value = true
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

async function joinClass() {
  if (!joinClassCode.value.trim()) {
    ElMessage.error('请填写班级邀请码')
    return
  }
  joining.value = true
  try {
    const { data } = await client.post('/classes/join', { join_code: joinClassCode.value.trim() })
    ElMessage.success('已加入班级')
    classroom.setActive(data.class_id)
    joinClassCode.value = ''
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    joining.value = false
  }
}

async function leaveClass(row: ClassContext) {
  try {
    await ElMessageBox.confirm(`确定退出班级「${row.class_name}」吗？退出后将无法继续查看该班级题库、作业和考试。`, '退出班级', {
      confirmButtonText: '退出',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }
  leavingClassId.value = row.class_id
  try {
    await client.post(`/classes/${row.class_id}/leave`)
    ElMessage.success('已退出班级')
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    leavingClassId.value = undefined
  }
}

function activate(classID: number) {
  classroom.setActive(classID)
  ElMessage.success('已切换班级')
}

function clearCourseFilter() {
  router.push('/classes')
}

watch(() => route.query.course_id, load)
watch(pageSize, clampPage)

onMounted(load)

function clampPage() {
  const maxPage = Math.max(1, Math.ceil(filteredClasses.value.length / pageSize.value))
  if (page.value > maxPage) page.value = maxPage
  if (page.value < 1) page.value = 1
}
</script>

<style scoped>
.page-header p {
  margin: 6px 0 0;
}

.join-strip {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.join-code-input {
  width: 180px;
}

.filtered-course {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
</style>
