<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h2>{{ canManage ? '班级列表' : '我的班级' }}</h2>
        <p class="muted">{{ canManage ? '查看课程下的班级，管理学生名单和导入。' : '这里仅显示你已加入的班级。' }}</p>
      </div>
      <div class="toolbar">
        <el-button v-if="canManage" type="primary" @click="openClassDialog()">新建班级</el-button>
        <el-button @click="router.push('/courses')">返回入口</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>

    <div v-if="auth.role === 'student'" class="panel join-strip">
      <el-input v-model="joinClassCode" placeholder="班级邀请码" class="join-code-input" @input="joinPreview = undefined" />
      <el-button :loading="previewing" @click="previewClass">预览</el-button>
      <el-button type="primary" :disabled="!joinPreview" :loading="joining" @click="joinClass">确认加入</el-button>
      <div v-if="joinPreview" class="join-preview">
        <strong>{{ joinPreview.course_name }} / {{ joinPreview.class_name }}</strong>
        <span class="muted">{{ joinPreview.term || '未设置学期' }} · {{ joinPreview.teacher_name || '未设置教师' }}</span>
        <span class="muted">{{ joinPreview.course_description || '暂无课程描述' }}</span>
      </div>
    </div>

    <div class="panel">
      <div class="list-tools">
        <div v-if="selectedCourseName" class="filtered-course">
          <span class="muted">当前课程：</span>
          <strong>{{ selectedCourseName }}</strong>
          <el-button text type="primary" @click="clearCourseFilter">查看全部班级</el-button>
        </div>
        <el-switch v-if="canManage" v-model="showArchived" active-text="显示归档" />
      </div>

      <el-table :data="pagedClasses" v-loading="loading">
        <el-table-column v-if="canManage" prop="class_id" label="班级 ID" width="100" />
        <el-table-column prop="class_name" label="班级名称" min-width="170" />
        <el-table-column label="所属课程" min-width="210">
          <template #default="{ row }">{{ classCourseLabel(row) }}</template>
        </el-table-column>
        <el-table-column prop="term" label="学期" width="130" />
        <el-table-column v-if="canManage" prop="join_code" label="邀请码" width="120" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.archived ? 'info' : 'success'" effect="plain">{{ row.archived ? '已归档' : '进行中' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="430" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="activate(row.class_id)">切换</el-button>
            <el-button v-if="canManage" size="small" @click="openStudents(row)">学生</el-button>
            <el-button v-if="canManage" size="small" @click="openImport(row)">导入</el-button>
            <el-button v-if="canManage" size="small" @click="openClassDialog(row)">编辑</el-button>
            <el-button v-if="canManage" size="small" :type="row.archived ? 'success' : 'warning'" plain @click="setClassArchived(row, !row.archived)">
              {{ row.archived ? '恢复' : '归档' }}
            </el-button>
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

    <el-dialog v-model="classDialogVisible" :title="editingClassId ? '编辑班级' : '新建班级'" width="520px">
      <el-form :model="classForm" label-width="90px">
        <el-form-item label="课程">
          <el-select v-model="classForm.course_id" style="width: 100%" :disabled="Boolean(editingClassId)">
            <el-option v-for="course in activeCourses" :key="course.id" :label="courseOptionLabel(course)" :value="course.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="班级名称">
          <el-input v-model="classForm.name" placeholder="计科一班" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="classDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingClass" @click="submitClass">{{ editingClassId ? '保存' : '创建' }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="studentsDrawerVisible" size="760px" :title="selectedClass ? `学生名单：${selectedClass.class_name}` : '学生名单'">
      <el-table :data="students" v-loading="loadingStudents">
        <el-table-column prop="name" label="姓名" min-width="120" />
        <el-table-column prop="email" label="邮箱" min-width="190" show-overflow-tooltip />
        <el-table-column prop="student_no" label="学号" width="130" />
        <el-table-column label="加入时间" width="170">
          <template #default="{ row }">{{ formatTime(row.joined_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="90">
          <template #default="{ row }">
            <el-button size="small" type="danger" plain @click="removeStudent(row)">移除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-drawer>

    <el-drawer v-model="importDrawerVisible" size="720px" :title="selectedClass ? `批量导入：${selectedClass.class_name}` : '批量导入'">
      <div class="import-stack">
        <el-alert type="info" show-icon :closable="false" title="支持 CSV 或 Excel，列名可为：学号、姓名、邮箱、初始密码。无表头时按“学号, 姓名, 邮箱”读取。" />
        <el-input v-model="defaultPassword" placeholder="默认初始密码：Aa123456" />
        <el-upload drag :auto-upload="false" :limit="1" accept=".csv,.xlsx" :on-change="selectImportFile" :on-remove="clearImportFile">
          <div class="upload-text">拖入 CSV / Excel，或点击选择文件</div>
        </el-upload>
        <el-input v-model="importText" type="textarea" :rows="8" placeholder="也可以直接粘贴 CSV：学号,姓名,邮箱" />
        <el-button type="primary" :loading="importing" @click="submitImport">开始导入</el-button>
        <el-table v-if="importResults.length" :data="importResults" max-height="280">
          <el-table-column prop="name" label="姓名" width="110" />
          <el-table-column prop="email" label="邮箱" min-width="190" show-overflow-tooltip />
          <el-table-column prop="student_no" label="学号" width="120" />
          <el-table-column label="结果" width="160">
            <template #default="{ row }">
              <el-tag v-if="row.error" type="danger" effect="plain">{{ row.error }}</el-tag>
              <el-tag v-else type="success" effect="plain">{{ row.created ? '已创建并加入' : row.joined ? '已加入' : '已存在' }}</el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-drawer>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client, type ClassContext } from '../api/client'
