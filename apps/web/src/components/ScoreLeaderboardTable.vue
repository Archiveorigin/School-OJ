<template>
  <div
    class="score-table"
    data-scoreboard-branch="score"
    role="table"
    :style="{ '--problem-columns-width': `${data.problems.length * 109}px` }"
  >
    <div class="score-sticky-header" data-sticky-header="score">
      <div class="score-header-track" role="row" ref="headerTrack">
        <div class="score-metrics score-header-metrics">
          <div class="score-col score-rank-col">
            <div class="score-header-card" role="columnheader">
              <strong>排名</strong><small>Rank</small>
            </div>
          </div>
          <div class="score-col score-name-col">
            <div class="score-header-card" role="columnheader">
              <strong>姓名</strong><small>Name</small>
            </div>
          </div>
          <div class="score-col score-full-col">
            <div class="score-header-card" role="columnheader">
              <strong>满分</strong><small>Full</small>
            </div>
          </div>
          <div class="score-col score-total-col">
            <div class="score-header-card" role="columnheader">
              <strong>总分</strong><small>Score</small>
            </div>
          </div>
          <div class="score-problems">
            <div
              v-for="problem in data.problems"
              :key="problem.id"
              class="score-col score-problem-col"
            >
              <div
                class="score-problem-header"
                role="columnheader"
                :title="problem.title"
              >
                <span
                  class="problem-color"
                  :style="{ backgroundColor: problem.color }"
                ></span>
                <strong>{{ problem.label }}</strong>
                <small
                  >{{ problemStats(problem.id).full }}/{{
                    problemStats(problem.id).attempted
                  }}</small
                >
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
      @scroll.passive="syncHeader"
    >
      <TransitionGroup name="rank-reveal" tag="div" class="score-grid" role="rowgroup">
        <article
          v-for="row in rows"
          :key="row.id"
          class="score-row native-stripe-row"
          :data-award="awardTier(row)"
          role="row"
        >
          <div class="score-metrics score-row-metrics">
            <div class="score-col score-rank-col">
              <div
                class="score-stat score-rank-stat"
                :class="awardRankClass(row)"
              >
                {{ displayRank(row) }}
              </div>
            </div>
            <div class="score-col score-name-col">
              <div
                class="score-participant-name"
                role="cell"
                :class="awardRankClass(row)"
                :title="row.name"
              >
                {{ row.name }}
              </div>
            </div>
            <div class="score-col score-full-col">
              <div class="score-stat">{{ row.solved }}</div>
            </div>
            <div class="score-col score-total-col">
              <div class="score-stat score-total">
                {{ row.metricDisplay ?? row.metric }}
              </div>
            </div>
            <div class="score-problems">
              <div
                v-for="problem in data.problems"
                :key="problem.id"
                class="score-col score-problem-col"
              >
                <div
                  class="score-result"
                  :class="`result-${resultFor(row, problem.id).status}`"
                  :title="resultTitle(row, problem.id)"
                >
                  <span class="score-result-primary">{{
                    resultFor(row, problem.id).primary || problem.label
                  }}</span>
                  <template v-if="resultFor(row, problem.id).secondary">
                    <span class="score-separator">|</span>
                    <small>{{ resultFor(row, problem.id).secondary }}</small>
                  </template>
                </div>
              </div>
            </div>
          </div>
        </article>
      </TransitionGroup>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import type {
  LeaderboardData,
  LeaderboardResult,
  LeaderboardRow,
} from "../features/leaderboard/types";
import type { LeaderboardViewMode } from "../features/leaderboard/sorting";

type AwardTier = "gold" | "silver" | "bronze" | "none";

const props = withDefaults(
  defineProps<{
    data: LeaderboardData;
    rows: LeaderboardRow[];
    mode: LeaderboardViewMode;
    rankByID?: Record<string, number>;
    awardByID?: Record<string, AwardTier>;
  }>(),
  {
    rankByID: () => ({}),
    awardByID: () => ({}),
  },
);
const headerTrack = ref<HTMLElement | null>(null);
function syncHeader(event: Event) {
  // Scroll only changes this DOM transform, not the whole results grid.
  if (headerTrack.value)
    headerTrack.value.style.transform = `translate3d(-${(event.currentTarget as HTMLElement).scrollLeft}px, 0, 0)`;
}

