<template>
  <el-button @click="start">赛后滚榜</el-button>
  <el-dialog
    v-model="open"
    title="赛后滚榜"
    fullscreen
    append-to-body
    :show-close="false"
    class="contest-replay"
    header-class="replay-hidden-header"
    @close="reset"
  >
    <div
      ref="surface"
      class="replay-surface"
      data-replay-surface
      tabindex="0"
      @keydown="hotkey"
    >
      <el-alert
        v-if="error || playbackError"
        :title="error || playbackError"
        type="error"
        :closable="false"
      />
      <div
        class="replay-stage"
        :aria-busy="moving || busy"
        aria-label="滚榜演示区域：空格、回车或方向右键揭晓，Home 重播，F 全屏，Escape 退出"
      >
        <LeaderboardBoard
          v-if="board"
          :key="board.replayRunId"
          :data="{ ...board, motionSpeed: 1 }"
          :context="{ kind, eventId }"
        >
          <template #replay-reload
            ><el-button :loading="busy" :disabled="busy" @click="load"
              >重新加载</el-button
            ></template
          >
        </LeaderboardBoard>
        <el-button v-else :loading="busy" :disabled="busy" @click="load"
          >重新加载</el-button
        >
      </div>
    </div>
  </el-dialog>
</template>
<script setup lang="ts">
import { ref, watch, onBeforeUnmount, nextTick } from "vue";
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
  surface = ref<HTMLElement>();
let packet: ReplayPacket | undefined,
  generation = 0,
  controller: AbortController | undefined;
const {
  board,
  playing,
  moving,
  error: playbackError,
  install,
  dispose,
  next,
  pause,
} = useRevealPlayback();
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
  void focusSurface();
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

    void focusSurface();
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
async function focusSurface() {
  await nextTick();
  if (!open.value) return;
  surface.value?.focus({ preventScroll: true });
}
function hotkey(e: KeyboardEvent) {
  const target = e.target as HTMLElement;
  if (
    e.defaultPrevented ||
    e.altKey ||
    e.ctrlKey ||
    e.metaKey ||
    target.closest(
      'input,select,textarea,[contenteditable]:not([contenteditable="false"]),.el-drawer',
    )
  )
    return;
  if (
    (e.key === " " || e.key === "Enter") &&
    target.closest('button,a,[role="button"]')
  )
    return;
  if (![" ", "Enter", "ArrowRight", "Home", "f", "F"].includes(e.key)) return;
  e.preventDefault();
  if (e.repeat || busy.value) return;
  if (e.key === "Home") {
    pause();
    restart();
  } else if (e.key.toLowerCase() === "f") {
    void (
      document.fullscreenElement
        ? document.exitFullscreen()
        : surface.value?.requestFullscreen()
    )?.catch(() => {});
  } else next();
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
<style>
.contest-replay .replay-hidden-header {
  display: none;
}
</style>
<style scoped>
.replay-stage {
  padding-bottom: 50dvh;
}
.replay-surface {
  box-sizing: border-box;
  height: calc(100dvh - 32px);
  overflow: auto;
  outline: none;
}
.replay-surface:fullscreen {
  box-sizing: border-box;
  width: 100%;
  height: 100dvh;
  overflow: auto;
  padding: 16px;
  background: var(--surface);
  color: var(--text);
}
</style>
