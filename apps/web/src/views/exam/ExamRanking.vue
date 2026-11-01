<template>
  <section class="ranking-view">
    <LeaderboardBoard
      :data="scoreboardData"
      :context="{ kind: 'exam', eventId: Number(route.params.id) }"
      :loading="loading"
      :error="error"
      :empty-text="emptyText"
      :updated-at="updatedAtLabel"
      :auto-refresh-seconds="autoRefresh ? REFRESH_SECONDS : 0"
      @refresh="loadRanking()"
    />
  </section>
</template>

<script setup lang="ts">
import { apiErrorMessage } from "../../api/errors";
defineOptions({ inheritAttrs: false });

import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { generatedApi } from "../../api/client";
import LeaderboardBoard from "../../components/LeaderboardBoard.vue";
import { adaptExamRanking } from "../../features/leaderboard/adapters";
import { formatDateTime } from "../../features/time";
import { useVisibleInterval } from "../../features/visibleInterval";

const REFRESH_SECONDS = 60;
const props = defineProps<{ detail?: any }>();
const route = useRoute();
const ranking = ref<any>(null);
const loading = ref(false);
const error = ref("");
const lastLoadedAt = ref<Date | null>(null);
let disposed = false;
let requestSequence = 0;
let controller: AbortController | undefined;

const scoreboardData = computed(() =>
  adaptExamRanking(ranking.value || { exam: props.detail?.exam }),
);
const emptyText = computed(() =>
  ranking.value?.has_class === false
    ? "暂无课程学生的考试记录"
    : "暂无考试记录",
);
// Standings change while the exam runs and until the last judgement after it.
const autoRefresh = computed(
  () =>
    ranking.value?.lifecycle === "running" ||
    (ranking.value?.lifecycle === "closed" &&
      ranking.value?.final_ready === false),
);
const updatedAtLabel = computed(() =>
  lastLoadedAt.value ? formatDateTime(lastLoadedAt.value) : "",
);

// A silent refresh leaves the board untouched: no loading mask, never interrupts a
// manual reload, and a transient failure keeps the last standings.
async function loadRanking(silent = false) {
  const examID = Number(route.params.id);
  if (!examID || disposed || (silent && loading.value)) return;
  controller?.abort();
  controller = new AbortController();
  const signal = controller.signal,
    sequence = ++requestSequence;
  if (!silent) {
    loading.value = true;
    error.value = "";
  }
  try {
    const { data } = await generatedApi.examRanking<any>({
      path: { id: examID },
      signal,
    });
    if (
      disposed ||
      sequence !== requestSequence ||
      examID !== Number(route.params.id)
    )
      return;
    ranking.value = data;
    error.value = "";
    lastLoadedAt.value = new Date();
  } catch (err: any) {
    if (signal.aborted || sequence !== requestSequence || silent) return;
    ranking.value = null;
    const apiError = apiErrorMessage(err);
    error.value =
      err.response?.status === 403
        ? apiError === "exam has not started"
          ? "考试尚未开始，开始后可查看榜单。"
          : "创建者未向考生开放该榜单。"
        : apiError;
  } finally {
    if (sequence === requestSequence) loading.value = false;
  }
}

watch(
  () => route.params.id,
  () => {
    ranking.value = null;
    void loadRanking();
  },
);
onMounted(() => loadRanking());
useVisibleInterval(
  () => loadRanking(true),
  REFRESH_SECONDS * 1000,
  () => autoRefresh.value,
);
onBeforeUnmount(() => {
  disposed = true;
  controller?.abort();
});
</script>

<style scoped>
.ranking-view {
  min-width: 0;
  padding: 2px;
}
</style>
