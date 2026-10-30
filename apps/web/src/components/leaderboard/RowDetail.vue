<template>
  <section class="row-detail-inner" :aria-label="`${row.name} 排名摘要`">
    <p>
      {{ row.name }} · {{ row.unranked ? "不参与排名" : `第 ${row.rank} 名` }} ·
      {{ row.solved }} {{ data.solvedLabel }} ·
      {{ row.metricDisplay ?? row.metric }} {{ data.metricLabel }} ·
      {{ row.submissions }} 次提交
    </p>
    <div class="row-problem-summary">
      <button
        v-for="problem in data.problems"
        :key="problem.id"
        type="button"
        @click.stop="$emit('open', problem)"
      >
        {{ problem.label }} ·
        {{ labels[row.results[String(problem.id)]?.status || "none"] }} ·
        {{ row.results[String(problem.id)]?.attempts || 0 }} 次
      </button>
    </div>
  </section>
</template>
<script setup lang="ts">
import type {
  LeaderboardRow,
  LeaderboardData,
  LeaderboardProblem,
} from "../../features/leaderboard/types";
defineProps<{ row: LeaderboardRow; data: LeaderboardData }>();
defineEmits<{ open: [problem: LeaderboardProblem] }>();
const labels = {
  accepted: "已通过",
  wrong: "未通过",
  pending: "评测中",
  frozen: "封榜中",
  none: "未提交",
};
</script>
<style scoped>
.row-detail-inner {
  overflow: hidden;
  min-height: 0;
  padding: 0 12px;
  color: var(--score-text, var(--text));
  font-size: 13px;
}
.row-detail-inner p {
  margin: 12px 0;
}
.row-problem-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding-bottom: 12px;
}
button {
  border: 1px solid var(--score-border, var(--border));
  background: var(--score-surface, var(--surface));
  color: inherit;
  border-radius: 4px;
  padding: 8px;
  cursor: pointer;
}
button:focus-visible {
  outline: 2px solid var(--el-color-primary);
}
</style>
