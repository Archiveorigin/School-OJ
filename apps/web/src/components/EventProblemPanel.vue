<template>
  <div class="event-problem-layout">
    <section class="panel statement-panel">
      <div class="problem-toolbar"><h1>{{ selected.label }}.{{ selected.problem.title }}</h1><slot name="actions" /></div>
      <ProblemStatementView embedded :problem="selected.problem" :show-meta="false" :show-title="false" :asset-base="assetBase" />
    </section>
    <aside class="problem-side">
      <section class="panel navigator"><h3>题目导航</h3><div>
        <router-link v-for="item in items" :key="item.problem_id" :class="{ active: item.problem_id === selected.problem_id }" :to="problemPath(base, item.label)">{{ item.label }}</router-link>
      </div></section>
      <section class="panel records">
        <div class="records-heading"><h3>本题提交</h3><el-button link @click="$emit('refresh')">刷新</el-button></div>
        <article v-for="row in records" :key="row.id">
          <StatusBadge :status="row.status" /><span>{{ languageLabel(row.language) }}</span>
          <el-button link type="primary" :aria-label="`查看提交 ${row.id}`" @click="$emit('view-submission', row.id)">查看</el-button>
          <time>{{ formatDateTime(row.created_at) }}</time>
        </article>
        <el-empty v-if="!records.length" :image-size="54" description="暂无提交" />
      </section>
    </aside>
  </div>
</template>
<script setup lang="ts">
import type { Submission } from '../api/client'
import type { EventProblem } from '../features/events/workspace'
import { problemPath } from '../features/events/workspace'
import { languageLabel } from '../features/submissions/presentation'
import { formatDateTime } from '../features/time'
import StatusBadge from './StatusBadge.vue'
import ProblemStatementView from './ProblemStatementView.vue'
defineProps<{ selected: EventProblem; items: EventProblem[]; records: Submission[]; base: string; assetBase?: string }>()
defineEmits<{ refresh: []; 'view-submission': [id: number] }>()
</script>
<style scoped>
.event-problem-layout { display: grid; grid-template-columns: minmax(0, 1fr) 300px; gap: 16px; }
.statement-panel, .navigator, .records { padding: 20px; min-width: 0; }
.problem-toolbar, .records-heading { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.problem-toolbar { padding-bottom: 16px; border-bottom: 1px solid var(--border); }
.problem-toolbar h1 { margin: 0; font-size: 25px; line-height: 1.4; overflow-wrap: anywhere; }
.problem-side { display: grid; align-content: start; gap: 12px; }
h3 { margin: 0; font-size: 16px; }
.navigator > div { display: flex; flex-wrap: wrap; gap: 7px; margin-top: 14px; }
.navigator a { min-width: 38px; padding: 8px; text-align: center; border: 1px solid var(--border); border-radius: 6px; }
.navigator a.active { color: #fff; border-color: var(--accent); background: var(--accent); }
.records article { display: grid; grid-template-columns: auto 1fr auto; align-items: center; gap: 6px; padding: 12px 0; border-bottom: 1px solid var(--border); }
.records time { grid-column: 1 / -1; color: var(--muted); font-size: 13px; }
@media (max-width: 880px) { .event-problem-layout { grid-template-columns: minmax(0, 1fr); } }
@media (max-width: 600px) { .statement-panel { padding: 14px; } .problem-toolbar { flex-wrap: wrap; } .problem-toolbar h1 { font-size: 22px; } }
</style>
