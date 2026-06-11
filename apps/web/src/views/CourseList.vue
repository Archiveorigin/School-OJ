<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h2>{{ canManage ? '课程列表' : '我的课程' }}</h2>
        <p class="muted">{{ canManage ? '管理课程资料、班级和协作成员。' : '这里仅显示你所属班级对应的课程。' }}</p>
      </div>
      <div class="toolbar">
        <el-button v-if="canManage" type="primary" @click="openCourseDialog()">新建课程</el-button>
        <el-button @click="router.push('/courses')">返回入口</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>

    <div class="panel">
      <div class="list-tools">
        <el-select v-model="termFilter" clearable placeholder="全部学期" class="term-filter">
          <el-option v-for="term in termOptions" :key="term" :label="term" :value="term" />
        </el-select>
        <el-switch v-if="canManage" v-model="showArchived" active-text="显示归档" />
      </div>

      <el-table :data="pagedCourses" v-loading="loading">
        <el-table-column v-if="canManage" prop="code" label="课程代码" width="150" />
        <el-table-column prop="name" label="课程名称" min-width="180">
          <template #default="{ row }">
            <button class="link-button" type="button" @click="openClasses(row)">{{ row.name }}</button>
          </template>
        </el-table-column>
        <el-table-column prop="term" label="学期" width="140" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.archived ? 'info' : 'success'" effect="plain">{{ row.archived ? '已归档' : '进行中' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip />
        <el-table-column label="班级" width="100">
          <template #default="{ row }">{{ courseClassCount(row.id) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="420" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="openClasses(row)">班级</el-button>
            <el-button v-if="canManage" size="small" @click="openMembers(row)">成员</el-button>
            <el-button v-if="canManage" size="small" @click="openCourseDialog(row)">编辑</el-button>
            <el-button v-if="canManage && !row.archived" size="small" @click="openClassDialog(row.id)">加班级</el-button>
            <el-button v-if="canManage" size="small" :type="row.archived ? 'success' : 'warning'" plain @click="setCourseArchived(row, !row.archived)">
              {{ row.archived ? '恢复' : '归档' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <ListPagination v-model:page="page" v-model:page-size="pageSize" :total="filteredCourses.length" />
    </div>

    <el-dialog v-model="courseDialogVisible" :title="editingCourseId ? '编辑课程' : '新建课程'" width="560px">
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
        <el-button type="primary" :loading="savingCourse" @click="submitCourse">{{ editingCourseId ? '保存' : '创建' }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="classDialogVisible" title="新建班级" width="520px">
      <el-form :model="classForm" label-width="90px">
        <el-form-item label="课程">
          <el-select v-model="classForm.course_id" style="width: 100%">
            <el-option v-for="course in activeCourses" :key="course.id" :label="courseOptionLabel(course)" :value="course.id" />
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

    <el-drawer v-model="memberDrawerVisible" size="720px" :title="memberCourse ? `成员管理：${memberCourse.name}` : '成员管理'">
      <div class="member-form">
        <el-input v-model="memberForm.email" placeholder="成员邮箱" />
        <el-select v-model="memberForm.role" class="role-select">
          <el-option label="主讲教师" value="course_admin" />
          <el-option label="助教" value="course_assistant" />
          <el-option label="学生" value="student" />
        </el-select>
        <el-select v-model="memberForm.class_id" clearable placeholder="加入班级" class="class-select">
          <el-option v-for="item in memberCourseClasses" :key="item.class_id" :label="item.class_name" :value="item.class_id" />
        </el-select>
        <el-button type="primary" :loading="savingMember" @click="submitMember">添加</el-button>
      </div>

      <el-table :data="members" v-loading="loadingMembers">
        <el-table-column prop="name" label="姓名" min-width="120" />
        <el-table-column prop="email" label="邮箱" min-width="190" show-overflow-tooltip />
        <el-table-column prop="student_no" label="学号" width="130" />
        <el-table-column label="课程角色" width="120">
          <template #default="{ row }">{{ courseRoleLabel(row.role) }}</template>
        </el-table-column>
        <el-table-column prop="class_count" label="班级数" width="90" />
        <el-table-column label="操作" width="90">
          <template #default="{ row }">
            <el-button size="small" type="danger" plain @click="removeMember(row)">移除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-drawer>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
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
  archived?: boolean
}

type CourseMemberRole = 'student' | 'teacher' | 'admin' | 'course_admin' | 'course_assistant'

type CourseMember = {
  id: number
  course_id: number
  user_id: number
  role: CourseMemberRole
  user_role: string
  email: string
  name: string
  student_no?: string
  class_count: number
  created_at: string
}

const auth = useAuthStore()
const classroom = useClassroomStore()
const router = useRouter()
const canManage = computed(() => auth.role === 'admin' || auth.role === 'teacher')
const loading = ref(false)
const courses = ref<Course[]>([])
const page = ref(1)
const pageSize = ref(10)
const termFilter = ref('')
const showArchived = ref(false)
const courseDialogVisible = ref(false)
const classDialogVisible = ref(false)
const memberDrawerVisible = ref(false)
const savingCourse = ref(false)
const savingClass = ref(false)
const savingMember = ref(false)
const loadingMembers = ref(false)
const editingCourseId = ref<number>()
const memberCourse = ref<Course>()
const members = ref<CourseMember[]>([])
const courseForm = reactive({ code: '', name: '', term: '2026 春', description: '' })
const classForm = reactive({ course_id: 0, name: '' })
const memberForm = reactive<{ email: string; role: CourseMemberRole; class_id?: number }>({ email: '', role: 'course_assistant', class_id: undefined })

const termOptions = computed(() => Array.from(new Set(courses.value.map((course) => course.term).filter(Boolean) as string[])))
const activeCourses = computed(() => courses.value.filter((course) => !course.archived))
const filteredCourses = computed(() => {
  if (!termFilter.value) return courses.value
  return courses.value.filter((course) => course.term === termFilter.value)
})
const pagedCourses = computed(() => filteredCourses.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const memberCourseClasses = computed(() => {
  if (!memberCourse.value) return []
  return classroom.classes.filter((item) => item.course_id === memberCourse.value?.id)
})

function list<T>(value: T[] | null | undefined) {
  return Array.isArray(value) ? value : []
}

async function load() {
  loading.value = true
  try {
    const [coursesRes] = await Promise.all([
      client.get('/courses', { params: { include_archived: showArchived.value || undefined } }),
      classroom.load({ force: true })
    ])
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

function courseRoleLabel(role: CourseMemberRole) {
  const labels: Record<CourseMemberRole, string> = {
    student: '学生',
    teacher: '教师',
    admin: '管理员',
    course_admin: '主讲教师',
    course_assistant: '助教'
  }
  return labels[role] || role
}

function openClasses(course: Course) {
  router.push({ path: '/classes', query: { course_id: course.id } })
}

function openCourseDialog(course?: Course) {
  editingCourseId.value = course?.id
  courseForm.code = course?.code || ''
  courseForm.name = course?.name || ''
  courseForm.term = course?.term || '2026 春'
  courseForm.description = course?.description || ''
  courseDialogVisible.value = true
}

function openClassDialog(courseID: number) {
  classForm.course_id = courseID
  classForm.name = ''
  classDialogVisible.value = true
}

async function submitCourse() {
  if (!courseForm.code.trim() || !courseForm.name.trim()) {
    ElMessage.error('请填写课程代码和课程名称')
    return
  }
  savingCourse.value = true
  try {
    if (editingCourseId.value) {
      await client.put(`/courses/${editingCourseId.value}`, { ...courseForm })
      ElMessage.success('课程已保存')
    } else {
      await client.post('/courses', { ...courseForm })
      ElMessage.success('课程已创建')
    }
    courseDialogVisible.value = false
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    savingCourse.value = false
  }
}

async function setCourseArchived(course: Course, archived: boolean) {
  try {
    await ElMessageBox.confirm(`确定${archived ? '归档' : '恢复'}课程「${course.name}」吗？`, archived ? '归档课程' : '恢复课程', {
      confirmButtonText: archived ? '归档' : '恢复',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }
  try {
    await client.post(`/courses/${course.id}/archive`, { archived })
    ElMessage.success(archived ? '课程已归档' : '课程已恢复')
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

async function submitClass() {
  if (!classForm.course_id || !classForm.name.trim()) {
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

async function openMembers(course: Course) {
  memberCourse.value = course
  memberDrawerVisible.value = true
  memberForm.email = ''
  memberForm.role = 'course_assistant'
  memberForm.class_id = undefined
  await loadMembers()
}

async function loadMembers() {
  if (!memberCourse.value) return
  loadingMembers.value = true
  try {
    const { data } = await client.get(`/courses/${memberCourse.value.id}/members`)
    members.value = list(data)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loadingMembers.value = false
  }
}

async function submitMember() {
  if (!memberCourse.value || !memberForm.email.trim()) {
    ElMessage.error('请填写成员邮箱')
    return
  }
  savingMember.value = true
  try {
    await client.post(`/courses/${memberCourse.value.id}/members`, {
      email: memberForm.email.trim(),
      role: memberForm.role,
      class_id: memberForm.class_id || undefined
    })
    ElMessage.success('成员已添加')
    memberForm.email = ''
    await loadMembers()
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    savingMember.value = false
  }
}

async function removeMember(member: CourseMember) {
  if (!memberCourse.value) return
  try {
    await ElMessageBox.confirm(`确定移除「${member.name || member.email}」吗？`, '移除成员', {
      confirmButtonText: '移除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }
  try {
    await client.delete(`/courses/${memberCourse.value.id}/members/${member.user_id}`)
    ElMessage.success('成员已移除')
    await loadMembers()
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

watch([pageSize, termFilter], clampPage)
watch(showArchived, load)

onMounted(load)

function clampPage() {
  const maxPage = Math.max(1, Math.ceil(filteredCourses.value.length / pageSize.value))
  if (page.value > maxPage) page.value = maxPage
  if (page.value < 1) page.value = 1
}
</script>

<style scoped>
.page-header p {
  margin: 6px 0 0;
}

.list-tools,
.member-form {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.term-filter {
  width: 180px;
}

.role-select {
  width: 130px;
}

.class-select {
  width: 160px;
}

.link-button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--accent);
  font: inherit;
  cursor: pointer;
}

.link-button:hover {
  color: var(--accent-strong);
}
</style>
