<template>
  <section class="page">
    <div class="page-header">
      <h2>考试</h2>
      <div class="toolbar">
        <el-button v-if="canManage" type="primary" @click="create">新建考试</el-button>
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
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import { client } from '../api/client'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const canManage = computed(() => auth.role !== 'student')
const items = ref<any[]>([])

async function load() {
  items.value = (await client.get('/exams')).data
}

async function create() {
  const courseID = Number((await ElMessageBox.prompt('课程 ID', '新建考试')).value)
  const title = (await ElMessageBox.prompt('标题', '新建考试')).value
  const problemIDs = ((await ElMessageBox.prompt('题目 ID，逗号分隔', '新建考试')).value || '')
    .split(',')
    .map((v) => Number(v.trim()))
    .filter(Boolean)
  await client.post('/exams', { course_id: courseID, title, problem_ids: problemIDs })
  ElMessage.success('已创建')
  load()
}

onMounted(load)
</script>
