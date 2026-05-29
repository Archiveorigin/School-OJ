<template>
  <section class="page">
    <div class="page-header">
      <h2>课程班级</h2>
      <div class="toolbar">
        <el-button v-if="canManage" type="primary" @click="createCourse">新建课程</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>
    <el-row :gutter="16">
      <el-col :span="14">
        <div class="panel">
          <el-table :data="courses">
            <el-table-column prop="code" label="代码" width="140" />
            <el-table-column prop="name" label="课程" />
            <el-table-column prop="term" label="学期" width="120" />
            <el-table-column label="操作" width="180">
              <template #default="{ row }">
                <el-button size="small" @click="loadClasses(row.id)">班级</el-button>
                <el-button v-if="canManage" size="small" @click="createClass(row.id)">加班级</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>
      <el-col :span="10">
        <div class="panel">
          <h3>班级</h3>
          <el-table :data="classes" size="small">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="course_id" label="课程" width="90" />
            <el-table-column prop="name" label="名称" />
          </el-table>
        </div>
      </el-col>
    </el-row>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import { client } from '../api/client'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const canManage = computed(() => auth.role === 'admin' || auth.role === 'teacher')
const courses = ref<any[]>([])
const classes = ref<any[]>([])

async function load() {
  courses.value = (await client.get('/courses')).data
  classes.value = (await client.get('/classes')).data
}

async function loadClasses(courseID: number) {
  classes.value = (await client.get('/classes', { params: { course_id: courseID } })).data
}

async function createCourse() {
  const code = await promptText('课程代码')
  const name = await promptText('课程名称')
  await client.post('/courses', { code, name, term: '2026 春' })
  ElMessage.success('已创建')
  load()
}

async function createClass(courseID: number) {
  const name = await promptText('班级名称')
  await client.post(`/courses/${courseID}/classes`, { name })
  ElMessage.success('已创建')
  loadClasses(courseID)
}

async function promptText(title: string) {
  const { value } = await ElMessageBox.prompt(title, title)
  return value
}

onMounted(load)
</script>
