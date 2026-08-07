<template>
  <section ref="root" class="scoreboard-core" :aria-busy="loading">
    <div class="scoreboard-shell" :data-theme="theme" :data-rank-mode="mode">
      <header class="rank-header">
        <div class="scoreboard-title">
          <span class="scoreboard-title-icon" aria-hidden="true">☷</span>
          <span class="scoreboard-title-text">{{ data.title }}</span>
          <span v-if="data.subtitle" class="scoreboard-subtitle">{{ data.subtitle }}</span>
        </div>

        <div class="controls-time-section">
          <div class="controls-time-container">
            <label class="score-control-field">
              <span class="score-control-label">排名模式</span>
              <select v-model="mode" class="score-control-select" aria-label="排名模式">
                <option value="published">公布排名</option>
                <option value="performance">实时表现</option>
              </select>
            </label>

            <div class="time-progress-wrapper">
              <div class="time-display-group">
                <span class="time-current">{{ formatClock(currentTime) }}</span>
                <span class="time-total">{{ formatClock(data.durationSeconds) }}</span>
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
              <button type="button" class="time-reset-btn" title="回到最新时间" aria-label="回到最新时间" @click="resetTime">↦</button>
            </div>

            <div class="score-filter-group">
              <input v-model="query" class="score-query-input" type="search" placeholder="筛选参赛者…" aria-label="筛选参赛者" />
              <label class="score-check"><input v-model="attemptedOnly" type="checkbox" /><span>有提交</span></label>
              <label class="score-check"><input v-model="solvedOnly" type="checkbox" /><span>有通过</span></label>
              <label v-if="showAutoRefresh" class="score-check"><input :checked="autoRefresh" type="checkbox" @change="updateAutoRefresh" /><span>自动刷新</span></label>
            </div>

            <div class="controls-toolbar" :class="{ 'is-open': moreOpen }">
              <select v-model="theme" class="score-theme-select" aria-label="榜单主题">
                <option value="default">默认</option>
                <option value="dark">深色</option>
                <option value="macaron">马卡龙</option>
              </select>
              <button type="button" class="control-btn" title="刷新榜单" aria-label="刷新榜单" :disabled="loading" @click="$emit('refresh')">↻</button>
              <button type="button" class="control-btn" title="榜单汇总" aria-label="榜单汇总" @click="summaryOpen = true">ⓘ</button>
              <button type="button" class="control-btn" title="全屏显示" aria-label="全屏显示" @click="toggleFullscreen">⛶</button>
            </div>
            <button
              type="button"
              class="controls-toggle-btn"
              :class="{ 'is-open': moreOpen }"
              :aria-expanded="moreOpen"
              title="更多选项"
              aria-label="更多选项"
              @click="moreOpen = !moreOpen"
            >⋯</button>
            <output class="filter-summary">P{{ filteredRows.length }} · A{{ summary.attempted }} · S{{ summary.solved }}</output>
          </div>
        </div>
      </header>

      <div class="rank-scroll-viewport" v-loading="loading">
        <div class="rank-system" :style="{ '--problem-count': data.problems.length }">
          <div class="rank-header-row" role="row">
            <div class="rank-header-identity">{{ data.identityLabel }}</div>
            <div class="stats-section rank-header-stats">
              <div class="rank-col rank-col-rank"><div class="rank-header-cell">排名</div></div>
              <div class="rank-col rank-col-solve"><div class="rank-header-cell">{{ data.solvedLabel }}</div></div>
              <div class="rank-col rank-col-penalty"><div class="rank-header-cell">{{ data.metricLabel }}</div></div>
              <div class="problem-group">
                <div v-for="problem in data.problems" :key="problem.id" class="rank-col rank-col-problem">
                  <div class="problem-header-item" :title="problem.title">
                    <span class="problem-color" :style="{ backgroundColor: problem.color }"></span>
                    <strong class="problem-header-title">{{ problem.label }}</strong>
                    <small class="problem-header-stats">{{ problemStats(problem.id).solved }}/{{ problemStats(problem.id).attempted }}</small>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="rank-grid">
            <article v-for="(row, index) in filteredRows" :key="row.id" class="rank-row" :class="index % 2 ? 'odd' : 'even'" role="row">
              <div class="rank-main-content">
                <div class="school-logo school-logo-placeholder" aria-hidden="true">{{ row.organization?.slice(0, 1) || row.name.slice(0, 1) }}</div>
                <div class="top-section">
                  <div class="coach-player-section">
                    <span v-if="row.meta" class="player-info">{{ row.meta }}</span>
                  </div>
                  <div class="team-info-section">
                    <div class="team-info">
                      <span class="team-type-icon" aria-hidden="true">◆</span>
                      <div class="school-name">{{ row.organization || '参赛者' }}</div>
                      <div class="team-names">{{ row.name }}</div>
                    </div>
                  </div>
                </div>

                <div class="stats-section">
                  <div class="rank-col rank-col-rank"><div class="rank-item">{{ mode === 'performance' ? index + 1 : row.rank }}</div></div>
                  <div class="rank-col rank-col-solve"><div class="solve-item">{{ row.solved }}</div></div>
                  <div class="rank-col rank-col-penalty"><div class="penalty-item">{{ row.metricDisplay ?? row.metric }}</div></div>
                  <div class="problem-group">
                    <div v-for="problem in data.problems" :key="problem.id" class="rank-col rank-col-problem">
                      <div
                        class="problem-item"
                        :class="[`pro-${resultFor(row, problem.id).status}`, { 'pro-first-blood': resultFor(row, problem.id).firstBlood }]"
                        :data-status="resultFor(row, problem.id).status"
                        :title="resultTitle(row, problem.id)"
                      >
                        <div class="problem-content">
                          <span v-if="resultFor(row, problem.id).firstBlood" class="first-blood">★</span>
                          <span class="pro-submit-cnt">{{ resultFor(row, problem.id).primary || problem.label }}</span>
                          <template v-if="resultFor(row, problem.id).secondary">
                            <span class="problem-separator">|</span>
                            <span class="time-brief">{{ resultFor(row, problem.id).secondary }}</span>
                          </template>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </article>

            <div v-if="error" class="rank-empty"><strong>榜单暂不可用</strong><span>{{ error }}</span></div>
            <div v-else-if="!loading && !filteredRows.length" class="rank-empty">{{ emptyText }}</div>
          </div>
        </div>
      </div>

      <footer class="scoreboard-footer">
        <div class="score-legend"><span class="accepted">已通过</span><span class="wrong">未通过</span><span class="pending">评分中</span><span class="first">★ 最快通过</span></div>
        <small v-if="updatedAt">更新于 {{ updatedAt }}</small>
      </footer>
    </div>

    <div v-if="summaryOpen" class="summary-overlay" role="presentation" @click.self="summaryOpen = false">
      <section class="summary-dialog" role="dialog" aria-modal="true" aria-labelledby="scoreboard-summary-title">
        <button type="button" class="summary-close" aria-label="关闭汇总" @click="summaryOpen = false">×</button>
        <h2 id="scoreboard-summary-title">榜单汇总</h2>
        <div class="summary-cards">
          <div class="summary-card"><strong>{{ filteredRows.length }}</strong><span>参赛者</span></div>
          <div class="summary-card"><strong>{{ summary.attempted }}</strong><span>有提交</span></div>
          <div class="summary-card"><strong>{{ summary.solved }}</strong><span>有通过</span></div>
          <div class="summary-card"><strong>{{ data.problems.length }}</strong><span>题目</span></div>
        </div>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { LeaderboardData, LeaderboardResult, LeaderboardRow } from '../features/leaderboard/types'

