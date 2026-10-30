<template>
  <el-button @click="start">赛后滚榜</el-button>
  <el-dialog
    v-model="open"
    title="赛后滚榜"
    fullscreen
    append-to-body
    @closed="reset"
  >
    <div class="replay-tools">
      <el-button :loading="busy" @click="load">重新加载</el-button>
      <el-button :disabled="!engine || busy" @click="restart"
        >从封榜开始</el-button
      >
      <el-button
        type="primary"
        :disabled="!board || step >= total || moving || busy"
        @click="next"
        >揭晓下一题</el-button
      >
      <span role="status"
        >{{ step }} / {{ total
        }}{{ board && step === total ? " · 滚榜结束" : "" }}</span
      >
    </div>
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <p>
      点击榜单或“揭晓下一题”逐题揭晓；比赛结束且评测完成后可用。选手最终榜单在管理员确认完结后公开。
    </p>
    <div class="replay-stage" :aria-busy="moving || busy" @click="clickBoard">
      <LeaderboardBoard v-if="board" :data="board" @refresh="load" />
      <el-skeleton v-else-if="busy" :rows="10" animated />
    </div>
  </el-dialog>
</template>
<script setup lang="ts">
import { ref, shallowRef, onUnmounted, watch, nextTick } from "vue";
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
import type { LeaderboardData } from "../../features/leaderboard/types";
const props = defineProps<{ kind: EventKind; eventId: number }>();
const auth = useAuthStore();
const open = ref(false),
  busy = ref(false),
  error = ref(""),
  moving = ref(false),
  step = ref(0),
  total = ref(0);
const board = shallowRef<LeaderboardData>();
const engine = shallowRef<ReturnType<typeof createContestReplay>>();
let packet: ReplayPacket | undefined,
  generation = 0;
let animationTimer: ReturnType<typeof setTimeout> | undefined;
function reset() {
  generation++;
  clearTimeout(animationTimer);
  moving.value = false;
  busy.value = false;
  board.value = undefined;
  engine.value = undefined;
  packet = undefined;
}
function restart() {
  if (!packet) return;
  clearTimeout(animationTimer);
  moving.value = false;
  engine.value = createContestReplay(packet);
  step.value = 0;
  total.value = engine.value.total;
  board.value = engine.value.snapshot();
}
async function start() {
  if (!auth.can("event:review")) return;
  open.value = true;
  await load();
}
async function load() {
  if (!auth.can("event:review") || busy.value) return;
  const request = ++generation;
  busy.value = true;
  error.value = "";
  board.value = undefined;
  engine.value = undefined;
  packet = undefined;
  try {
    const { data } = await generatedApi.getContestReplay({
      path: eventPath(props.kind, props.eventId),
    });
    if (request !== generation) return;
    packet = parseReplay(JSON.stringify(data));
    restart();
  } catch (e) {
    if (request === generation) error.value = apiErrorMessage(e);
  } finally {
    if (request === generation) busy.value = false;
  }
}
async function next() {
  if (
    !auth.can("event:review") ||
    !engine.value ||
    busy.value ||
    moving.value ||
    step.value >= total.value
  )
    return;
  moving.value = true;
  board.value = engine.value.next();
  step.value = engine.value.step;
  await nextTick();
  document
    .querySelector('.replay-stage [data-reveal-focus="true"]')
    ?.scrollIntoView({
      block: "nearest",
      behavior: matchMedia("(prefers-reduced-motion: reduce)").matches
        ? "instant"
        : "smooth",
    });
  animationTimer = setTimeout(() => (moving.value = false), 750);
}
function clickBoard(event: MouseEvent) {
  if (
    !(event.target as HTMLElement).closest(
      "button,input,select,label,a,.summary-overlay",
    )
  )
    void next();
}
watch(
  () => [props.kind, props.eventId],
  () => {
    open.value = false;
    reset();
  },
);
watch(
  () => auth.can("event:review"),
  (allowed) => {
    if (!allowed) {
      open.value = false;
      reset();
    }
  },
);
onUnmounted(reset);
</script>
<style scoped>
.replay-tools {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  align-items: center;
}
.replay-stage {
  cursor: pointer;
}
.replay-stage :deep([data-reveal-focus="true"]) {
  outline: 2px solid var(--el-color-primary);
  outline-offset: -2px;
}
</style>
