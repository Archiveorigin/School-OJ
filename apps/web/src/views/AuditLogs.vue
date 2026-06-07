<template>
  <section class="page">
    <div class="page-header">
      <h2>审计日志</h2>
      <el-button @click="load">刷新</el-button>
    </div>
    <div class="panel">
      <el-table :data="items">
        <el-table-column label="用户" min-width="120">
          <template #default="{ row }">{{ row.actor_name || '系统' }}</template>
        </el-table-column>
        <el-table-column prop="action" label="动作" width="180" />
        <el-table-column label="资源" width="140">
          <template #default="{ row }">{{ row.resource_label || row.resource_type || '-' }}</template>
        </el-table-column>
        <el-table-column prop="ip" label="IP" width="140" />
        <el-table-column label="时间" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
      </el-table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { client } from '../api/client'
import { formatDateTime } from '../features/time'

const items = ref<any[]>([])

async function load() {
  items.value = (await client.get('/audit-logs')).data
}

onMounted(load)
</script>
