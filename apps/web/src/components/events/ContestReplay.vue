<template>
  <el-button @click="open = true">赛后滚榜</el-button>
  <el-dialog
    v-model="open"
    title="赛后滚榜"
    width="min(1500px, calc(100vw - 24px))"
    @closed="stop"
  >
    <div class="replay-tools">
      <el-button v-if="canExport" :loading="busy" @click="download"
        >导出滚榜数据包</el-button
      ><el-button @click="fileInput?.click()">导入离线数据包</el-button
      ><input
        ref="fileInput"
        hidden
        type="file"
        accept="application/json,.json"
        @change="importFile"
      /><el-button
        :disabled="!frames.length"
        @click="
          step = 0;
          stop();
        "
        >从封榜开始</el-button
      ><el-button :disabled="step >= frames.length - 1" @click="next"
        >揭晓下一题</el-button
      ><el-button
        :disabled="!frames.length"
        @click="playing ? stop() : play()"
        >{{ playing ? "暂停" : "自动滚榜" }}</el-button
      ><span>{{ step }} / {{ Math.max(0, frames.length - 1) }}</span>
    </div>
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <p>
      导出须等待比赛结束且所有评测完成。数据包包含封榜快照、最终过题情况与封榜后的提交事件，不包含源代码；导入后可离线逐题揭晓。
    </p>
    <LeaderboardBoard v-if="frames[step]" :data="frames[step]" />
  </el-dialog>
</template>
<script setup lang="ts">
import { ref, onUnmounted } from "vue";
import LeaderboardBoard from "../LeaderboardBoard.vue";
import { generatedApi } from "../../api/client";
import {
  eventPath,
  saveDownload,
  type EventKind,
} from "../../api/modules/events";
import { apiErrorMessage } from "../../api/errors";
import { parseReplay, replayFrames } from "../../features/leaderboard/replay";
import type { LeaderboardData } from "../../features/leaderboard/types";
const props = defineProps<{
  kind: EventKind;
  eventId: number;
  canExport: boolean;
}>();
const fileInput = ref<HTMLInputElement>();
const open = ref(false),
  busy = ref(false),
  error = ref(""),
  frames = ref<LeaderboardData[]>([]),
  step = ref(0),
  playing = ref(false);
let timer: ReturnType<typeof setInterval> | undefined;
function stop() {
  clearInterval(timer);
  playing.value = false;
}
function next() {
  if (step.value < frames.value.length - 1) step.value++;
  else stop();
}
function play() {
  stop();
  playing.value = true;
  timer = setInterval(next, 1500);
}
onUnmounted(stop);
async function download() {
  stop();
  busy.value = true;
  error.value = "";
  try {
    const { data } = await generatedApi.exportEventRanking<Blob>({
      path: eventPath(props.kind, props.eventId),
      query: { format: "json" },
      responseType: "blob",
    });
    saveDownload(data, `contest-${props.eventId}-replay.json`);
    frames.value = replayFrames(parseReplay(await data.text()));
    step.value = 0;
  } catch (e: any) {
    if (e.response?.data instanceof Blob) {
      try {
        e.response.data = JSON.parse(await e.response.data.text());
      } catch {
        /* retain the transport error */
      }
    }
    error.value = apiErrorMessage(e);
  } finally {
    busy.value = false;
  }
}
async function importFile(e: Event) {
  stop();
  error.value = "";
  try {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file) return;
    if (file.size > 32 * 1024 * 1024) throw new Error("数据包超过 32 MiB");
    frames.value = replayFrames(parseReplay(await file.text()));
    step.value = 0;
  } catch (e) {
    error.value = apiErrorMessage(e);
  }
}
</script>
<style scoped>
.replay-tools {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.replay-tools label {
  display: flex;
  gap: 8px;
  align-items: center;
  max-width: 100%;
}
input {
  max-width: 220px;
}
</style>
