<template>
  <section class="page">
    <div class="page-header">
      <h2>用户管理</h2>
      <div class="toolbar">
        <el-button type="primary" @click="openCreate">新建用户</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>

    <div class="panel applications-panel">
      <div class="section-heading">
        <div>
          <h3>出题者申请</h3>
          <p>审核通过后，学生账号会自动转换为出题者。</p>
        </div>
        <el-tag type="warning">{{ pendingApplications }} 待审核</el-tag>
      </div>
      <el-table :data="applications" empty-text="暂无出题者申请">
        <el-table-column label="申请人" min-width="180">
          <template #default="{ row }">
            <strong>{{ row.user?.name || `用户 #${row.user_id}` }}</strong>
            <div class="muted">{{ row.user?.email }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="motivation" label="申请说明" min-width="300" show-overflow-tooltip />
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="applicationStatusType(row.status)">{{ applicationStatusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="申请时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="审核说明" min-width="180">
          <template #default="{ row }">{{ row.review_note || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <template v-if="row.status === 'pending'">
              <el-button size="small" type="success" text @click="openReview(row, 'approved')">通过</el-button>
              <el-button size="small" type="danger" text @click="openReview(row, 'rejected')">驳回</el-button>
            </template>
            <span v-else class="muted">已审核</span>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <div class="panel">
      <el-table :data="pagedUsers" v-loading="loading">
        <el-table-column prop="email" label="邮箱" min-width="220" />
        <el-table-column prop="name" label="姓名" min-width="140" />
        <el-table-column label="角色" width="120">
          <template #default="{ row }">{{ roleText(row.role) }}</template>
        </el-table-column>
        <el-table-column prop="student_no" label="学号" width="150" />
        <el-table-column label="创建时间" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="250" fixed="right">
          <template #default="{ row }">
            <div class="row-actions">
              <el-button size="small" type="primary" text @click="openEdit(row)">编辑</el-button>
              <el-button size="small" type="warning" text @click="openResetPassword(row)">重置密码</el-button>
              <el-button size="small" type="danger" text :disabled="row.id === auth.user?.id" @click="confirmDelete(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <ListPagination v-model:page="page" v-model:page-size="pageSize" :total="users.length" />
    </div>

    <el-dialog v-model="createVisible" title="新建用户" width="520px" @closed="resetCreateForm">
      <el-form :model="createForm" label-width="90px">
        <el-form-item label="邮箱">
          <el-input v-model="createForm.email" placeholder="student@example.edu" />
        </el-form-item>
        <el-form-item label="姓名">
          <el-input v-model="createForm.name" placeholder="张三" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="createForm.role" style="width: 100%">
            <el-option label="学生" value="student" />
            <el-option label="出题者" value="problem_setter" />
            <el-option label="教师" value="teacher" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="学号">
          <el-input v-model="createForm.student_no" placeholder="学生账号可填写" />
        </el-form-item>
        <el-form-item label="初始密码">
          <el-input v-model="createForm.password" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingCreate" @click="submitCreate">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="editVisible" title="编辑用户" width="520px">
      <el-form :model="editForm" label-width="90px">
        <el-form-item label="邮箱">
          <el-input v-model="editForm.email" />
        </el-form-item>
        <el-form-item label="姓名">
          <el-input v-model="editForm.name" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="editForm.role" style="width: 100%">
            <el-option label="学生" value="student" />
            <el-option label="出题者" value="problem_setter" />
            <el-option label="教师" value="teacher" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="学号">
          <el-input v-model="editForm.student_no" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingEdit" @click="submitEdit">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="resetVisible" title="重置密码" width="480px" @closed="resetPasswordForm.password = ''">
      <el-form :model="resetPasswordForm" label-width="90px">
        <el-form-item label="用户">
          <el-input :model-value="resetPasswordForm.label" disabled />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="resetPasswordForm.password" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingReset" @click="submitResetPassword">重置</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="reviewVisible" :title="reviewForm.status === 'approved' ? '通过出题者申请' : '驳回出题者申请'" width="520px">
      <el-form label-position="top">
        <el-form-item label="申请人">
          <el-input :model-value="reviewForm.label" disabled />
        </el-form-item>
        <el-form-item label="审核说明">
          <el-input
            v-model="reviewForm.review_note"
            type="textarea"
            :rows="4"
            :placeholder="reviewForm.status === 'approved' ? '可选：填写欢迎语或出题规范提示' : '请说明驳回原因，便于申请人修改'"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="reviewVisible = false">取消</el-button>
        <el-button
          :type="reviewForm.status === 'approved' ? 'success' : 'danger'"
          :loading="reviewing"
          @click="submitReview"
        >
          确认{{ reviewForm.status === 'approved' ? '通过' : '驳回' }}
        </el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { client, type AuthorApplication, type Role, type User } from '../api/client'
import ListPagination from '../components/ListPagination.vue'
import { formatDateTime } from '../features/time'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const users = ref<User[]>([])
const applications = ref<AuthorApplication[]>([])
const loading = ref(false)
const createVisible = ref(false)
const editVisible = ref(false)
const resetVisible = ref(false)
const savingCreate = ref(false)
const savingEdit = ref(false)
const savingReset = ref(false)
const reviewVisible = ref(false)
const reviewing = ref(false)
const page = ref(1)
const pageSize = ref(20)
const pagedUsers = computed(() => users.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const pendingApplications = computed(() => applications.value.filter((item) => item.status === 'pending').length)

const createForm = reactive({
  email: '',
  name: '',
  role: 'student' as Role,
  student_no: '',
  password: ''
})

const editForm = reactive({
  id: 0,
  email: '',
  name: '',
  role: 'student' as Role,
  student_no: ''
})

const resetPasswordForm = reactive({
  id: 0,
  label: '',
  password: ''
})
const reviewForm = reactive({
  id: 0,
  label: '',
  status: 'approved' as 'approved' | 'rejected',
  review_note: ''
})

async function load() {
  loading.value = true
  try {
    const [usersResponse, applicationsResponse] = await Promise.all([
      client.get('/users'),
      client.get('/author-applications')
    ])
    users.value = usersResponse.data
    applications.value = applicationsResponse.data
  } catch (err: any) {
    ElMessage.error(errorText(err))
  } finally {
    loading.value = false
  }
}

function openReview(row: AuthorApplication, status: 'approved' | 'rejected') {
  reviewForm.id = row.id
  reviewForm.label = `${row.user?.name || `用户 #${row.user_id}`} <${row.user?.email || '-'}>`
  reviewForm.status = status
  reviewForm.review_note = ''
  reviewVisible.value = true
}

async function submitReview() {
  if (reviewForm.status === 'rejected' && !reviewForm.review_note.trim()) {
    ElMessage.error('驳回申请时请填写审核说明')
    return
  }
  reviewing.value = true
  try {
    await client.put(`/author-applications/${reviewForm.id}/review`, {
      status: reviewForm.status,
      review_note: reviewForm.review_note.trim()
    })
    reviewVisible.value = false
    ElMessage.success(reviewForm.status === 'approved' ? '已通过申请并授予出题者身份' : '已驳回申请')
    await load()
  } catch (err: any) {
    ElMessage.error(errorText(err))
  } finally {
    reviewing.value = false
  }
}

function openCreate() {
  resetCreateForm()
  createVisible.value = true
}

async function openEdit(row: User) {
  try {
    const { data } = await client.get(`/users/${row.id}`)
    editForm.id = data.id
    editForm.email = data.email || ''
    editForm.name = data.name || ''
    editForm.role = data.role || 'student'
    editForm.student_no = data.student_no || ''
    editVisible.value = true
  } catch (err: any) {
    ElMessage.error(errorText(err))
  }
}

function openResetPassword(row: User) {
  resetPasswordForm.id = row.id
  resetPasswordForm.label = `${row.name} <${row.email}>`
  resetPasswordForm.password = ''
  resetVisible.value = true
}

async function submitCreate() {
  if (!validateUserForm(createForm.email, createForm.name, createForm.password)) return
  savingCreate.value = true
  try {
    await client.post('/users', { ...createForm })
    ElMessage.success('用户已创建')
    createVisible.value = false
    await load()
  } catch (err: any) {
    ElMessage.error(errorText(err))
  } finally {
    savingCreate.value = false
  }
}

async function submitEdit() {
  if (!validateUserForm(editForm.email, editForm.name)) return
  savingEdit.value = true
  try {
    await client.put(`/users/${editForm.id}`, {
      email: editForm.email,
      name: editForm.name,
      role: editForm.role,
      student_no: editForm.student_no
    })
    ElMessage.success('用户已更新')
    editVisible.value = false
    await load()
  } catch (err: any) {
    ElMessage.error(errorText(err))
  } finally {
    savingEdit.value = false
  }
}

async function submitResetPassword() {
  if (resetPasswordForm.password.length < 6) {
    ElMessage.error('新密码至少 6 位')
    return
  }
  savingReset.value = true
  try {
    await client.post(`/users/${resetPasswordForm.id}/reset-password`, { password: resetPasswordForm.password })
    ElMessage.success('密码已重置')
    resetVisible.value = false
  } catch (err: any) {
    ElMessage.error(errorText(err))
  } finally {
    savingReset.value = false
  }
}

async function confirmDelete(row: User) {
  if (row.id === auth.user?.id) {
    ElMessage.warning('不能删除当前登录账号')
    return
  }
  try {
    await ElMessageBox.confirm(`确认删除用户 ${row.name}？`, '删除用户', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
      confirmButtonClass: 'el-button--danger'
    })
    await client.delete(`/users/${row.id}`)
    ElMessage.success('用户已删除')
    await load()
  } catch (err: any) {
    if (err === 'cancel' || err === 'close') return
    ElMessage.error(errorText(err))
  }
}

function resetCreateForm() {
  createForm.email = ''
  createForm.name = ''
  createForm.role = 'student'
  createForm.student_no = ''
  createForm.password = ''
}

function validateUserForm(email: string, name: string, password?: string) {
  if (!email.trim() || !name.trim()) {
    ElMessage.error('请填写邮箱和姓名')
    return false
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) {
    ElMessage.error('邮箱格式不正确')
    return false
  }
  if (password !== undefined && password.length < 6) {
    ElMessage.error('密码至少 6 位')
    return false
  }
  return true
}

function errorText(err: any) {
  if (err.response?.status === 403) {
    return '当前请求没有管理员权限，请退出后使用管理员账号重新登录'
  }
  return err.response?.data?.error || err.message
}

function roleText(role: string) {
  if (role === 'admin') return '管理员'
  if (role === 'teacher') return '教师'
  if (role === 'problem_setter') return '出题者'
  return '学生'
}

function applicationStatusText(status: string) {
  if (status === 'approved') return '已通过'
  if (status === 'rejected') return '已驳回'
  return '待审核'
}

function applicationStatusType(status: string): 'success' | 'danger' | 'warning' {
  if (status === 'approved') return 'success'
  if (status === 'rejected') return 'danger'
  return 'warning'
}

watch(pageSize, () => { page.value = 1 })
onMounted(load)
</script>

<style scoped>
.row-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}
.applications-panel {
  margin-bottom: 18px;
}
.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}
.section-heading h3 {
  margin: 0 0 4px;
}
.section-heading p {
  margin: 0;
  color: var(--muted);
}
</style>
