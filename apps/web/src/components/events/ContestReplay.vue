<template>
  <el-button @click="start">赛后滚榜</el-button>
  <el-dialog
    v-model="open"
    title="赛后滚榜"
    fullscreen
    append-to-body
    @close="reset"
  >
    <div class="replay-tools">
      <el-button :loading="busy" :disabled="busy" @click="load"
        >重新加载</el-button
      >
      <el-button :disabled="!board || busy || playing" @click="restart"
        >从封榜开始</el-button
      >
      <el-button
        type="primary"
        :disabled="
          !board ||
          finished ||
          moving ||
          playing ||
          busy ||
          Boolean(playbackError)
        "
        @click="next"
        >揭晓下一题</el-button
      >
      <el-button
        :disabled="!board || finished || busy || Boolean(playbackError)"
        @click="toggle"
        >{{ playing ? "暂停" : "播放" }}</el-button
      >
      <label
        >播放速度
        <select v-model.number="speed" aria-label="播放速度">
          <option :value="0.5">0.5×</option>
          <option :value="1">1×</option>
          <option :value="2">2×</option>
        </select></label
      >
      <span role="status"
        >{{ step }} / {{ total
        }}{{ board && finished ? " · 滚榜结束" : "" }}</span
      >
    </div>
    <progress
      v-if="board && total"
      class="replay-progress"
      :value="step"
      :max="total"
      aria-label="滚榜进度"
    />
    <el-alert
      v-if="error || playbackError"
      :title="error || playbackError"
      type="error"
      :closable="false"
    />
    <p>
      使用按钮或键盘播放、暂停、逐题揭晓；Space 播放/暂停，方向右键下一题，Home
      重播。选手最终榜单在管理员确认完结后公开。
    </p>
    <p v-if="board && total === 0" class="replay-empty">
      本场没有可揭晓的封榜提交（可能未开启封榜，或封榜后没有提交）。当前显示最终榜单。
    </p>
    <p v-if="total > 1000">
      本场揭晓步骤较多，建议使用 2× 速度或查看最终榜单。
    </p>
    <p v-if="board" class="replay-announcement" aria-live="polite">
      {{ announcement }}
    </p>
    <div
      ref="stage"
      class="replay-stage"
      :aria-busy="moving || busy"
      tabindex="0"
      aria-label="滚榜演示区域"
      @keydown="hotkey"
    >
      <LeaderboardBoard
        v-if="board"
        :data="{ ...board, motionSpeed: speed }"
        :context="{ kind, eventId }"
        @refresh="load"
      />
      <el-skeleton v-else-if="busy" :rows="10" animated />
    </div>
  </el-dialog>
</template>
<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from "vue";
import LeaderboardBoard from "../LeaderboardBoard.vue";
import { useAuthStore } from "../../stores/auth";
import { generatedApi } from "../../api/client";
import { eventPath, type EventKind } from "../../api/modules/events";
import { apiErrorMessage } from "../../api/errors";
import {
  createContestReplay,
  parseReplay,
  type ReplayPacket,
} from "../../features/leaderboard/replay";
import { useRevealPlayback } from "../../features/leaderboard/useRevealPlayback";
const props = defineProps<{ kind: EventKind; eventId: number }>(),
  auth = useAuthStore();
const open = ref(false),
  busy = ref(false),
  error = ref(""),
  stage = ref<HTMLElement>();
let packet: ReplayPacket | undefined,
  generation = 0,
  controller: AbortController | undefined;
const {
  board,
  speed,
  playing,
  moving,
  step,
  total,
  finished,
  error: playbackError,
  install,
  dispose,
  next,
  toggle,
  pause,
} = useRevealPlayback();
const announcement = computed(() => {
  if (!board.value || board.value.revealUserId === undefined)
    return "从封榜快照开始，等待揭晓。";
  const row = board.value.rows.find((r) => r.id === board.value?.revealUserId),
    problem = board.value.problems.find(
      (p) => p.id === board.value?.revealProblemId,
    );
  if (!row || !problem) return "";
  const result = row.results[String(problem.id)];
  return `第 ${step.value}/${total.value} 步：${row.name} · ${problem.label} · ${result?.status === "accepted" ? "已通过" : result?.status === "wrong" ? "未通过" : "已揭晓"} · ${row.unranked ? "不参与排名" : `当前第 ${row.rank} 名`}`;
});
function reset() {
  generation++;
  controller?.abort();
  packet = undefined;
  busy.value = false;
  error.value = "";
  dispose();
}
function restart() {
  if (!packet || playing.value) return;
  install(createContestReplay(packet));
}
async function start() {
  if (!auth.can("event:review")) return;
  open.value = true;
  await load();
}
async function load() {
  if (!auth.can("event:review") || busy.value) return;
  dispose();
  const current = ++generation;
  controller?.abort();
  controller = new AbortController();
  busy.value = true;
  error.value = "";
  packet = undefined;
  try {
    const { data } = await generatedApi.getContestReplay({
      path: eventPath(props.kind, props.eventId),
      signal: controller.signal,
    });
    if (current !== generation) return;
    packet = parseReplay(JSON.stringify(data));
    install(createContestReplay(packet));
    if (total.value > 300) speed.value = 2;
  } catch (e: any) {
    if (current === generation && !controller?.signal.aborted)
      error.value =
        e.response?.status === 403
          ? "只有系统管理员可以查看赛后滚榜。"
          : e.response?.status === 409
            ? "比赛尚未截止或仍有提交未判完，暂不能滚榜。"
            : apiErrorMessage(e, "滚榜数据不完整，请刷新重试");
  } finally {
    if (current === generation) busy.value = false;
  }
}
function hotkey(e: KeyboardEvent) {
  if (
    (e.target as HTMLElement).closest(
      'button,input,select,textarea,[role="button"],.el-drawer',
    )
  )
    return;
  if (e.key === " ") {
    e.preventDefault();
    toggle();
  } else if (e.key === "ArrowRight") {
    e.preventDefault();
    next();
  } else if (e.key === "Home") {
    e.preventDefault();
    restart();
  }
}
watch(
  () => [props.kind, props.eventId, auth.can("event:review")],
  () => {
    open.value = false;
    reset();
  },
);
onBeforeUnmount(reset);
</script>
<style scoped>
.replay-tools {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  align-items: center;
}
.replay-tools label {
  display: flex;
  gap: 8px;
  align-items: center;
}
.replay-tools select {
  padding: 6px;
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text);
  border-radius: 4px;
}
.replay-progress {
  width: 100%;
  height: 8px;
  margin-top: 12px;
  accent-color: var(--el-color-primary);
}
.replay-stage:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 2px;
}
.replay-empty {
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
}
.replay-announcement {
  min-height: 24px;
  color: var(--muted);
}
</style>