const props = withDefaults(defineProps<{
  data: LeaderboardData
  loading?: boolean
  error?: string
  emptyText?: string
  updatedAt?: string
  showAutoRefresh?: boolean
  autoRefresh?: boolean
}>(), {
  loading: false,
  error: '',
  emptyText: '暂无参赛记录',
  updatedAt: '',
  showAutoRefresh: false,
  autoRefresh: false
})

const emit = defineEmits<{
  refresh: []
  'update:autoRefresh': [value: boolean]
}>()

const root = ref<HTMLElement | null>(null)
const mode = ref<'published' | 'performance'>('published')
const query = ref('')
const attemptedOnly = ref(false)
const solvedOnly = ref(false)
const theme = ref<'default' | 'dark' | 'macaron'>('default')
const currentTime = ref(props.data.currentTimeSeconds)
const summaryOpen = ref(false)
const moreOpen = ref(false)

watch(() => props.data.currentTimeSeconds, (value) => { currentTime.value = value })

const progress = computed(() => props.data.durationSeconds ? Math.min(100, Math.max(0, currentTime.value / props.data.durationSeconds * 100)) : 0)
const filteredRows = computed(() => {
  const keyword = query.value.trim().toLocaleLowerCase()
  const rows = props.data.rows.filter((row) => {
    if (attemptedOnly.value && row.submissions <= 0) return false
    if (solvedOnly.value && row.solved <= 0) return false
    if (!keyword) return true
    return [row.name, row.organization, row.meta].filter(Boolean).join('\n').toLocaleLowerCase().includes(keyword)
  })
  if (mode.value === 'published') return rows.sort((left, right) => left.rank - right.rank)
  const direction = props.data.metricDirection === 'ascending' ? 1 : -1
  return rows.sort((left, right) => right.solved - left.solved || direction * (left.metric - right.metric) || left.rank - right.rank)
})
const summary = computed(() => ({
  attempted: filteredRows.value.filter((row) => row.submissions > 0).length,
  solved: filteredRows.value.filter((row) => row.solved > 0).length
}))

