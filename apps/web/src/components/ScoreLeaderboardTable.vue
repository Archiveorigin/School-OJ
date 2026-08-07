<template>
  <div
    class="score-table"
    data-scoreboard-branch="score"
    role="table"
    :style="{ '--problem-columns-width': `${data.problems.length * 112}px` }"
  >
    <div class="score-sticky-header" data-sticky-header="score">
      <div class="score-header-track" role="row" :style="headerTrackStyle">
        <div class="score-identity-header" role="columnheader">学生 / 学号</div>
        <div class="score-metrics score-header-metrics">
          <div class="score-col score-rank-col"><div class="score-header-card" role="columnheader">排名</div></div>
          <div class="score-col score-full-col"><div class="score-header-card" role="columnheader">满分</div></div>
          <div class="score-col score-total-col"><div class="score-header-card" role="columnheader">总分</div></div>
          <div class="score-problems">
            <div v-for="problem in data.problems" :key="problem.id" class="score-col score-problem-col">
              <div class="score-problem-header" role="columnheader" :title="problem.title">
                <span class="problem-color" :style="{ backgroundColor: problem.color }"></span>
                <strong>{{ problem.label }}</strong>
                <small>{{ problemStats(problem.id).full }}/{{ problemStats(problem.id).attempted }}</small>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div
      class="score-body-scroller"
      data-horizontal-scroll="score"
      tabindex="0"
      aria-label="总分榜横向滚动区域"
      @scroll="syncHeader"
    >
      <div class="score-grid" role="rowgroup">
        <article v-for="(row, index) in rows" :key="row.id" class="score-row" role="row">
          <LeaderboardStudentIdentity :row="row" variant="score" />
          <div class="score-metrics score-row-metrics">
            <div class="score-col score-rank-col"><div class="score-stat">{{ mode === 'performance' ? index + 1 : row.rank }}</div></div>
            <div class="score-col score-full-col"><div class="score-stat">{{ row.solved }}</div></div>
            <div class="score-col score-total-col"><div class="score-stat score-total">{{ row.metricDisplay ?? row.metric }}</div></div>
            <div class="score-problems">
              <div v-for="problem in data.problems" :key="problem.id" class="score-col score-problem-col">
                <div
                  class="score-result"
                  :class="`result-${resultFor(row, problem.id).status}`"
                  :title="resultTitle(row, problem.id)"
                >
                  <span class="score-result-primary">{{ resultFor(row, problem.id).primary || problem.label }}</span>
                  <template v-if="resultFor(row, problem.id).secondary">
                    <span class="score-separator">|</span>
                    <small>{{ resultFor(row, problem.id).secondary }}</small>
                  </template>
                </div>
              </div>
            </div>
          </div>
        </article>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import LeaderboardStudentIdentity from './LeaderboardStudentIdentity.vue'
import type { LeaderboardData, LeaderboardResult, LeaderboardRow } from '../features/leaderboard/types'
import type { LeaderboardViewMode } from '../features/leaderboard/sorting'

const props = defineProps<{ data: LeaderboardData, rows: LeaderboardRow[], mode: LeaderboardViewMode }>()
const horizontalOffset = ref(0)
const headerTrackStyle = computed(() => ({ transform: `translate3d(-${horizontalOffset.value}px, 0, 0)` }))

function syncHeader(event: Event) {
  horizontalOffset.value = (event.currentTarget as HTMLElement).scrollLeft
}

function resultFor(row: LeaderboardRow, problemID: number | string): LeaderboardResult {
  return row.results[String(problemID)] || { status: 'none', attempts: 0 }
}

function problemStats(problemID: number | string) {
  let attempted = 0
  let full = 0
  for (const row of props.data.rows) {
    const result = resultFor(row, problemID)
    if (result.attempts > 0) attempted += 1
    if (result.status === 'accepted') full += 1
  }
  return { attempted, full }
}

function resultTitle(row: LeaderboardRow, problemID: number | string) {
  const result = resultFor(row, problemID)
  const status = { accepted: '满分', wrong: '未满分', pending: '待评分', frozen: '已封榜', none: '未提交' }[result.status]
  return [status, result.primary].filter(Boolean).join(' · ')
}
</script>

