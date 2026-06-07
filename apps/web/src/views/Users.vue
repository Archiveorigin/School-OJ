<template>
  <section class="page">
    <div class="page-header">
      <h2>用户</h2>
      <div class="toolbar">
        <el-button type="primary" @click="create">新建用户</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>
    <div class="panel">
      <el-table :data="users">
        <el-table-column prop="email" label="邮箱" />
        <el-table-column prop="name" label="姓名" />
        <el-table-column label="角色" width="120">
          <template #default="{ row }">{{ roleText(row.role) }}</template>
        </el-table-column>
        <el-table-column prop="student_no" label="学号" width="140" />
        <el-table-column label="创建时间" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
      </el-table>
    </div>
    <el-dialog v-model="dialogVisible" title="新建用户" width="520px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="邮箱">
          <el-input v-model="form.email" placeholder="student@example.edu" />
        </el-form-item>
        <el-form-item label="姓名">
          <el-input v-model="form.name" placeholder="张三" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="form.role" style="width: 100%">
            <el-option label="学生" value="student" />
            <el-option label="教师" value="teacher" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="学号">
          <el-input v-model="form.student_no" placeholder="学生账号可填写" />
        </el-form-item>
        <el-form-item label="初始密码">
          <el-input v-model="form.password" type="password" show-password />
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
import { onMounted, reactive, ref } from 'vue'
import { client, type User } from '../api/client'
import { formatDateTime } from '../features/time'

const users = ref<User[]>([])
const dialogVisible = ref(false)
const saving = ref(false)
const form = reactive({
  email: '',
  name: '',
  role: 'student',
  student_no: '',
  password: ''
})

async function load() {
  try {
    users.value = (await client.get('/users')).data
  } catch (err: any) {
    ElMessage.error(errorText(err))
  }
}

async function create() {
  dialogVisible.value = true
}

async function submit() {
  if (!form.email || !form.name || !form.password) {
    ElMessage.error('请填写邮箱、姓名和密码')
    return
  }
  saving.value = true
  try {
    await client.post('/users', { ...form })
    ElMessage.success('用户已创建')
    dialogVisible.value = false
    reset()
    await load()
  } catch (err: any) {
    ElMessage.error(errorText(err))
  } finally {
    saving.value = false
  }
}

function reset() {
  form.email = ''
  form.name = ''
  form.role = 'student'
  form.student_no = ''
  form.password = ''
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