import { formatDateTime } from '../features/time'
import ListPagination from '../components/ListPagination.vue'
import { useAuthStore } from '../stores/auth'
import { useClassroomStore } from '../stores/classroom'

type Course = {
  id: number
  code?: string
  name: string
  term?: string
  archived?: boolean
}

type ClassRow = ClassContext & {
  id: number
  name?: string
  archived?: boolean
}

type StudentRow = {
  membership_id: number
  user_id: number
  email: string
  name: string
  student_no?: string
  joined_at: string
}

type ImportResult = {
  email: string
  student_no: string
  name: string
  user_id?: number
  created: boolean
  joined: boolean
  error?: string
}

type JoinPreview = {
  class_id: number
  class_name: string
  course_id: number
  course_code: string
  course_name: string
  course_description?: string
  term?: string
  teacher_name?: string
}

const auth = useAuthStore()
const classroom = useClassroomStore()
const route = useRoute()
const router = useRouter()
const canManage = computed(() => auth.role === 'admin' || auth.role === 'teacher')
const loading = ref(false)
const courses = ref<Course[]>([])
const classes = ref<ClassRow[]>([])
const students = ref<StudentRow[]>([])
const importResults = ref<ImportResult[]>([])
const page = ref(1)
const pageSize = ref(10)
const joinClassCode = ref('')
const joinPreview = ref<JoinPreview>()
const previewing = ref(false)
const joining = ref(false)
const leavingClassId = ref<number>()
const showArchived = ref(false)
const classDialogVisible = ref(false)
const studentsDrawerVisible = ref(false)
const importDrawerVisible = ref(false)
const savingClass = ref(false)
const loadingStudents = ref(false)
const importing = ref(false)
const editingClassId = ref<number>()
const selectedClass = ref<ClassRow>()
const importFile = ref<File>()
const importText = ref('')
const defaultPassword = ref('')
const classForm = reactive({ course_id: 0, name: '' })
const courseIDFilter = computed(() => Number(route.query.course_id || 0))
const activeCourses = computed(() => courses.value.filter((course) => !course.archived))
const filteredClasses = computed(() => {
  if (!courseIDFilter.value) return classes.value
  return classes.value.filter((item) => item.course_id === courseIDFilter.value)
})
const selectedCourseName = computed(() => {
  if (!courseIDFilter.value) return ''
  return courses.value.find((course) => course.id === courseIDFilter.value)?.name || filteredClasses.value[0]?.course_name || ''
})
const pagedClasses = computed(() => filteredClasses.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))

function list<T>(value: T[] | null | undefined) {
  return Array.isArray(value) ? value : []
}

function normalizeClass(row: any): ClassRow {
  return {
    ...row,
    id: row.id || row.class_id,
    class_id: row.class_id || row.id,
    class_name: row.class_name || row.name,
    course_id: row.course_id,
    course_code: row.course_code || '',
    course_name: row.course_name || '',
    term: row.term || '',
    join_code: row.join_code || '',
    archived: Boolean(row.archived)
  }
}

