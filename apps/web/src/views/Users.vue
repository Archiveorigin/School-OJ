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
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="email" label="邮箱" />
        <el-table-column prop="name" label="姓名" />
        <el-table-column prop="role" label="角色" width="120" />
        <el-table-column prop="student_no" label="学号" width="140" />
      </el-table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, ref } from 'vue'
import { client, type User } from '../api/client'

const users = ref<User[]>([])

async function load() {
  users.value = (await client.get('/users')).data
}

async function create() {
  const email = (await ElMessageBox.prompt('邮箱', '新建用户')).value
  const name = (await ElMessageBox.prompt('姓名', '新建用户')).value
  const role = (await ElMessageBox.prompt('角色 student/teacher/admin', '新建用户')).value
  await client.post('/users', { email, name, role, password: 'password' })
  ElMessage.success('已创建，默认密码 password')
  load()
}

onMounted(load)
</script>
