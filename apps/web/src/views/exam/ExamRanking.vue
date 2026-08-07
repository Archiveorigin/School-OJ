<template>
  <section class="ranking-view">
    <LeaderboardBoard
      :data="scoreboardData"
      :loading="loading"
      :error="error"
      :empty-text="emptyText"
      :updated-at="updatedAtLabel"
      show-auto-refresh
      v-model:auto-refresh="autoRefresh"
      @refresh="loadRanking"
    />
  </section>
</template>

<script setup lang="ts">
defineOptions({ inheritAttrs: false })

import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { client } from '../../api/client'
import LeaderboardBoard from '../../components/LeaderboardBoard.vue'
import { adaptExamRanking } from '../../features/leaderboard/adapters'
import { formatDateTime } from '../../features/time'

const props = defineProps<{ detail?: any }>()
const route = useRoute()
const ranking = ref<any>(null)
const loading = ref(false)
const autoRefresh = ref(true)
const error = ref('')
const lastLoadedAt = ref<Date | null>(null)
let refreshTimer: number | undefined

const scoreboardData = computed(() => adaptExamRanking(ranking.value))
const emptyText = computed(() => ranking.value?.has_class === false
  ? '暂无课程学生的考试记录'
  : '暂无考试记录')
const updatedAtLabel = computed(() => lastLoadedAt.value ? formatDateTime(lastLoadedAt.value) : '')

function rankingEnded() {
  const exam = ranking.value?.exam || props.detail?.exam
  if (!exam) return false
  if (exam.status === '已结束' || exam.status === 'closed') return true
  const end = exam.ends_at ? new Date(exam.ends_at).getTime() : Number.NaN
  return Number.isFinite(end) && Date.now() >= end
}

function clearRefreshTimer() {
  if (refreshTimer) window.clearTimeout(refreshTimer)
  refreshTimer = undefined
}

function scheduleRefresh() {
  clearRefreshTimer()
  if (!autoRefresh.value || rankingEnded()) return
  refreshTimer = window.setTimeout(() => {
    if (rankingEnded()) {
      clearRefreshTimer()
      return
    }
    void loadRanking()
  }, 5000)
}

async function loadRanking() {
  const examID = Number(route.params.id)
  if (!examID || loading.value) return
  loading.value = true
  error.value = ''
  try {
    ranking.value = (await client.get(`/exams/${examID}/ranking`)).data
    lastLoadedAt.value = new Date()
  } catch (err: any) {
    ranking.value = null
    const apiError = err.response?.data?.error
    error.value = err.response?.status === 403
      ? (apiError === 'exam has not started' ? '考试尚未开始，开始后可查看榜单。' : '创建者未向考生开放该榜单。')
      : (apiError || err.message)
  } finally {
    loading.value = false
    scheduleRefresh()
  }
}

watch(autoRefresh, scheduleRefresh)
watch(() => route.params.id, () => {
  clearRefreshTimer()
  ranking.value = null
  void loadRanking()
})
onMounted(loadRanking)
onBeforeUnmount(clearRefreshTimer)
</script>

<style scoped>
.ranking-view { min-width: 0; padding: 2px; }
</style>
