<template>
  <section class="ranking-view">
    <LeaderboardBoard
      :data="scoreboardData"
      :context="{ kind: 'exam', eventId: Number(route.params.id) }"
      :loading="loading"
      :error="error"
      :empty-text="emptyText"
      :updated-at="updatedAtLabel"
      @refresh="loadRanking"
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
const updatedAtLabel = computed(() =>
  lastLoadedAt.value ? formatDateTime(lastLoadedAt.value) : "",
);

async function loadRanking() {
  const examID = Number(route.params.id);
  if (!examID || disposed) return;
  controller?.abort();
  controller = new AbortController();
  const signal = controller.signal,
    sequence = ++requestSequence;
  loading.value = true;
  error.value = "";
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
    lastLoadedAt.value = new Date();
  } catch (err: any) {
    if (signal.aborted || sequence !== requestSequence) return;
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
onMounted(loadRanking);
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
