<template>
  <section class="page">
    <div class="page-header">
      <h2>用户管理</h2>
      <div class="toolbar">
        <el-button type="primary" @click="openCreate">新建用户</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>

    <div class="panel">
      <el-table :data="users" v-loading="loading">
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
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, reactive, ref } from 'vue'
import { client, type Role, type User } from '../api/client'
import { formatDateTime } from '../features/time'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const users = ref<User[]>([])
const loading = ref(false)
const createVisible = ref(false)
const editVisible = ref(false)
const resetVisible = ref(false)
const savingCreate = ref(false)
const savingEdit = ref(false)
const savingReset = ref(false)

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

async function load() {
  loading.value = true
  try {
    users.value = (await client.get('/users')).data
  } catch (err: any) {
    ElMessage.error(errorText(err))
  } finally {
    loading.value = false
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
  return '学生'
}

onMounted(load)
</script>

<style scoped>
.row-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}
</style>
