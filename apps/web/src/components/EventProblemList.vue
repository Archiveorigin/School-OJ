<template>
  <section class="panel event-problems">
    <div class="list-heading"><h2>题目列表</h2><slot name="actions" /></div>
    <div class="table-wrap"><table>
      <thead><tr><th>题号</th><th>题目名称</th><th>满分</th><th>我的状态</th></tr></thead>
      <tbody><tr v-for="item in items" :key="item.problem_id">
        <td><router-link :to="problemPath(base, item.label)">{{ item.label }}</router-link></td>
        <td><router-link :to="problemPath(base, item.label)">{{ item.problem.title }}</router-link></td>
        <td>{{ item.score }}</td>
        <td><StatusBadge v-if="item.submission_status" :status="item.submission_status" /><span v-else>未提交</span></td>
      </tr></tbody>
    </table></div>
    <el-empty v-if="!items.length" description="暂无题目" />
  </section>
</template>
<script setup lang="ts">
import type { EventProblem } from '../features/events/workspace'
import { problemPath } from '../features/events/workspace'
import StatusBadge from './StatusBadge.vue'
defineProps<{ items: EventProblem[]; base: string }>()
</script>
<style scoped>
.event-problems { padding: 20px; min-width: 0; }
.list-heading { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
h2 { margin: 0; font-size: 20px; }
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; margin-top: 16px; }
th, td { padding: 14px 16px; text-align: left; border-bottom: 1px solid var(--border); }
th { color: var(--muted); font-size: 13px; background: var(--surface-strong); white-space: nowrap; }
a { color: var(--text); font-weight: 700; }
td:first-child a { color: var(--accent); }
@media (max-width: 600px) { .event-problems { padding: 12px; } th, td { padding: 12px 7px; } td:nth-child(2) { min-width: 120px; } }
</style>