async function load() {
  loading.value = true
  try {
    const params = { include_archived: showArchived.value || undefined, course_id: courseIDFilter.value || undefined }
    const courseRequest = client.get('/courses', { params: { include_archived: showArchived.value || undefined } })
    const classRequest = canManage.value ? client.get('/classes', { params }) : classroom.load({ force: true })
    const [coursesRes, classesRes] = await Promise.all([courseRequest, classRequest])
    courses.value = list(coursesRes.data)
    if (canManage.value) {
      classes.value = list((classesRes as any).data).map(normalizeClass)
    } else {
      classes.value = classroom.classes.map(normalizeClass)
    }
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

function openClassDialog(row?: ClassRow) {
  editingClassId.value = row?.class_id
  classForm.course_id = row?.course_id || courseIDFilter.value || activeCourses.value[0]?.id || 0
  classForm.name = row?.class_name || ''
  classDialogVisible.value = true
}

async function submitClass() {
  if (!classForm.course_id || !classForm.name.trim()) {
    ElMessage.error('请选择课程并填写班级名称')
    return
  }
  savingClass.value = true
  try {
    if (editingClassId.value) {
      await client.put(`/classes/${editingClassId.value}`, { name: classForm.name })
      ElMessage.success('班级已保存')
    } else {
      await client.post(`/courses/${classForm.course_id}/classes`, { name: classForm.name })
      ElMessage.success('班级已创建')
    }
    classDialogVisible.value = false
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    savingClass.value = false
  }
}

async function setClassArchived(row: ClassRow, archived: boolean) {
  try {
    await ElMessageBox.confirm(`确定${archived ? '归档' : '恢复'}班级「${row.class_name}」吗？`, archived ? '归档班级' : '恢复班级', {
      confirmButtonText: archived ? '归档' : '恢复',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }
  try {
    await client.post(`/classes/${row.class_id}/archive`, { archived })
    ElMessage.success(archived ? '班级已归档' : '班级已恢复')
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

async function previewClass() {
  if (!joinClassCode.value.trim()) {
    ElMessage.error('请填写班级邀请码')
    return
  }
  previewing.value = true
  try {
    const { data } = await client.get('/classes/join-preview', { params: { join_code: joinClassCode.value.trim() } })
    joinPreview.value = data
  } catch (err: any) {
    joinPreview.value = undefined
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    previewing.value = false
  }
}

async function joinClass() {
  if (!joinPreview.value) {
    await previewClass()
    return
  }
  joining.value = true
  try {
    const { data } = await client.post('/classes/join', { join_code: joinClassCode.value.trim() })
    ElMessage.success('已加入班级')
    classroom.setActive(data.class_id)
    joinClassCode.value = ''
    joinPreview.value = undefined
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

async function openStudents(row: ClassRow) {
  selectedClass.value = row
  studentsDrawerVisible.value = true
  await loadStudents()
}

async function loadStudents() {
  if (!selectedClass.value) return
  loadingStudents.value = true
  try {
    const { data } = await client.get(`/classes/${selectedClass.value.class_id}/students`)
    students.value = list(data)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loadingStudents.value = false
  }
}

async function removeStudent(row: StudentRow) {
  if (!selectedClass.value) return
  try {
    await ElMessageBox.confirm(`确定将「${row.name || row.email}」移出班级吗？`, '移除学生', {
      confirmButtonText: '移除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }
  try {
    await client.delete(`/classes/${selectedClass.value.class_id}/students/${row.user_id}`)
    ElMessage.success('学生已移除')
    await loadStudents()
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

function openImport(row: ClassRow) {
  selectedClass.value = row
  importFile.value = undefined
  importText.value = ''
  defaultPassword.value = ''
  importResults.value = []
  importDrawerVisible.value = true
}

function selectImportFile(file: any) {
  importFile.value = file.raw
}

function clearImportFile() {
  importFile.value = undefined
}

async function submitImport() {
  if (!selectedClass.value) return
  if (!importFile.value && !importText.value.trim()) {
    ElMessage.error('请上传文件或粘贴 CSV 文本')
    return
  }
  importing.value = true
  try {
    const password = defaultPassword.value.trim()
    const url = `/classes/${selectedClass.value.class_id}/students/import`
    const { data } = importFile.value ? await importByFile(url, password) : await client.post(url, { text: importText.value, default_password: password || undefined })
    importResults.value = list(data.results)
    ElMessage.success(`导入完成：创建 ${data.created}，加入 ${data.joined}，失败 ${data.failed}`)
    await loadStudents()
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    importing.value = false
  }
}

function importByFile(url: string, password: string) {
  const form = new FormData()
  if (importFile.value) form.append('file', importFile.value)
  if (password) form.append('default_password', password)
  return client.post(url, form)
}

function formatTime(value: string) {
  return value ? formatDateTime(value) : '-'
}

watch(() => route.query.course_id, load)
watch([pageSize, showArchived], load)

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

.join-strip,
.list-tools,
.filtered-course,
.import-stack {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.join-code-input {
  width: 180px;
}

.join-preview {
  display: grid;
  gap: 2px;
  min-width: min(420px, 100%);
}

.list-tools {
  justify-content: space-between;
}

.import-stack {
  align-items: stretch;
  flex-direction: column;
}

.upload-text {
  padding: 12px;
  color: var(--muted);
}
</style>