<style scoped>
.score-table {
  --identity-width: 560px;
  --rank-width: 86px;
  --full-width: 96px;
  --total-width: 122px;
  --problem-width: 104px;
  --table-min-width: calc(
    var(--identity-width) + var(--rank-width) + var(--full-width) + var(--total-width) +
    var(--problem-columns-width) + 48px
  );
  width: 100%;
  min-width: 0;
}
.score-sticky-header {
  position: sticky;
  top: 0;
  z-index: 20;
  width: 100%;
  height: 79px;
  overflow: clip;
  clip-path: inset(0);
  border-bottom: 8px solid var(--score-header-separator, #e8edf3);
  background: var(--score-header-bg, #f7f8fa);
  box-shadow: 0 2px 8px rgba(15, 23, 42, .05);
}
.score-header-track {
  width: var(--table-min-width);
  min-width: 100%;
  height: 71px;
  display: grid;
  grid-template-columns: var(--identity-width) max-content;
  align-items: end;
  padding: 5px 16px 8px;
  will-change: transform;
}
.score-body-scroller {
  width: 100%;
  min-width: 0;
  overflow-x: auto;
  overscroll-behavior-inline: contain;
}
.score-body-scroller:focus-visible { outline: 2px solid var(--score-blue); outline-offset: -2px; }
.score-grid {
  width: var(--table-min-width);
  min-width: 100%;
  display: grid;
  gap: 7px;
  padding: 7px 16px 26px;
}
.score-row {
  width: 100%;
  min-height: 118px;
  display: grid;
  grid-template-columns: var(--identity-width) max-content;
  overflow: hidden;
  border: 1px solid var(--score-border);
  border-radius: 7px;
  background: var(--score-card);
  box-shadow: 0 1px 3px rgba(15, 23, 42, .05);
}
.score-row:hover { border-color: #4f8df5; box-shadow: 0 5px 16px rgba(34, 80, 148, .13); }
.score-identity-header { height: 58px; display: flex; align-items: center; padding: 0 22px; color: var(--score-muted); font-size: 15px; font-weight: 800; }
.score-metrics { display: flex; align-items: center; gap: 8px; min-width: max-content; }
.score-header-metrics { height: 58px; align-self: end; }
.score-col { flex: none; text-align: center; }
.score-rank-col { width: var(--rank-width); }
.score-full-col { width: var(--full-width); }
.score-total-col { width: var(--total-width); }
.score-problem-col { width: var(--problem-width); }
.score-problems { display: flex; gap: 8px; }
.score-header-card,
.score-problem-header {
  height: 58px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--score-card-border, #d9e0e8);
  border-radius: 6px;
  color: var(--score-text);
  background: var(--score-card);
  font-size: 15px;
  font-weight: 900;
}
.score-problem-header { position: relative; overflow: hidden; }
.score-problem-header .problem-color { position: absolute; inset: 0 0 auto; height: 4px; }
.score-problem-header strong { font-size: 22px; line-height: 1.05; }
.score-problem-header small { margin-top: 5px; color: var(--score-muted); font-size: 12px; }
.score-row-metrics { height: 118px; align-items: center; }
.score-stat,
.score-result {
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  font-size: 17px;
  font-weight: 900;
}
.score-stat { border: 1px solid var(--score-card-border, #d9e0e8); background: var(--score-surface); }
.score-total { padding: 0 6px; font-size: 16px; }
.score-result {
  width: 104px;
  gap: 4px;
  border: 1px solid transparent;
  color: #fff;
  transition: transform 150ms ease, box-shadow 150ms ease, filter 150ms ease;
}
.score-result:hover { transform: translateY(-1px); filter: saturate(1.1); box-shadow: 0 3px 8px rgba(15, 23, 42, .22); }
.score-result.result-accepted { border-color: #146c43; background: #198754; }
.score-result.result-wrong { border-color: #b02a37; background: #dc3545; }
.score-result.result-none { border-color: #e9ecef; color: #26313f; background: #adb5bd; }
.score-result.result-pending,
.score-result.result-frozen { border-color: #0a58ca; background: #0d6efd; }
.score-result-primary { white-space: nowrap; }
.score-result small { font-size: 10px; white-space: nowrap; }
.score-separator { opacity: .74; }
@media (max-width: 991.98px) { .score-table { --identity-width: 500px; } }
@media (max-width: 576px) { .score-table { --identity-width: 420px; } }
@media (prefers-reduced-motion: reduce) { .score-result { transition: none; } }
</style>
