<template>
  <section class="page">
    <div class="page-header">
      <h2>JPlag 查重</h2>
      <div class="toolbar">
        <el-input v-model="form.course_id" placeholder="课程 ID" style="width: 110px" />
        <el-input v-model="form.assignment_id" placeholder="作业 ID" style="width: 110px" clearable />
        <el-select v-model="form.language" style="width: 120px">
          <el-option label="C++" value="cpp" />
          <el-option label="C" value="c" />
          <el-option label="Python" value="python" />
          <el-option label="Java" value="java" />
        </el-select>
        <el-button type="primary" @click="create">启动查重</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>
    <div class="panel">
      <el-table :data="jobs">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="course_id" label="课程" width="90" />
        <el-table-column prop="assignment_id" label="作业" width="90" />
        <el-table-column prop="language" label="语言" width="100" />
        <el-table-column prop="status" label="状态" width="120" />
        <el-table-column prop="message" label="消息" />
        <el-table-column prop="report_object" label="报告对象" />
      </el-table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { onMounted, reactive, ref } from 'vue'
import { client } from '../api/client'

const jobs = ref<any[]>([])
const form = reactive({ course_id: '1', assignment_id: '1', language: 'cpp' })

async function load() {
  jobs.value = (await client.get('/plagiarism/jobs')).data
}

async function create() {
  await client.post('/plagiarism/jobs', {
    course_id: Number(form.course_id),
    assignment_id: form.assignment_id ? Number(form.assignment_id) : undefined,
    language: form.language
  })
  ElMessage.success('已启动')
  load()
}

onMounted(load)
</script>
