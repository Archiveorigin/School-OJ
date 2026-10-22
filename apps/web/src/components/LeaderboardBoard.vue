<template>
  <section
    ref="root"
    class="scoreboard-core"
    :data-theme="theme"
    :aria-busy="loading"
    :data-scoring-rule="data.scoringRule"
  >
    <div class="scoreboard-shell" :data-theme="theme" :data-rank-mode="mode">
      <header class="rank-header">
        <div class="scoreboard-title">
          <span class="scoreboard-title-icon" aria-hidden="true"
            ><Histogram
          /></span>
          <span class="scoreboard-title-text">{{ data.title }}</span>
          <span v-if="data.frozen" class="scoreboard-freeze" role="status"
            >排行榜已封榜</span
          >
          <span v-if="data.subtitle" class="scoreboard-subtitle">{{
            data.subtitle
          }}</span>
        </div>

        <div class="controls-time-section">
          <div class="controls-time-container">
            <div class="score-control-field">
              <span class="score-control-label">排名模式</span
              ><strong class="rank-mode">{{
                data.finalReady ? "最终排名" : "实时表现"
              }}</strong>
            </div>

            <div class="time-progress-wrapper">
              <div class="time-display-group">
                <span class="time-current">{{ formatClock(currentTime) }}</span>
                <span class="time-total">{{
                  formatClock(data.durationSeconds)
                }}</span>
              </div>
              <input
                v-model.number="currentTime"
                class="time-progress-slider"
                type="range"
                min="0"
                :max="Math.max(0, data.durationSeconds)"
                step="1"
                aria-label="比赛时间"
                :style="{ '--progress': `${progress}%` }"
              />
              <button
                type="button"
                class="time-reset-btn"
                title="回到最新时间"
                aria-label="回到最新时间"
                @click="resetTime"
              >
                <RefreshRight />
              </button>
            </div>

            <div class="score-filter-group">
              <input
                v-model="query"
                class="score-query-input"
                type="search"
                placeholder="筛选姓名…"
                aria-label="筛选姓名"
              />
              <label class="score-check"
                ><input v-model="attemptedOnly" type="checkbox" /><span
                  >有提交</span
                ></label
              >
              <label class="score-check"
                ><input v-model="solvedOnly" type="checkbox" /><span>{{
                  data.scoringRule === "score" ? "有满分" : "有通过"
                }}</span></label
              >
            </div>

            <div class="controls-toolbar" :class="{ 'is-open': moreOpen }">
              <select
                v-model="theme"
                class="score-theme-select"
                aria-label="榜单主题"
              >
                <option value="default">随页面</option>
                <option value="dark">深色</option>
                <option value="macaron">马卡龙</option>
              </select>
              <button
                type="button"
                class="control-btn"
                title="刷新榜单"
                aria-label="刷新榜单"
                :disabled="loading"
                @click="$emit('refresh')"
              >
                <Refresh />
              </button>
              <button
                type="button"
                class="control-btn"
                title="榜单汇总"
                aria-label="榜单汇总"
                @click="summaryOpen = true"
              >
                <InfoFilled />
              </button>
              <button
                type="button"
                class="control-btn"
                title="全屏显示"
                aria-label="全屏显示"
                @click="toggleFullscreen"
              >
                <FullScreen />
              </button>
            </div>
            <button
              type="button"
              class="controls-toggle-btn"
              :class="{ 'is-open': moreOpen }"
              :aria-expanded="moreOpen"
              title="更多选项"
              aria-label="更多选项"
              @click="moreOpen = !moreOpen"
            >
              <MoreFilled />
            </button>
            <output class="filter-summary"
              >P{{ filteredRows.length }} · A{{ summary.attempted }} · S{{
                summary.solved
              }}</output
            >
          </div>
        </div>
      </header>

      <div class="rank-table-region" v-loading="loading">
        <PenaltyLeaderboardTable
          v-if="data.scoringRule === 'penalty'"
          :data="data"
          :rows="filteredRows"
          :mode="mode"
          :rankByID="rankByID"
          :awardByID="awardByID"
        />
        <ScoreLeaderboardTable
          v-else
          :data="data"
          :rows="filteredRows"
          :mode="mode"
          :rankByID="rankByID"
          :awardByID="awardByID"
        />
        <div v-if="error" class="rank-empty">
          <strong>榜单暂不可用</strong><span>{{ error }}</span>
        </div>
        <div v-else-if="!loading && !filteredRows.length" class="rank-empty">
          {{ emptyText }}
        </div>
      </div>
    </div>

    <div
      v-if="summaryOpen"
      class="summary-overlay"
      role="presentation"
      @click.self="summaryOpen = false"
    >
      <section
        class="summary-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="scoreboard-summary-title"
      >
        <button
          type="button"
          class="summary-close"
          aria-label="关闭汇总"
          @click="summaryOpen = false"
        >
          <Close />
        </button>
        <h2 id="scoreboard-summary-title">榜单汇总</h2>
        <div class="summary-cards">
          <div class="summary-card">
            <strong>{{ filteredRows.length }}</strong
            ><span>参赛者</span>
          </div>
          <div class="summary-card">
            <strong>{{ summary.attempted }}</strong
            ><span>有提交</span>
          </div>
          <div class="summary-card">
            <strong>{{ summary.solved }}</strong
            ><span>{{
              data.scoringRule === "score" ? "有满分" : "有通过"
            }}</span>
          </div>
          <div class="summary-card">
            <strong>{{ data.problems.length }}</strong
            ><span>题目</span>
          </div>
        </div>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import {
  Histogram,
  Refresh,
  InfoFilled,
  FullScreen,
  MoreFilled,
  RefreshRight,
  Close,
} from "@element-plus/icons-vue";
import PenaltyLeaderboardTable from "./PenaltyLeaderboardTable.vue";
import ScoreLeaderboardTable from "./ScoreLeaderboardTable.vue";
import {
  awardTierForRank,
  normalizeAwardPercents,
} from "../features/leaderboard/awards";
import type {
  LeaderboardAwardTier,
  LeaderboardData,
} from "../features/leaderboard/types";
import {
  sortLeaderboardRows,
  type LeaderboardViewMode,
} from "../features/leaderboard/sorting";

