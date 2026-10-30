<template>
  <section class="competition-detail">
    <el-button
      @click="router.push(reviewMode ? '/admin/competitions' : '/competitions')"
      >返回{{ reviewMode ? "审核列表" : "比赛归档" }}</el-button
    >
    <EventReviewPanel
      v-if="reviewMode && validKind"
      :key="`${kind}-${eventID}`"
      :kind="kind"
      :event-id="eventID"
    />
    <template v-else-if="paper">
      <header>
        <div>
          <el-tag>已完结 · {{ kind === "exam" ? "考试" : "团队比赛" }}</el-tag>
          <h1>{{ paper.event.title }}</h1>
        </div>
        <p>本页保留本场题目版本与成绩记录，不再接收代码提交。</p>
      </header>
      <el-tabs v-model="tab"
        ><el-tab-pane label="说明与附件" name="description" /><el-tab-pane
          label="题目"
          name="problems" /><el-tab-pane label="排名" name="ranking"
      /></el-tabs>
      <template v-if="tab === 'description'"
        ><section class="panel description">
          <MarkdownRenderer :source="paper.event.description || '暂无说明'" />
        </section>
        <EventAttachments :kind="kind" :event-id="eventID"
      /></template>
      <template v-if="tab === 'problems'"
        ><div class="problem-buttons">
          <el-button
            v-for="link in paper.problems"
            :key="link.problem_id"
            :type="
              selected?.problem_id === link.problem_id ? 'primary' : 'default'
            "
            @click="selected = link"
            >{{ link.label }}. {{ link.problem.title }}</el-button
          >
        </div>
        <section v-if="selected" class="panel description">
          <h2>{{ selected.label }}. {{ selected.problem.title }}</h2>
          <ProblemStatementView
            :problem="selected.problem"
            embedded
            :asset-base="eventAssets(kind, eventID, selected.problem_id)"
            :score="selected.score"
            :show-meta="false"
          /></section
      ></template>
      <template v-if="tab === 'ranking'"
        ><ContestReplay v-if="kind==='contest' && auth.can('event:review')" :kind="kind" :event-id="eventID" /><LeaderboardBoard
          v-if="paper.can_view_ranking"
          :data="rankingData"
          :loading="rankingLoading"
          :error="rankingError"
          empty-text="暂无成绩记录"
          @refresh="loadRanking" /><el-alert
          v-else
          title="原场次未向当前用户开放排名。"
          type="info"
          :closable="false"
      /></template>
    </template>
    <el-result v-if="error" title="场次不可访问" :sub-title="error" />
    <el-skeleton v-else-if="!reviewMode && !paper" :rows="8" animated />
  </section>
</template>
<script setup lang="ts">
import ContestReplay from "../../components/events/ContestReplay.vue";
import { useAuthStore } from "../../stores/auth";
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { generatedApi } from "../../api/client";
import { apiErrorMessage } from "../../api/errors";
import {
  eventAssets,
  loadPaper,
  type EventKind,
  type Paper,
  type PaperProblem,
} from "../../api/modules/events";
import {
  adaptExamRanking,
  adaptTeamContestRanking,
} from "../../features/leaderboard/adapters";
import EventReviewPanel from "../../components/events/EventReviewPanel.vue";
import EventAttachments from "../../components/events/EventAttachments.vue";
import MarkdownRenderer from "../../components/MarkdownRenderer.vue";
import ProblemStatementView from "../../components/ProblemStatementView.vue";
import LeaderboardBoard from "../../components/LeaderboardBoard.vue";
const auth = useAuthStore();
const props = defineProps<{ reviewMode?: boolean }>(),
  route = useRoute(),
  router = useRouter();
const kind = computed(() => route.params.kind as EventKind),
  validKind = computed(() => kind.value === "exam" || kind.value === "contest"),
  eventID = computed(() => Number(route.params.eventId));
const paper = ref<Paper>(),
  selected = ref<PaperProblem>(),
  tab = ref("description"),
  error = ref(""),
  rankingError = ref(""),
  rankingLoading = ref(false),
  ranking = ref<any>();
const rankingData = computed(() =>
  kind.value === "exam"
    ? adaptExamRanking(ranking.value || { exam: paper.value?.event })
    : adaptTeamContestRanking(ranking.value || { contest: paper.value?.event }),
);
let sequence = 0;
async function load() {
  const current = ++sequence;
  paper.value = undefined;
  selected.value = undefined;
  ranking.value = undefined;
  error.value = "";
  tab.value = "description";
  if (!validKind.value || !eventID.value) {
    error.value = "场次编号无效";
    return;
  }
  if (props.reviewMode) return;
  try {
    const result = await loadPaper(kind.value, eventID.value, true);
    if (current === sequence) {
      paper.value = result;
      selected.value = result.problems[0];
    }
  } catch (e) {
    if (current === sequence) error.value = apiErrorMessage(e);
  }
}
async function loadRanking() {
  if (!paper.value?.can_view_ranking || rankingLoading.value) return;
  const current = sequence;
  rankingLoading.value = true;
  rankingError.value = "";
  try {
    const result =
      kind.value === "exam"
        ? await generatedApi.examRanking({ path: { id: eventID.value } })
        : await generatedApi.teamContestRankingViaContestsContestIdRanking({
            path: { contest_id: eventID.value },
          });
    if (current === sequence) ranking.value = result.data;
  } catch (e) {
    if (current === sequence) rankingError.value = apiErrorMessage(e);
  } finally {
    rankingLoading.value = false;
  }
}
watch(() => [kind.value, eventID.value, props.reviewMode], load, {
  immediate: true,
});
watch(tab, (value) => {
  if (value === "ranking") void loadRanking();
});
</script>
<style scoped>
.competition-detail {
  padding: 24px;
  max-width: 1440px;
  margin: auto;
  display: grid;
  gap: 16px;
  min-width: 0;
}
.competition-detail > .el-button {
  justify-self: start;
}
header p {
  color: var(--muted);
}
h1 {
  overflow-wrap: anywhere;
}
.description {
  padding: 22px;
  min-width: 0;
}
.problem-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.problem-buttons .el-button {
  margin: 0;
  max-width: 100%;
  height: auto;
  min-height: 34px;
  white-space: normal;
}
@media (max-width: 600px) {
  .competition-detail {
    padding: 14px;
  }
  .description {
    padding: 14px;
  }
}
</style>
