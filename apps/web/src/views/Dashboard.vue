<template>
  <section class="page">
    <div class="page-header">
      <h2>概览</h2>
      <el-button @click="load">刷新</el-button>
    </div>
    <el-row :gutter="16">
      <el-col :span="6" v-for="item in stats" :key="item.label">
        <div class="panel stat">
          <strong>{{ item.value }}</strong>
          <span class="muted">{{ item.label }}</span>
        </div>
      </el-col>
    </el-row>
    <div class="panel latest">
      <h3>最近提交</h3>
      <el-table :data="submissions" size="small">
        <el-table-column prop="id" label="ID" width="90" />
        <el-table-column prop="problem_id" label="题目" width="90" />
        <el-table-column prop="language" label="语言" width="110" />
        <el-table-column label="状态">
          <template #default="{ row }"><StatusBadge :status="row.status" /></template>
        </el-table-column>
        <el-table-column prop="score" label="分数" width="90" />
      </el-table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { client, type Submission } from '../api/client'
import StatusBadge from '../components/StatusBadge.vue'

const courses = ref<any[]>([])
const problems = ref<any[]>([])
const submissions = ref<Submission[]>([])
const stats = computed(() => [
  { label: '课程', value: courses.value.length },
  { label: '题目', value: problems.value.length },
  { label: '提交', value: submissions.value.length },
  { label: 'AC', value: submissions.value.filter((s) => s.status === 'accepted').length }
])

async function load() {
  const [c, p, s] = await Promise.all([
    client.get('/courses'),
    client.get('/problems'),
    client.get('/submissions')
  ])
  courses.value = c.data
  problems.value = p.data
  submissions.value = s.data
}

onMounted(load)
</script>

<style scoped>
.stat {
  display: grid;
  gap: 8px;
}

.stat strong {
  font-size: 28px;
}

.latest {
  margin-top: 16px;
}
</style>