const props = withDefaults(
  defineProps<{
    data: LeaderboardData;
    loading?: boolean;
    error?: string;
    emptyText?: string;
    updatedAt?: string;
  }>(),
  {
    loading: false,
    error: "",
    emptyText: "暂无参赛记录",
    updatedAt: "",
  },
);

defineEmits<{
  refresh: [];
}>();

const root = ref<HTMLElement | null>(null);
const mode = ref<LeaderboardViewMode>("performance");
const query = ref("");
const attemptedOnly = ref(false);
const solvedOnly = ref(false);
const theme = ref<"default" | "dark" | "macaron">("default");
const currentTime = ref(props.data.currentTimeSeconds);
const summaryOpen = ref(false);
const moreOpen = ref(false);

watch(
  () => props.data.currentTimeSeconds,
  (value) => {
    currentTime.value = value;
  },
);

const progress = computed(() =>
  props.data.durationSeconds
    ? Math.min(
        100,
        Math.max(0, (currentTime.value / props.data.durationSeconds) * 100),
      )
    : 0,
);
const rankedRows = computed(() =>
  sortLeaderboardRows(props.data.rows, props.data.scoringRule, mode.value),
);
const actualParticipantCount = computed(() =>
  Math.max(props.data.rows.length, Number(props.data.participantCount) || 0),
);
const awardPercents = computed(() =>
  normalizeAwardPercents(props.data.awardPercents),
);
const rankByID = computed<Record<string, number>>(() =>
  Object.fromEntries(rankedRows.value.map((row) => [String(row.id), row.rank])),
);
const awardByID = computed<Record<string, LeaderboardAwardTier>>(() =>
  Object.fromEntries(
    rankedRows.value.map((row) => {
      const rank = rankByID.value[String(row.id)] ?? row.rank;
      return [
        String(row.id),
        awardTierForRank(
          rank,
          actualParticipantCount.value,
          awardPercents.value,
        ),
      ];
    }),
  ),
);
const filteredRows = computed(() => {
  const keyword = query.value.trim().toLocaleLowerCase();
  return rankedRows.value.filter((row) => {
    if (attemptedOnly.value && row.submissions <= 0) return false;
    if (solvedOnly.value && row.solved <= 0) return false;
    if (!keyword) return true;
    return row.name.toLocaleLowerCase().includes(keyword);
  });
});
const summary = computed(() => ({
  attempted: filteredRows.value.filter((row) => row.submissions > 0).length,
  solved: filteredRows.value.filter((row) => row.solved > 0).length,
}));

function formatClock(seconds: number) {
  const value = Math.max(0, Math.floor(Number(seconds) || 0));
  const hours = Math.floor(value / 3600);
  const minutes = Math.floor((value % 3600) / 60);
  const remainder = value % 60;
  return [hours, minutes, remainder]
    .map((part) => String(part).padStart(2, "0"))
    .join(":");
}

