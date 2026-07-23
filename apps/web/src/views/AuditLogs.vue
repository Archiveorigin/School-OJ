<template>
  <section class="page">
    <div class="page-header">
      <h2>审计日志</h2>
      <el-button @click="load">刷新</el-button>
    </div>
    <div class="panel">
      <el-table :data="pagedItems" v-loading="loading">
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
      <ListPagination v-model:page="page" v-model:page-size="pageSize" :total="items.length" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref, watch } from 'vue'
import { client } from '../api/client'
import ListPagination from '../components/ListPagination.vue'
import { formatDateTime } from '../features/time'

const items = ref<any[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const pagedItems = computed(() => items.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))

async function load() {
  loading.value = true
  try {
    items.value = (await client.get('/audit-logs')).data || []
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

watch(pageSize, () => { page.value = 1 })
onMounted(load)
</script>