function displayRank(row: LeaderboardRow) {
  return row.unranked ? '*' : (props.rankByID[String(row.id)] ?? row.rank);
}

function awardTier(row: LeaderboardRow): AwardTier {
  return row.unranked ? "none" : (props.awardByID[String(row.id)] ?? "none");
}

function awardRankClass(row: LeaderboardRow) {
  const tier = awardTier(row);
  return tier === "none" ? "" : `is-${tier}`;
}

function resultFor(
  row: LeaderboardRow,
  problemID: number | string,
): LeaderboardResult {
  return row.results[String(problemID)] || emptyResult;
}

const emptyResult: LeaderboardResult = { status: "none", attempts: 0 };
const emptyStats = { attempted: 0, full: 0 };
const statsByProblem = computed(() => {
  const stats = new Map<string, { attempted: number; full: number }>();
  for (const row of props.data.rows) {
    for (const [id, result] of Object.entries(row.results)) {
      const count = stats.get(id) || { attempted: 0, full: 0 };
      if (result.attempts > 0) count.attempted++;
      if (result.status === "accepted") count.full++;
      stats.set(id, count);
    }
  }
  return stats;
});
function problemStats(problemID: number | string) {
  return statsByProblem.value.get(String(problemID)) || emptyStats;
}

function resultTitle(row: LeaderboardRow, problemID: number | string) {
  const result = resultFor(row, problemID);
  const status = {
    accepted: "满分",
    wrong: "未满分",
    pending: "待评分",
    frozen: "已封榜",
    none: "未提交",
  }[result.status];
  return [status, result.primary].filter(Boolean).join(" · ");
}
</script>

<style scoped>
.rank-reveal-move { transition: transform 700ms ease; }
@media (prefers-reduced-motion: reduce) { .rank-reveal-move { transition: none; } }

