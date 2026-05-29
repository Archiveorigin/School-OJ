<template>
  <section class="page">
    <div class="page-header">
      <h2>排行榜</h2>
      <div class="toolbar">
        <el-input v-model="assignmentID" placeholder="作业 ID" clearable style="width: 120px" />
        <el-input v-model="examID" placeholder="考试 ID" clearable style="width: 120px" />
        <el-button @click="load">刷新</el-button>
      </div>
    </div>
    <div class="panel">
      <el-table :data="rows">
        <el-table-column type="index" label="#" width="70" />
        <el-table-column prop="name" label="学生" />
        <el-table-column prop="solved" label="通过题数" width="120" />
        <el-table-column prop="score" label="总分" width="100" />
        <el-table-column prop="last_submission" label="最后提交" />
      </el-table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { client } from '../api/client'

const rows = ref<any[]>([])
const assignmentID = ref('')
const examID = ref('')

async function load() {
  rows.value = (
    await client.get('/leaderboard', {
      params: { assignment_id: assignmentID.value, exam_id: examID.value }
    })
  ).data
}

onMounted(load)
</script>
