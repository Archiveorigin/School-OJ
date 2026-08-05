<template>
  <section class="panel problem-overview">
    <header>
      <div>
        <span class="eyebrow">PROBLEM OVERVIEW</span>
        <h2>题目概览</h2>
      </div>
      <div class="legend" aria-label="题目状态说明">
        <el-tag type="success" effect="light">已通过</el-tag>
        <el-tag type="danger" effect="light">已提交未通过</el-tag>
      </div>
    </header>
    <div class="overview-table-wrap">
      <table>
        <thead><tr><th>题号</th><th>满分</th><th>题目</th></tr></thead>
        <tbody>
          <tr
            v-for="(item, index) in items"
            :key="item.problem.id"
            :class="[statusClass(item.submission_status), { active: item.problem.id === activeProblemId }]"
            @click="emit('select', item.problem.id)"
          >
            <td>{{ item.label || defaultProblemLabel(index) }}</td>
            <td>{{ item.score ?? 100 }}</td>
            <td><button type="button" @click.stop="emit('select', item.problem.id)">{{ item.problem.title }}</button></td>
          </tr>
        </tbody>
      </table>
    </div>
    <el-empty v-if="!items.length" description="暂无题目" />
  </section>
</template>

<script setup lang="ts">
import type { Problem } from '../api/client'

defineProps<{
  items: Array<{ problem: Problem; label?: string; score?: number; submission_status?: string }>
  activeProblemId?: number
}>()
const emit = defineEmits<{ select: [problemID: number] }>()

function statusClass(status?: string) {
  if (status === 'accepted') return 'accepted'
  return status ? 'attempted' : ''
}

function defaultProblemLabel(index: number) {
  index += 1
  let label = ''
  while (index > 0) {
    index -= 1
    label = String.fromCharCode(65 + (index % 26)) + label
    index = Math.floor(index / 26)
  }
  return label
}
</script>

<style scoped>
.problem-overview { width: min(1120px, 100%); margin: 0 auto; padding: 22px 26px; }
.problem-overview > header { display: flex; align-items: end; justify-content: space-between; gap: 20px; margin-bottom: 18px; }
.eyebrow { color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: .14em; }
h2 { margin: 5px 0 0; font-size: 23px; }
.legend { display: flex; align-items: center; flex-wrap: wrap; gap: 14px; color: var(--muted); font-size: 12px; }
.overview-table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: 16px; }
th, td { padding: 16px 18px; border-bottom: 1px solid var(--border); text-align: left; }
th { color: var(--text); background: color-mix(in srgb, var(--app-bg) 70%, var(--surface-strong)); font-size: 14px; }
th:first-child, td:first-child { width: 88px; }
th:nth-child(2), td:nth-child(2) { width: 100px; }
tbody tr { cursor: pointer; transition: background .16s ease, box-shadow .16s ease; }
tbody tr:hover { background: color-mix(in srgb, var(--accent) 7%, var(--surface-strong)); }
tbody tr.accepted { background: #eaf8ef; }
tbody tr.attempted { background: #fff0f0; }
tbody tr.active { box-shadow: inset 4px 0 var(--accent); }
td button { padding: 0; color: #2494dd; border: 0; background: transparent; font: inherit; text-align: left; cursor: pointer; }
:global(.dark) tbody tr.accepted { background: color-mix(in srgb, #16a34a 22%, var(--surface-strong)); }
:global(.dark) tbody tr.attempted { background: color-mix(in srgb, #ef4444 20%, var(--surface-strong)); }
@media (max-width: 680px) {
  .problem-overview { padding: 16px 12px; }
  .problem-overview > header { align-items: flex-start; flex-direction: column; }
  th, td { padding: 13px 10px; }
}
</style>
