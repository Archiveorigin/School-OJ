<template>
  <section class="page">
    <div class="page-header">
      <h2>审计日志</h2>
      <el-button @click="load">刷新</el-button>
    </div>
    <div class="panel">
      <el-table :data="items">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="actor_user_id" label="用户" width="90" />
        <el-table-column prop="action" label="动作" width="180" />
        <el-table-column prop="resource_type" label="资源" width="120" />
        <el-table-column prop="resource_id" label="资源 ID" width="100" />
        <el-table-column prop="ip" label="IP" width="140" />
        <el-table-column prop="created_at" label="时间" />
      </el-table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { client } from '../api/client'

const items = ref<any[]>([])

async function load() {
  items.value = (await client.get('/audit-logs')).data
}

onMounted(load)
</script>
