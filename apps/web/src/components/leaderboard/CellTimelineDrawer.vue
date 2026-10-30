<template>
  <el-drawer
    :model-value="true"
    :class="['scoreboard-timeline', `theme-${theme || 'default'}`]"
    :title="`${row.name} · ${problem.label}`"
    size="min(480px, 100vw)"
    :append-to="teleportTarget"
    append-to-body
    @close="$emit('close')"
  >
    <p>
      {{ resultLabel }} · {{ result.attempts }} 次尝试
      <span v-if="result.secondary">· {{ result.secondary }}</span>
    </p>
    <el-alert
      v-if="!mayLoad"
      :title="
        replaying && result.status === 'frozen'
          ? '该题尚未揭晓，仅显示封榜汇总。'
          : '仅显示榜单汇总；提交明细只向本人或具有查看权限的管理员开放。'
      "
      type="info"
      :closable="false"
    />
    <template v-else
      ><el-button :loading="loading" @click="load">刷新提交</el-button>
      <p class="timeline-note">最近 200 条可见提交中的本场记录。</p>
      <el-alert
        v-if="error"
        :title="error"
        type="warning"
        :closable="false"
      /><el-empty
        v-else-if="!loading && !items.length"
        description="暂无可见提交，当前显示榜单汇总"
      />
      <ol class="cell-timeline">
        <li v-for="item in items" :key="item.id">
          <time>{{ formatDateTime(item.created_at) }}</time
          ><SubmissionStatusMark :status="item.status" compact /><span>{{
            item.status === "recorded"
              ? "结果仅本人可见 / 封榜中"
              : `${languageLabel(item.language)} · ${item.score} 分`
          }}</span
          ><el-button link @click="showSubmission(item.id)"
            >查看提交 #{{ item.id }}</el-button
          >
        </li>
      </ol>
    </template>
    <SubmissionDetailDialog v-model="detailOpen" :detail="detail" />
  </el-drawer>
</template>
<script setup lang="ts">
import { computed, ref, watch, onBeforeUnmount } from "vue";
import { generatedApi, type Submission } from "../../api/client";
import { useAuthStore } from "../../stores/auth";
import { apiErrorMessage } from "../../api/errors";
import type {
  LeaderboardRow,
  LeaderboardProblem,
  LeaderboardResult,
  LeaderboardContext,
} from "../../features/leaderboard/types";
import { formatDateTime } from "../../features/time";
import { languageLabel } from "../../features/submissions/presentation";
import SubmissionStatusMark from "../SubmissionStatusMark.vue";
import SubmissionDetailDialog from "../SubmissionDetailDialog.vue";
const props = defineProps<{
  row: LeaderboardRow;
  problem: LeaderboardProblem;
  result: LeaderboardResult;
  context?: LeaderboardContext;
  replaying?: boolean;
  theme?: string;
}>();
defineEmits<{ close: [] }>();
const teleportTarget =
  typeof document !== "undefined"
    ? (document.fullscreenElement as HTMLElement) || "body"
    : "body";
const auth = useAuthStore(),
  items = ref<Submission[]>([]),
  loading = ref(false),
  error = ref(""),
  detailOpen = ref(false),
  detail = ref<any>(null);
const mayLoad = computed(() =>
  Boolean(
    props.context &&
    auth.isAuthed &&
    (Number(props.row.id) === auth.user?.id ||
      auth.can("submission:read:any")) &&
    !(props.replaying && props.result.status === "frozen"),
  ),
);
const resultLabel = computed(
  () =>
    ({
      accepted: "已通过",
      wrong: "未通过",
      pending: "评测中",
      frozen: "封榜中",
      none: "未提交",
    })[props.result.status],
);
let controller: AbortController | undefined;
let sequence = 0;
async function load() {
  controller?.abort();
  const current = ++sequence;
  items.value = [];
  error.value = "";
  if (!mayLoad.value || !props.context) return;
  controller = new AbortController();
  loading.value = true;
  try {
    const context = props.context;
    let data: Submission[];
    if (context.kind === "contest" && Number(props.row.id) === auth.user?.id) {
      data = (
        await generatedApi.listTeamContestSubmissionsViaContestsContestIdSubmissions<
          Submission[]
        >({ path: { contest_id: context.eventId }, signal: controller.signal })
      ).data;
    } else {
      data = (
        await generatedApi.listSubmissions({
          query: {
            problem_id: props.problem.id,
            ...(context.kind === "exam" ? { exam_id: context.eventId } : {}),
            ...(auth.can("submission:read:any") ? {} : { mine: true }),
          },
          signal: controller.signal,
        })
      ).data;
    }
    if (current !== sequence) return;
    items.value = data
      .filter(
        (s) =>
          s.user_id === Number(props.row.id) &&
          s.problem_id === Number(props.problem.id) &&
          (context.kind === "contest"
            ? s.team_contest_id === context.eventId
            : s.exam_id === context.eventId),
      )
      .sort((a, b) => a.id - b.id);
  } catch (e) {
    if (current === sequence && !controller?.signal.aborted)
      error.value = apiErrorMessage(e, "提交记录不可用，当前仅显示汇总");
  } finally {
    if (current === sequence) loading.value = false;
  }
}
async function showSubmission(id: number) {
  const current = sequence;
  try {
    const { data } = await generatedApi.getSubmission({ path: { id } });
    if (current === sequence && mayLoad.value) {
      detail.value = data;
      detailOpen.value = true;
    }
  } catch (e) {
    error.value = apiErrorMessage(e);
  }
}
watch(
  () => [
    props.row.id,
    props.problem.id,
    props.context?.kind,
    props.context?.eventId,
    mayLoad.value,
  ],
  () => {
    detail.value = null;
    detailOpen.value = false;
    void load();
  },
  { immediate: true },
);
onBeforeUnmount(() => {
  sequence++;
  controller?.abort();
});
</script>
<style scoped>
.cell-timeline {
  list-style: none;
  padding: 0;
  display: grid;
  gap: 16px;
}
.cell-timeline li {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding: 12px 0;
  border-bottom: 1px solid var(--border);
}
time {
  width: 100%;
  font-size: 12px;
  color: var(--muted);
}
.timeline-note {
  font-size: 12px;
  color: var(--muted);
}
</style>