function resultFor(row: LeaderboardRow, problemID: number | string): LeaderboardResult {
  return row.results[String(problemID)] || { status: 'none', attempts: 0 }
}

function problemStats(problemID: number | string) {
  let attempted = 0
  let solved = 0
  for (const row of props.data.rows) {
    const result = resultFor(row, problemID)
    if (result.attempts) attempted += 1
    if (result.status === 'accepted') solved += 1
  }
  return { attempted, solved }
}

function resultTitle(row: LeaderboardRow, problemID: number | string) {
  const result = resultFor(row, problemID)
  const status = { accepted: '已通过', wrong: '未通过', pending: '评分中', frozen: '已封榜', none: '未提交' }[result.status]
  return [status, result.primary, result.secondary].filter(Boolean).join(' · ')
}

function formatClock(seconds: number) {
  const value = Math.max(0, Math.floor(Number(seconds) || 0))
  const hours = Math.floor(value / 3600)
  const minutes = Math.floor((value % 3600) / 60)
  const remainder = value % 60
  return [hours, minutes, remainder].map((part) => String(part).padStart(2, '0')).join(':')
}

function resetTime() { currentTime.value = props.data.currentTimeSeconds }
function updateAutoRefresh(event: Event) { emit('update:autoRefresh', (event.target as HTMLInputElement).checked) }
async function toggleFullscreen() {
  try {
    if (document.fullscreenElement) await document.exitFullscreen()
    else await root.value?.requestFullscreen()
  } catch {
    // 浏览器或嵌入策略拒绝全屏时，榜单仍可正常使用。
  }
}
</script>

<style scoped>
.scoreboard-core, .scoreboard-core * { box-sizing: border-box; }
.scoreboard-core {
  --score-blue: #0d6efd; --score-border: #dee2e6; --score-muted: #6c757d;
  --score-surface: #f8f9fa; --score-card: #fff; --score-text: #111827;
  position: relative; min-width: 0; overflow: hidden; border: 1px solid var(--score-border);
  border-radius: 10px; color: var(--score-text); background: #eef2f7;
  font-family: Inter, ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}