function resetTime() {
  currentTime.value = props.data.currentTimeSeconds;
}
async function toggleFullscreen() {
  try {
    if (document.fullscreenElement) await document.exitFullscreen();
    else await root.value?.requestFullscreen();
  } catch {
    // 浏览器或嵌入策略拒绝全屏时，榜单仍可正常使用。
  }
}
</script>

<style scoped>
.control-btn svg,
.time-reset-btn svg,
.controls-toggle-btn svg,
.summary-close svg {
  width: 18px;
  height: 18px;
}
.scoreboard-title-icon svg {
  width: 26px;
  height: 26px;
}
.scoreboard-core,
.scoreboard-core :deep(*) {
  box-sizing: border-box;
}
.scoreboard-core {
  --score-blue: var(--accent);
  --score-border: var(--border);
  --score-muted: var(--muted);
  --score-surface: var(--surface-strong);
  --score-card: var(--surface);
  --score-text: var(--text);
  --score-header-bg: var(--surface-strong);
  --score-header-separator: var(--border);
  --score-card-border: var(--border);
  position: relative;
  min-width: 0;
  overflow: visible;
  border: 0;
  border-radius: 0;
  color: var(--score-text);
  background: var(--score-surface);
  font-family:
    Inter,
    ui-sans-serif,
    -apple-system,
    BlinkMacSystemFont,
    "Segoe UI",
    sans-serif;
}
.scoreboard-shell {
  color: var(--score-text);
  background: var(--score-surface);
}
.scoreboard-core[data-theme="dark"] {
  --score-surface: #182235;
  --score-card: #101827;
  --score-text: #e5edf8;
  --score-border: #334155;
  --score-muted: #9ca3af;
  --score-header-bg: #121c2d;
  --score-header-separator: #243149;
  --score-card-border: #46546a;
  background: #0b1220;
}
.scoreboard-core[data-theme="macaron"] {
  --score-surface: #fff7fb;
  --score-card: #fffdfd;
  --score-border: #e9d9e7;
  --score-text: #3f3440;
  --score-muted: #806f82;
  --score-blue: #6f76d9;
  --score-header-bg: #fff7fb;
  --score-header-separator: #eee3ee;
  --score-card-border: #e9d9e7;
  background: #f7f1fb;
}
.scoreboard-title {
  min-height: 73px;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 10px clamp(16px, 4vw, 40px);
  border-bottom: 1px solid var(--score-border);
  background: var(--score-card);
}
.scoreboard-title-icon {
  flex: none;
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  border-radius: 0;
  color: #fff;
  background: #2563eb;
  font-size: 23px;
  font-weight: 900;
}
.scoreboard-title-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: clamp(17px, 2vw, 24px);
  font-weight: 800;
}
.scoreboard-subtitle {
  margin-left: auto;
  color: var(--score-muted);
  font-size: 13px;
}
.scoreboard-freeze {
  flex-shrink: 0;
  padding: 3px 7px;
  border: 1px solid currentColor;
  border-radius: 5px;
  color: var(--score-muted);
  font-size: 12px;
}
.controls-time-section {
  padding: 8px clamp(16px, 4vw, 40px);
  border-bottom: 1px solid var(--score-border);
  background: var(--score-surface);
}
.controls-time-container {
  min-height: 54px;
  display: flex;
  align-items: center;
  gap: 8px 16px;
  max-width: 1360px;
  margin: 0 auto;
}
.score-control-field {
  display: grid;
  gap: 3px;
}
.score-control-label {
  color: var(--score-muted);
  font-size: 11px;
  font-weight: 700;
}
.score-control-select,
.score-theme-select,
.score-query-input {
  height: 36px;
  border: 1px solid var(--score-border);
  border-radius: 6px;
  padding: 0 10px;
  color: var(--score-text);
  background: var(--score-card);
  font: inherit;
}
.rank-mode {
  font-size: 14px;
  white-space: nowrap;
  padding: 5px 0;
}
.score-query-input {
  width: min(190px, 20vw);
}
.time-progress-wrapper {
  flex: 1 1 300px;
  min-width: 210px;
  display: grid;
  grid-template-columns: minmax(130px, 1fr) 34px;
  grid-template-rows: 19px 24px;
  align-items: center;
  column-gap: 9px;
}
.time-display-group {
  grid-column: 1;
  display: flex;
  justify-content: space-between;
  color: var(--score-muted);
  font:
    700 12px/1 ui-monospace,
    SFMono-Regular,
    Consolas,
    monospace;
}
.time-current {
  color: var(--score-blue);
}
.time-progress-slider {
  grid-column: 1;
  width: 100%;
  height: 16px;
  margin: 0;
  cursor: pointer;
  accent-color: var(--score-blue);
}
.time-reset-btn {
  grid-column: 2;
  grid-row: 1 / 3;
}
.score-filter-group {
  display: flex;
  align-items: center;
  gap: 8px;
}
.score-check {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--score-muted);
  font-size: 12px;
  white-space: nowrap;
  cursor: pointer;
}
.score-check input {
  accent-color: var(--score-blue);
}
.controls-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border: 1px solid var(--score-border);
  border-radius: 8px;
  background: var(--score-card);
}
.score-theme-select {
  width: 88px;
}
.control-btn,
.time-reset-btn,
.controls-toggle-btn,
.summary-close {
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 1px solid var(--score-border);
  border-radius: 5px;
  color: var(--score-blue);
  background: var(--score-card);
  font-size: 18px;
  cursor: pointer;
}
.control-btn:hover,
.time-reset-btn:hover,
.controls-toggle-btn:hover {
  border-color: var(--score-blue);
  color: var(--score-blue);
  background: color-mix(in srgb, var(--score-blue) 10%, var(--score-card));
}
.control-btn:disabled {
  opacity: 0.55;
  cursor: wait;
}
.controls-toggle-btn {
  display: none;
}
.filter-summary {
  color: var(--score-muted);
  font:
    700 12px/1 ui-monospace,
    SFMono-Regular,
    Consolas,
    monospace;
  white-space: nowrap;
}
.rank-table-region {
  width: 100%;
  min-width: 0;
  background: var(--score-surface);
}
.rank-empty {
  display: grid;
  gap: 5px;
  margin: 16px;
  padding: 48px;
  border: 1px dashed var(--score-border);
  border-radius: 8px;
  color: var(--score-muted);
  background: var(--score-card);
  text-align: center;
}
.summary-overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: grid;
  place-items: center;
  padding: 20px;
  background: var(--color-scrim);
}
.summary-dialog {
  position: relative;
  width: min(520px, 100%);
  padding: 24px;
  border: 1px solid var(--score-border);
  border-radius: 12px;
  color: var(--score-text);
  background: var(--score-card);
  box-shadow: var(--shadow-panel);
}
.summary-dialog h2 {
  margin: 0 0 20px;
}
.summary-close {
  position: absolute;
  top: 12px;
  right: 12px;
}
.summary-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
}
.summary-card {
  display: grid;
  gap: 4px;
  padding: 15px 10px;
  border-radius: 8px;
  color: var(--score-muted);
  background: var(--score-surface);
  text-align: center;
  font-size: 12px;
}
.summary-card strong {
  color: var(--score-blue);
  font-size: 24px;
}
@media (max-width: 1100px) {
  .scoreboard-subtitle {
    display: none;
  }
  .controls-time-section {
    padding: 8px 16px;
  }
  .controls-time-container {
    position: relative;
    min-height: 0;
    flex-wrap: wrap;
    align-items: stretch;
  }
  .score-control-field {
    flex: 1 1 170px;
  }
  .score-control-select,
  .score-query-input {
    width: 100%;
  }
  .time-progress-wrapper {
    order: 3;
    flex-basis: 100%;
  }
  .score-filter-group {
    order: 4;
    flex: 1 1 calc(100% - 50px);
    overflow-x: auto;
    padding-bottom: 2px;
  }
  .score-filter-group .score-query-input {
    min-width: 170px;
    flex: 1;
  }
  .controls-toolbar {
    display: none;
    order: 6;
    flex: 1 1 100%;
    justify-content: flex-end;
  }
  .controls-toolbar.is-open {
    display: flex;
  }
  .controls-toggle-btn {
    order: 5;
    display: grid;
    margin-left: auto;
  }
  .filter-summary {
    order: 7;
    flex-basis: 100%;
    padding: 2px 0;
  }
}
@media (max-width: 520px) {
  .scoreboard-title {
    min-height: 58px;
    padding: 10px 14px;
  }
  .scoreboard-title-icon {
    width: 34px;
    height: 34px;
  }
  .controls-time-container {
    gap: 8px;
  }
  .score-control-field {
    flex-basis: calc(100% - 44px);
  }
  .time-progress-wrapper {
    min-width: 0;
  }
  .score-filter-group {
    flex-wrap: nowrap;
  }
  .filter-summary {
    display: none;
  }
  .summary-cards {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
