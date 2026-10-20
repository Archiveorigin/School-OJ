<template>
  <section class="ranking-review">
    <header class="ranking-tools">
      <div>
        <h2>成绩与排名</h2>
        <p>完整后台榜单 · 不受参赛者封榜和页面筛选影响</p>
      </div>
      <div class="ranking-exports">
        <el-button :loading="exporting === 'xlsx'" :disabled="!canExport || !!exporting" @click="download('xlsx')">导出 Excel</el-button>
        <el-button :loading="exporting === 'html'" :disabled="!canExport || !!exporting" @click="download('html')">导出打印榜单</el-button>
      </div>
    </header>
    <p class="ranking-readiness" role="status">{{ readiness }}</p>
    <p v-if="exportError" class="ranking-error" role="alert">{{ exportError }}</p>
    <LeaderboardBoard
      :data="board" :loading="loading" :error="error"
      :updated-at="snapshot ? formatDateTime(snapshot.generated_at) : ''"
      show-auto-refresh v-model:auto-refresh="autoRefresh" @refresh="load"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { generatedApi } from '../../api/client'
import { apiErrorMessage } from '../../api/errors'
import { eventPath, type EventKind, type EventLifecycle, saveDownload } from '../../api/modules/events'
import { adaptExamRanking } from '../../features/leaderboard/adapters'
import { formatDateTime } from '../../features/time'
import LeaderboardBoard from '../LeaderboardBoard.vue'

type Snapshot = {
  title: string; scoring_rule: string; lifecycle: EventLifecycle
  starts_at?: string; ends_at?: string; generated_at: string
  completion_checks: { judging: number; ungraded: number; failed?: number }
  problems: any[]; rows: any[]
}
const props = defineProps<{ kind: EventKind; eventId: number }>()
const emit = defineEmits<{ updated: [snapshot: Snapshot] }>()
const snapshot = ref<Snapshot>(), loading = ref(false), error = ref(''), exportError = ref('')
const autoRefresh = ref(true), exporting = ref('')
let timer: number | undefined, disposed = false
const board = computed(() => adaptExamRanking({
  ...snapshot.value,
  exam: { ...snapshot.value, title: snapshot.value?.title || '本场实际排名' },
  now: snapshot.value?.generated_at,
}))
const canExport = computed(() => !!snapshot.value && !error.value &&
  ['closed', 'completed'].includes(snapshot.value.lifecycle) &&
  !snapshot.value.completion_checks.judging && !snapshot.value.completion_checks.ungraded && !snapshot.value.completion_checks.failed)
const readiness = computed(() => {
  if (!snapshot.value) return '正在读取最新成绩…'
  const { lifecycle, completion_checks: checks } = snapshot.value
  if (checks.failed) return `判题异常 ${checks.failed} 份，请处理并重新评测后再导出或确认完结。`
  if (checks.judging || checks.ungraded) return `待评测 ${checks.judging} 份 · 待人工评分 ${checks.ungraded} 份。榜单持续更新，全部处理完成后可导出。`
  if (lifecycle === 'completed') return '已完结：排名已经管理员确认。'
  if (lifecycle === 'closed') return '已截止：评测与人工评分均已完成，可导出供管理员审核；确认完结后为最终成绩。'
  return '当前为实时排名；截止后且全部评测、评分完成，才能导出实际排名。'
})
function schedule() {
  window.clearTimeout(timer)
  if (!disposed && autoRefresh.value && snapshot.value?.lifecycle !== 'completed') timer = window.setTimeout(load, 5000)
}
async function load() {
  if (loading.value || disposed) return
  loading.value = true
  error.value = ''
  const id = props.eventId, kind = props.kind
  try {
    const { data } = await generatedApi.getEventRanking<Snapshot>({ path: eventPath(kind, id) })
    if (disposed || id !== props.eventId || kind !== props.kind) return
    snapshot.value = data
    emit('updated', data)
  } catch (err) {
    error.value = apiErrorMessage(err)
  } finally { loading.value = false; schedule() }
}
async function download(format: 'xlsx' | 'html') {
  exporting.value = format
  exportError.value = ''
  try {
    const response = await generatedApi.exportEventRanking<Blob>({
      path: eventPath(props.kind, props.eventId), query: { format }, responseType: 'blob',
    })
    saveDownload(response.data, `排名表-${props.eventId}.${format}`)
  } catch (err: any) {
    if (err.response?.data instanceof Blob) {
      try { err.response.data = JSON.parse(await err.response.data.text()) } catch { /* Preserve the original API error. */ }
    }
    exportError.value = apiErrorMessage(err)
    await load()
  } finally { exporting.value = '' }
}
watch(autoRefresh, schedule)
watch(() => [props.kind, props.eventId], () => { snapshot.value = undefined; void load() }, { immediate: true })
onBeforeUnmount(() => { disposed = true; window.clearTimeout(timer) })
</script>

<style scoped>
.ranking-review { min-width: 0; }
.ranking-tools { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 16px; margin-bottom: 16px; }
.ranking-tools h2 { margin: 0 0 6px; font-size: 20px; }
.ranking-tools p { margin: 0; color: var(--color-text-muted); font-size: 13px; }
.ranking-exports { display: flex; flex-wrap: wrap; gap: 8px; }
.ranking-exports .el-button { margin: 0; }
.ranking-readiness { padding: 12px 14px; background: var(--color-info-soft); border-left: 3px solid var(--color-info); color: var(--color-text); line-height: 1.6; }
.ranking-error { color: var(--color-danger); }
</style>