.scoreboard-shell[data-theme="dark"] { --score-surface: #182235; --score-card: #101827; --score-text: #e5edf8; --score-border: #334155; --score-muted: #9ca3af; background: #0b1220; }
.scoreboard-shell[data-theme="macaron"] { --score-surface: #fff7fb; --score-card: #fffdfd; --score-border: #e9d9e7; --score-text: #3f3440; --score-muted: #806f82; --score-blue: #6f76d9; background: #f7f1fb; }
.scoreboard-shell { background: #eef2f7; color: var(--score-text); }
.scoreboard-title { min-height: 68px; display: flex; align-items: center; gap: 12px; padding: 12px clamp(16px, 4vw, 40px); background: var(--score-card); border-bottom: 1px solid var(--score-border); }
.scoreboard-title-icon { display: grid; place-items: center; width: 38px; height: 38px; flex: none; border-radius: 8px; color: #fff; background: linear-gradient(135deg, #0d6efd, #2451a4); font-size: 22px; font-weight: 900; }
.scoreboard-title-text { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: clamp(17px, 2vw, 24px); font-weight: 800; }
.scoreboard-subtitle { margin-left: auto; color: var(--score-muted); font-size: 13px; }
.controls-time-section { padding: 8px clamp(16px, 4vw, 40px); background: var(--score-surface); border-bottom: 1px solid var(--score-border); }
.controls-time-container { min-height: 54px; display: flex; align-items: center; gap: 8px 16px; max-width: 1360px; margin: 0 auto; }
.score-control-field { display: grid; gap: 3px; }
.score-control-label { color: var(--score-muted); font-size: 11px; font-weight: 700; }
.score-control-select, .score-theme-select, .score-query-input { height: 36px; border: 1px solid var(--score-border); border-radius: 6px; padding: 0 10px; color: var(--score-text); background: var(--score-card); font: inherit; }
.score-control-select { width: 132px; }
.score-query-input { width: min(190px, 20vw); }
.time-progress-wrapper { flex: 1 1 300px; min-width: 210px; display: grid; grid-template-columns: minmax(130px, 1fr) 34px; grid-template-rows: 19px 24px; align-items: center; column-gap: 9px; }
.time-display-group { grid-column: 1; display: flex; justify-content: space-between; color: var(--score-muted); font: 700 12px/1 ui-monospace, SFMono-Regular, Consolas, monospace; }
.time-current { color: var(--score-blue); }
.time-progress-slider { grid-column: 1; width: 100%; height: 16px; margin: 0; cursor: pointer; accent-color: var(--score-blue); }
.time-reset-btn { grid-column: 2; grid-row: 1 / 3; }
.score-filter-group { display: flex; align-items: center; gap: 8px; }
.score-check { display: inline-flex; align-items: center; gap: 4px; color: var(--score-muted); font-size: 12px; white-space: nowrap; cursor: pointer; }
.score-check input { accent-color: var(--score-blue); }
.controls-toolbar { display: flex; align-items: center; gap: 8px; padding: 6px 8px; border: 1px solid var(--score-border); border-radius: 8px; background: var(--score-card); }
.score-theme-select { width: 88px; }
.control-btn, .time-reset-btn, .controls-toggle-btn, .summary-close { display: grid; place-items: center; width: 34px; height: 34px; padding: 0; border: 1px solid var(--score-border); border-radius: 5px; color: #2451a4; background: var(--score-card); font-size: 18px; cursor: pointer; }
.control-btn:hover, .time-reset-btn:hover, .controls-toggle-btn:hover { border-color: var(--score-blue); color: var(--score-blue); background: #eaf2ff; }
.control-btn:disabled { opacity: .55; cursor: wait; }
.controls-toggle-btn { display: none; }
.filter-summary { color: var(--score-muted); font: 700 12px/1 ui-monospace, SFMono-Regular, Consolas, monospace; white-space: nowrap; }
.rank-scroll-viewport { width: 100%; max-height: 650px; overflow: auto; overscroll-behavior-inline: contain; }
.rank-system { --identity-width: 470px; --rank-width: 64px; --solve-width: 72px; --penalty-width: 92px; --problem-width: 85px; min-width: calc(var(--identity-width) + var(--rank-width) + var(--solve-width) + var(--penalty-width) + var(--problem-count) * var(--problem-width) + 32px); padding: 0 16px 24px; }
.rank-header-row { position: sticky; top: 0; z-index: 20; height: 73px; display: grid; grid-template-columns: var(--identity-width) max-content; align-items: end; padding: 12px 0 0; background: var(--score-surface); border-bottom: 1px solid var(--score-border); box-shadow: 0 3px 10px rgba(15, 23, 42, .06); }
.rank-header-identity { padding: 0 18px 14px; color: var(--score-muted); font-size: 13px; font-weight: 800; letter-spacing: .04em; }
.stats-section { display: flex; align-items: center; gap: 5px; min-width: max-content; }
.rank-header-stats { height: 60px; padding: 0 0 8px; align-self: end; }
.rank-col { flex: none; text-align: center; }
.rank-col-rank { width: var(--rank-width); } .rank-col-solve { width: var(--solve-width); } .rank-col-penalty { width: var(--penalty-width); } .rank-col-problem { width: 80px; }
.problem-group { display: flex; gap: 5px; }
.rank-header-cell, .problem-header-item { height: 44px; display: flex; flex-direction: column; align-items: center; justify-content: center; border: 1px solid var(--score-border); border-radius: 5px; color: var(--score-text); background: var(--score-card); font-size: 12px; font-weight: 800; }
.problem-header-item { position: relative; overflow: hidden; }
.problem-color { position: absolute; inset: 0 0 auto; height: 4px; }
.problem-header-title { font-size: 17px; } .problem-header-stats { color: var(--score-muted); font-size: 10px; }
.rank-grid { display: grid; gap: 6px; padding-top: 6px; }
.rank-row { min-height: 85px; overflow: hidden; border: 1px solid var(--score-border); border-radius: 7px; background: var(--score-card); box-shadow: 0 1px 3px rgba(15, 23, 42, .05); }
.rank-row:hover { border-color: #9bbcf4; box-shadow: 0 5px 16px rgba(34, 80, 148, .12); }
.rank-main-content { position: relative; min-height: 85px; display: grid; grid-template-columns: var(--identity-width) max-content; align-items: end; }
.school-logo { position: absolute; left: 10px; top: 8px; width: 68px; height: 68px; opacity: .14; pointer-events: none; }
.school-logo-placeholder { display: grid; place-items: center; border-radius: 50%; color: #2451a4; background: #dbeafe; font-size: 30px; font-weight: 900; }
.top-section { min-width: 0; height: 85px; display: grid; grid-template-rows: 31px 45px; padding: 5px 12px 0 88px; }
.coach-player-section { display: flex; align-items: center; min-width: 0; color: var(--score-muted); font-size: 12px; }
.player-info, .school-name, .team-names { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.team-info-section, .team-info { min-width: 0; }
.team-info { height: 42px; display: grid; grid-template-columns: 24px minmax(100px, 160px) minmax(160px, 1fr); align-items: center; gap: 8px; }
.team-type-icon { color: #58759f; text-align: center; }
.school-name { font-size: 15px; font-weight: 800; } .team-names { font-size: 18px; font-weight: 800; }
.rank-main-content > .stats-section { grid-column: 2; grid-row: 1; height: 85px; align-items: end; padding-bottom: 4px; }
.rank-item, .solve-item, .penalty-item, .problem-item { height: 32px; display: flex; align-items: center; justify-content: center; border-radius: 4px; font-size: 16px; font-weight: 900; }
.rank-item, .solve-item, .penalty-item { border: 1px solid var(--score-border); background: var(--score-surface); }
.penalty-item { padding: 0 4px; font-size: 14px; line-height: 1.05; }
.problem-item { position: relative; width: 80px; border: 1px solid transparent; color: #fff; transition: transform 150ms ease, box-shadow 150ms ease, filter 150ms ease; }
.problem-item:hover { transform: translateY(-1px); filter: saturate(1.1); box-shadow: 0 3px 8px rgba(15, 23, 42, .22); }
.problem-item.pro-ac, .problem-item.pro-accepted { border-color: #146c43; background: #198754; }
.problem-item.pro-wa, .problem-item.pro-wrong { border-color: #b02a37; background: #dc3545; }
.problem-item.pro-none { border-color: #e9ecef; color: #26313f; background: #adb5bd; }
.problem-item.pro-pending, .problem-item.pro-frozen { border-color: #0a58ca; background: #0d6efd; }
.problem-item.pro-frozen { background-image: repeating-linear-gradient(135deg, transparent 0 7px, rgba(255,255,255,.16) 7px 12px); }
.problem-content { display: flex; align-items: center; justify-content: center; gap: 3px; }
.problem-separator { opacity: .72; } .time-brief { font-size: 11px; }
.first-blood { position: absolute; top: -7px; right: 1px; color: #ffd43b; font-size: 14px; line-height: 1; text-shadow: 0 1px 2px rgba(0,0,0,.55); }
.rank-empty { display: grid; gap: 5px; padding: 48px; border: 1px dashed var(--score-border); border-radius: 8px; color: var(--score-muted); background: var(--score-card); text-align: center; }
.scoreboard-footer { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 10px 18px; border-top: 1px solid var(--score-border); color: var(--score-muted); background: var(--score-card); }
.score-legend { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.score-legend span { padding: 3px 7px; border-radius: 4px; font-size: 11px; }
.score-legend .accepted { color: #166534; background: #dcfce7; } .score-legend .wrong { color: #991b1b; background: #fee2e2; } .score-legend .pending { color: #075985; background: #dbeafe; } .score-legend .first { color: #8a5800; background: #fef3c7; }
.summary-overlay { position: fixed; inset: 0; z-index: 100; display: grid; place-items: center; padding: 20px; background: rgba(0,0,0,.5); }
.summary-dialog { position: relative; width: min(520px, 100%); padding: 24px; border: 1px solid var(--score-border); border-radius: 12px; color: var(--score-text); background: var(--score-card); box-shadow: 0 24px 60px rgba(0,0,0,.3); }
.summary-dialog h2 { margin: 0 0 20px; } .summary-close { position: absolute; top: 12px; right: 12px; }
.summary-cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; }
.summary-card { display: grid; gap: 4px; padding: 15px 10px; border-radius: 8px; color: var(--score-muted); background: var(--score-surface); text-align: center; font-size: 12px; }
.summary-card strong { color: var(--score-blue); font-size: 24px; }
@media (max-width: 1100px) {
  .scoreboard-subtitle { display: none; }
  .controls-time-section { padding: 8px 16px; }
  .controls-time-container { position: relative; min-height: 0; flex-wrap: wrap; align-items: stretch; }
  .score-control-field { flex: 1 1 170px; }
  .score-control-select, .score-query-input { width: 100%; }
  .time-progress-wrapper { order: 3; flex-basis: 100%; }
  .score-filter-group { order: 4; flex: 1 1 calc(100% - 50px); overflow-x: auto; padding-bottom: 2px; }
  .score-filter-group .score-query-input { min-width: 170px; flex: 1; }
  .controls-toolbar { display: none; order: 6; flex: 1 1 100%; justify-content: flex-end; }
  .controls-toolbar.is-open { display: flex; }
  .controls-toggle-btn { order: 5; display: grid; margin-left: auto; }
  .filter-summary { order: 7; flex-basis: 100%; padding: 2px 0; }
  .rank-system { --identity-width: 430px; }
}
@media (max-width: 520px) {
  .scoreboard-title { min-height: 58px; padding: 10px 14px; } .scoreboard-title-icon { width: 34px; height: 34px; }
  .controls-time-container { gap: 8px; } .score-control-field { flex-basis: calc(100% - 44px); }
  .time-progress-wrapper { min-width: 0; } .score-filter-group { flex-wrap: nowrap; } .filter-summary { display: none; }
  .rank-system { --identity-width: 360px; } .summary-cards { grid-template-columns: repeat(2, 1fr); }
  .scoreboard-footer { align-items: flex-start; flex-direction: column; }
}
@media (prefers-reduced-motion: reduce) { .problem-item { transition: none; } }
</style>