.score-table {
  --name-width: 180px;
  --rank-width: 86px;
  --full-width: 96px;
  --total-width: 122px;
  --problem-width: 104px;
  --table-min-width: max(
    760px,
    calc(
      var(--rank-width) + var(--name-width) + 5px + var(--full-width) +
        var(--total-width) + var(--problem-columns-width) + 42px
    )
  );
  --score-row-odd: var(--score-card);
  --score-row-even: var(--score-surface);
  width: 100%;
  min-width: 0;
}
.score-table,
.score-table * {
  box-sizing: border-box;
}
.score-sticky-header {
  position: sticky;
  top: 0;
  z-index: 20;
  width: 100%;
  height: 73px;
  overflow: clip;
  clip-path: inset(0);
  border-bottom: 1px solid var(--score-border);
  background: var(--score-header-bg, #f7f8fa);
  box-shadow: 0 3px 10px rgba(15, 23, 42, 0.08);
}
.score-header-track {
  width: var(--table-min-width);
  min-width: 100%;
  height: 73px;
  display: flex;
  align-items: end;
  padding: 12px 16px 8px;
  will-change: transform;
}
.score-body-scroller {
  width: 100%;
  min-width: 0;
  overflow-x: auto;
  overscroll-behavior-inline: contain;
}
.score-body-scroller:focus-visible {
  outline: 2px solid var(--score-blue);
  outline-offset: -2px;
}
.score-grid {
  width: var(--table-min-width);
  min-width: 100%;
  padding: 0 16px 24px;
}
.score-row {
  --score-row-background: var(--score-row-odd);
  position: relative;
  width: 100%;
  height: 64px;
  display: grid;
  grid-template-rows: 1fr;
  overflow: hidden;
  border-bottom: 1px solid var(--score-border);
  background: var(--score-row-background);
  transition: background-color 150ms ease;
}
.score-row:first-child {
  border-top: 1px solid var(--score-border);
}
.score-row:nth-child(even) {
  --score-row-background: var(--score-row-even);
}
.score-row:hover {
  --score-row-background: color-mix(
    in srgb,
    var(--score-blue) 6%,
    var(--score-card)
  );
}
.score-name-col {
  width: var(--name-width);
}
.score-participant-name {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 40px;
  padding: 0 10px;
  border: 1px solid var(--score-border);
  border-radius: 4px;
  color: var(--score-text);
  background: var(--score-surface);
  font-size: 14px;
  font-weight: 700;
  line-height: 1.3;
  overflow: hidden;
  overflow-wrap: anywhere;
}
.score-metrics {
  display: flex;
  align-items: center;
  gap: 5px;
  min-width: max-content;
}
.score-header-metrics {
  height: 52px;
  align-self: end;
}
.score-col {
  flex: none;
  text-align: center;
}
.score-rank-col {
  width: var(--rank-width);
}
.score-full-col {
  width: var(--full-width);
}
.score-total-col {
  width: var(--total-width);
}
.score-problem-col {
  width: var(--problem-width);
}
.score-problems {
  display: flex;
  gap: 5px;
}
.score-header-card,
.score-problem-header {
  height: 52px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 4px;
  color: #fff;
  background: #182235;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.22);
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.28);
}
.score-header-card strong {
  font-size: 15px;
  font-weight: 900;
  line-height: 1.1;
}
.score-header-card small {
  margin-top: 3px;
  color: #cbd5e1;
  font-size: 10px;
  font-weight: 700;
  line-height: 1;
}
.score-problem-header {
  position: relative;
  overflow: hidden;
}
.score-problem-header .problem-color {
  position: absolute;
  inset: 0 0 auto;
  height: 4px;
}
.score-problem-header strong {
  font-size: 18px;
  line-height: 1.05;
}
.score-problem-header small {
  margin-top: 3px;
  color: #cbd5e1;
  font-size: 10px;
}
.score-row-metrics {
  position: relative;
  z-index: 1;
  height: 64px;
  align-items: center;
}
.score-stat,
.score-result {
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  font-size: 17px;
  font-weight: 900;
}
.score-stat {
  border: 1px solid var(--score-card-border, #d9e0e8);
  background: var(--score-surface);
}
.score-total {
  padding: 0 6px;
  font-size: 16px;
}
.score-rank-stat.is-gold,
.score-participant-name.is-gold {
  border-color: #ff8f00;
  color: #1a1a1a;
  background: linear-gradient(135deg, #ffd700, #ffb300);
  box-shadow: 0 2px 8px rgba(255, 179, 0, 0.4);
}
.score-rank-stat.is-silver,
.score-participant-name.is-silver {
  border-color: #909090;
  color: #1a1a1a;
  background: linear-gradient(135deg, #d5d5d5, #a8a8a8);
  box-shadow: 0 2px 8px rgba(144, 144, 144, 0.32);
}
.score-rank-stat.is-bronze,
.score-participant-name.is-bronze {
  border-color: #9d5f1f;
  color: #fff;
  background: linear-gradient(135deg, #cd7f32, #b87333);
  box-shadow: 0 2px 8px rgba(157, 95, 31, 0.32);
}
.score-result {
  width: 104px;
  gap: 4px;
  border: 1px solid transparent;
  color: #fff;
  transition:
    transform 150ms ease,
    box-shadow 150ms ease,
    filter 150ms ease;
}
.score-result:hover {
  transform: translateY(-1px);
  filter: saturate(1.1);
  box-shadow: 0 3px 8px rgba(15, 23, 42, 0.22);
}
.score-result.result-accepted {
  border-color: #146c43;
  background: #198754;
}
.score-result.result-wrong {
  border-color: #b02a37;
  background: #dc3545;
}
.score-result.result-none {
  border-color: var(--score-card-border);
  color: var(--score-muted);
  background: var(--score-header-bg);
}
.score-result.result-pending,
.score-result.result-frozen {
  border-color: #0a58ca;
  background: #0d6efd;
}
.score-result-primary {
  white-space: nowrap;
}
.score-result small {
  font-size: 10px;
  white-space: nowrap;
}
.score-separator {
  opacity: 0.74;
}
.scoreboard-core[data-theme="dark"] .score-table {
  --score-row-odd: #101827;
  --score-row-even: #151f30;
}
.scoreboard-core[data-theme="dark"] .score-row {
  border-color: #334155;
}
.scoreboard-core[data-theme="dark"] .score-row:hover {
  --score-row-background: #1c2b42;
}
.scoreboard-core[data-theme="macaron"] .score-table {
  --score-row-odd: #fffdfd;
  --score-row-even: #faf3f8;
}
@media (prefers-reduced-motion: reduce) {
  .score-result,
  .score-row {
    transition: none;
  }
}
</style>
