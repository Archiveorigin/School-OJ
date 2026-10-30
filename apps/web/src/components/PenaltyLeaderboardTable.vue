<template>
  <div
    class="penalty-table"
    data-scoreboard-branch="penalty"
    role="table"
    :style="{ '--problem-columns-width': `${data.problems.length * 85}px` }"
  >
    <div class="penalty-sticky-header" data-sticky-header="penalty">
      <div class="penalty-header-track" role="row" ref="headerTrack">
        <div class="penalty-metrics penalty-header-metrics">
          <div class="penalty-col penalty-rank-col">
            <div
              class="penalty-header-card native-header-card"
              role="columnheader"
            >
              <strong>排名</strong><small>Rank</small>
            </div>
          </div>
          <div class="penalty-col penalty-name-col">
            <div class="penalty-header-card" role="columnheader">
              <strong>姓名</strong><small>Name</small>
            </div>
          </div>
          <div class="penalty-col penalty-solved-col">
            <div
              class="penalty-header-card native-header-card"
              role="columnheader"
            >
              <strong>题数</strong><small>Solved</small>
            </div>
          </div>
          <div class="penalty-col penalty-total-col">
            <div
              class="penalty-header-card native-header-card"
              role="columnheader"
            >
              <strong>罚时</strong><small>Penalty</small>
            </div>
          </div>
          <div class="penalty-problems">
            <div
              v-for="problem in data.problems"
              :key="problem.id"
              class="penalty-col penalty-problem-col"
            >
              <div
                class="penalty-problem-header native-header-card"
                role="columnheader"
                :title="problem.title"
              >
                <span
                  class="problem-color"
                  :style="{ backgroundColor: problem.color }"
                ></span>
                <strong>{{ problem.label }}</strong>
                <small
                  >{{ problemStats(problem.id).solved }}/{{
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
      class="penalty-body-scroller"
      data-horizontal-scroll="penalty"
      tabindex="0"
      aria-label="罚时榜横向滚动区域"
      @scroll.passive="syncHeader"
    >
      <TransitionGroup name="rank-reveal" tag="div" class="penalty-grid" role="rowgroup">
        <article
          v-for="row in rows"
          :key="row.id"
          class="penalty-row native-stripe-row"
          :data-award="awardTier(row)"
          role="row"
        >
          <div class="penalty-metrics penalty-row-metrics">
            <div class="penalty-col penalty-rank-col">
              <div
                class="penalty-stat penalty-rank-stat"
                :class="awardRankClass(row)"
              >
                {{ displayRank(row) }}
              </div>
            </div>
            <div class="penalty-col penalty-name-col">
              <div
                class="penalty-participant-name"
                role="cell"
                :class="awardRankClass(row)"
                :title="row.name"
              >
                {{ row.name }}
              </div>
            </div>
            <div class="penalty-col penalty-solved-col">
              <div class="penalty-stat penalty-solved-stat">
                {{ row.solved }}
              </div>
            </div>
            <div class="penalty-col penalty-total-col">
              <div class="penalty-stat penalty-time-stat">
                {{ row.metricDisplay ?? row.metric }}
              </div>
            </div>
            <div class="penalty-problems">
              <div
                v-for="problem in data.problems"
                :key="problem.id"
                class="penalty-col penalty-problem-col"
              >
                <div
                  class="penalty-result"
                  :class="[
                    `result-${resultFor(row, problem.id).status}`,
                    { 'is-first': resultFor(row, problem.id).firstBlood },
                  ]"
                  :title="resultTitle(row, problem.id)"
                >
                  <span
                    v-if="resultFor(row, problem.id).firstBlood"
                    class="first-blood"
                    aria-label="最快通过"
                    >★</span
                  >
                  <span>{{
                    resultFor(row, problem.id).primary || problem.label
                  }}</span>
                  <template v-if="resultFor(row, problem.id).secondary">
                    <span class="result-separator">|</span>
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
const emptyStats = { attempted: 0, solved: 0 };
const statsByProblem = computed(() => {
  const stats = new Map<string, { attempted: number; solved: number }>();
  for (const row of props.data.rows) {
    for (const [id, result] of Object.entries(row.results)) {
      const count = stats.get(id) || { attempted: 0, solved: 0 };
      if (result.attempts > 0) count.attempted++;
      if (result.status === "accepted") count.solved++;
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
    accepted: "已通过",
    wrong: "未通过",
    pending: "评测中",
    frozen: "已封榜",
    none: "未提交",
  }[result.status];
  return [status, result.primary, result.secondary].filter(Boolean).join(" · ");
}
</script>

<style scoped>
.rank-reveal-move { transition: transform 700ms ease; }
@media (prefers-reduced-motion: reduce) { .rank-reveal-move { transition: none; } }

.penalty-table {
  --name-width: 180px;
  --rank-width: 64px;
  --solved-width: 72px;
  --penalty-width: 88px;
  --problem-width: 80px;
  --table-min-width: max(
    720px,
    calc(
      var(--rank-width) + var(--name-width) + 5px + var(--solved-width) +
        var(--penalty-width) + var(--problem-columns-width) + 42px
    )
  );
  --penalty-row-odd: var(--score-card);
  --penalty-row-even: var(--score-surface);
  width: 100%;
  min-width: 0;
}
.penalty-table,
.penalty-table * {
  box-sizing: border-box;
}
.penalty-sticky-header {
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
.penalty-header-track {
  width: var(--table-min-width);
  min-width: 100%;
  height: 73px;
  display: flex;
  align-items: end;
  padding: 12px 16px 8px;
  will-change: transform;
}
.penalty-body-scroller {
  width: 100%;
  min-width: 0;
  overflow-x: auto;
  overscroll-behavior-inline: contain;
}
.penalty-body-scroller:focus-visible {
  outline: 2px solid var(--score-blue);
  outline-offset: -2px;
}
.penalty-grid {
  width: var(--table-min-width);
  min-width: 100%;
  padding: 0 16px 24px;
}
.penalty-row {
  --penalty-row-background: var(--penalty-row-odd);
  position: relative;
  width: 100%;
  height: 64px;
  display: grid;
  grid-template-rows: 1fr;
  overflow: hidden;
  border-bottom: 1px solid var(--score-border);
  background: var(--penalty-row-background);
  transition: background-color 150ms ease;
}
.penalty-row:first-child {
  border-top: 1px solid var(--score-border);
}
.penalty-row:nth-child(even) {
  --penalty-row-background: var(--penalty-row-even);
}
.penalty-row:hover {
  --penalty-row-background: color-mix(
    in srgb,
    var(--score-blue) 6%,
    var(--score-card)
  );
}
.penalty-name-col {
  width: var(--name-width);
}
.penalty-participant-name {
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
.penalty-metrics {
  display: flex;
  align-items: center;
  gap: 5px;
  min-width: max-content;
}
.penalty-header-metrics {
  height: 52px;
  align-self: end;
}
.penalty-col {
  flex: none;
  text-align: center;
}
.penalty-rank-col {
  width: var(--rank-width);
}
.penalty-solved-col {
  width: var(--solved-width);
}
.penalty-total-col {
  width: var(--penalty-width);
}
.penalty-problem-col {
  width: var(--problem-width);
}
.penalty-problems {
  display: flex;
  gap: 5px;
}
.penalty-header-card,
.penalty-problem-header {
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
.penalty-header-card strong {
  font-size: 15px;
  font-weight: 900;
  line-height: 1.1;
}
.penalty-header-card small {
  margin-top: 3px;
  color: #cbd5e1;
  font-size: 10px;
  font-weight: 700;
  line-height: 1;
}
.penalty-problem-header {
  position: relative;
  overflow: hidden;
}
.penalty-problem-header .problem-color {
  position: absolute;
  inset: 0 0 auto;
  height: 4px;
}
.penalty-problem-header strong {
  font-size: 18px;
}
.penalty-row-metrics {
  position: relative;
  z-index: 1;
  height: 64px;
  align-items: center;
}
.penalty-stat,
.penalty-result {
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  font-size: 16px;
  font-weight: 900;
  font-variant-numeric: tabular-nums;
}
.penalty-rank-stat {
  border: 1px solid var(--score-border);
  color: var(--score-text);
  background: var(--score-surface);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.8);
}
.penalty-rank-stat.is-gold,
.penalty-participant-name.is-gold {
  border-color: #ff8f00;
  color: #1a1a1a;
  background: linear-gradient(135deg, #ffd700 0%, #ffb300 100%);
  box-shadow:
    0 2px 8px rgba(255, 179, 0, 0.45),
    inset 0 1px 0 rgba(255, 255, 255, 0.4);
}
.penalty-rank-stat.is-silver,
.penalty-participant-name.is-silver {
  border-color: #909090;
  color: #1a1a1a;
  background: linear-gradient(135deg, #d5d5d5 0%, #a8a8a8 100%);
  box-shadow:
    0 2px 8px rgba(144, 144, 144, 0.35),
    inset 0 1px 0 rgba(255, 255, 255, 0.35);
}
.penalty-rank-stat.is-bronze,
.penalty-participant-name.is-bronze {
  border-color: #9d5f1f;
  color: #fff;
  background: linear-gradient(135deg, #cd7f32 0%, #b87333 100%);
  box-shadow:
    0 2px 8px rgba(157, 95, 31, 0.35),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
}
.penalty-solved-stat {
  border: 1px solid rgba(14, 165, 233, 0.35);
  color: var(--score-text);
  background: color-mix(in srgb, var(--color-info) 10%, var(--score-card));
}
.penalty-time-stat {
  padding: 0 4px;
  border: 1px solid rgba(234, 88, 12, 0.35);
  color: var(--score-text);
  background: color-mix(in srgb, var(--color-warning) 10%, var(--score-card));
  font-size: 14px;
}
.penalty-result {
  position: relative;
  width: 80px;
  gap: 3px;
  border: 1px solid transparent;
  color: #fff;
  transition:
    transform 150ms ease,
    box-shadow 150ms ease,
    filter 150ms ease;
}
.penalty-result:hover {
  transform: translateY(-1px);
  filter: saturate(1.1);
  box-shadow: 0 3px 8px rgba(15, 23, 42, 0.22);
}
.penalty-result.result-accepted {
  border-color: #146c43;
  background: #198754;
}
.penalty-result.result-wrong {
  border-color: #b02a37;
  background: #dc3545;
}
.penalty-result.result-none {
  border-color: var(--score-card-border);
  color: var(--score-muted);
  background: var(--score-header-bg);
}
.penalty-result.result-pending,
.penalty-result.result-frozen {
  border-color: #0a58ca;
  background: #0d6efd;
}
.penalty-result.result-frozen {
  background-image: repeating-linear-gradient(
    135deg,
    transparent 0 7px,
    rgba(255, 255, 255, 0.16) 7px 12px
  );
}
.penalty-result small {
  font-size: 11px;
}
.result-separator {
  opacity: 0.72;
}
.first-blood {
  position: absolute;
  top: -7px;
  right: 1px;
  color: #ffd43b;
  font-size: 14px;
  line-height: 1;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.55);
}
.scoreboard-core[data-theme="dark"] .penalty-table {
  --penalty-row-odd: #101827;
  --penalty-row-even: #151f30;
}
.scoreboard-core[data-theme="dark"] .penalty-row {
  border-color: #334155;
}
.scoreboard-core[data-theme="dark"] .penalty-row:hover {
  --penalty-row-background: #1c2b42;
}
.scoreboard-core[data-theme="macaron"] .penalty-table {
  --penalty-row-odd: #fffdfd;
  --penalty-row-even: #faf3f8;
}
@media (prefers-reduced-motion: reduce) {
  .penalty-result,
  .penalty-row {
    transition: none;
  }
}
</style>
