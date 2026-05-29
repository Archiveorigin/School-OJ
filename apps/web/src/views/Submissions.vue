<template>
  <section class="page">
    <div class="page-header">
      <h2>提交</h2>
      <el-button @click="load">刷新</el-button>
    </div>
    <div class="panel">
      <el-table :data="items" @row-click="open">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="user_id" label="用户" width="90" />
        <el-table-column prop="problem_id" label="题目" width="90" />
        <el-table-column prop="language" label="语言" width="110" />
        <el-table-column label="状态">
          <template #default="{ row }"><StatusBadge :status="row.status" /></template>
        </el-table-column>
        <el-table-column prop="score" label="分数" width="90" />
        <el-table-column prop="message" label="消息" />
      </el-table>
    </div>
    <el-dialog v-model="visible" title="提交详情" width="720px">
      <pre>{{ detail }}</pre>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { client, type Submission } from '../api/client'
import StatusBadge from '../components/StatusBadge.vue'

const items = ref<Submission[]>([])
const detail = ref('')
const visible = ref(false)

async function load() {
  items.value = (await client.get('/submissions')).data
}

async function open(row: Submission) {
  detail.value = JSON.stringify((await client.get(`/submissions/${row.id}`)).data, null, 2)
  visible.value = true
}

onMounted(load)
</script>
