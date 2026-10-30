<template>
  <button
    type="button"
    :class="[
      variant === 'penalty' ? 'penalty-result' : 'score-result',
      `result-${result.status}`,
      {
        'is-first': result.firstBlood && result.status === 'accepted',
        'is-revealing': revealing,
        'is-unfreezing': unfreezing,
      },
    ]"
    :data-cell-key="`${rowId}:${problem.id}`"
    :data-reveal="reveal"
    :aria-label="label"
    :title="label"
    @click.stop="$emit('open')"
  >
    <span
      v-if="result.firstBlood && result.status === 'accepted'"
      class="first-blood"
      aria-label="最快通过"
      >★</span
    ><span>{{
      result.primary || (result.status === "none" ? "—" : problem.label)
    }}</span
    ><template v-if="result.secondary"
      ><span class="result-separator">|</span
      ><small>{{ result.secondary }}</small></template
    >
  </button>
</template>
<script setup lang="ts">
import { computed, ref, watch, onBeforeUnmount } from "vue";
import type {
  LeaderboardProblem,
  LeaderboardResult,
} from "../../features/leaderboard/types";
import { MOTION } from "../../features/leaderboard/motion";
import { useReducedMotion } from "../../features/leaderboard/useReducedMotion";
const props = defineProps<{
  result: LeaderboardResult;
  problem: LeaderboardProblem;
  rowId: number | string;
  rowName: string;
  variant: "penalty" | "score";
  reveal?: boolean;
  animate?: boolean;
  speed?: number;
  replayStep?: number;
}>();
defineEmits<{ open: [] }>();
const reduced = useReducedMotion(),
  revealing = ref(false),
  unfreezing = ref(false);
let timer: ReturnType<typeof setTimeout> | undefined;
const label = computed(
  () =>
    `${props.rowName} · ${props.problem.label} · ${{ accepted: "已通过", wrong: "未通过", pending: "评测中", frozen: "封榜中", none: "未提交" }[props.result.status]}${props.result.status === "frozen" ? "" : ` · ${props.result.primary || ""} ${props.result.secondary || ""}`}`,
);
watch(
  () => [
    props.result.status,
    props.result.firstBlood,
    props.reveal,
    props.replayStep,
  ],
  (next, old) => {
    clearTimeout(timer);
    revealing.value = false;
    unfreezing.value = false;
    if (reduced.value || props.animate === false) return;
    if (
      (props.reveal && next[3] !== old[3]) ||
      next[0] !== old[0] ||
      (next[1] && !old[1])
    ) {
      revealing.value = true;
      unfreezing.value = old[0] === "frozen" && next[0] === "accepted";
      timer = setTimeout(
        () => {
          revealing.value = false;
          unfreezing.value = false;
        },
        MOTION.cellReveal / (props.speed || 1),
      );
    }
  },
);
onBeforeUnmount(() => clearTimeout(timer));
</script>
<style scoped>
button {
  font-family: inherit;
  cursor: pointer;
  padding: 0;
  position: relative;
  isolation: isolate;
  touch-action: manipulation;
}
button:focus-visible {
  outline: 2px solid var(--score-blue, var(--el-color-primary));
  outline-offset: 2px;
}
button:active {
  transform: scale(0.94);
}
small {
  font-size: 12px;
}
.result-separator {
  opacity: 0.65;
}
.first-blood {
  position: absolute;
  right: 3px;
  top: -7px;
  color: var(--score-first-marker);
  font-size: 15px;
  text-shadow: 0 1px 2px var(--score-surface);
  z-index: 1;
}
.result-frozen {
  background-image: repeating-linear-gradient(
    135deg,
    transparent 0 6px,
    color-mix(in srgb, var(--color-on-emphasis) 16%, transparent) 6px 11px
  );
}
.is-revealing {
  animation: cell-pop var(--score-motion-cell, 520ms) var(--score-motion-ease);
  z-index: 2;
}
.is-unfreezing::after {
  content: "";
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: repeating-linear-gradient(
    135deg,
    var(--el-color-primary) 0 6px,
    color-mix(in srgb, var(--el-color-primary) 65%, transparent) 6px 11px
  );
  animation: cell-wipe var(--score-motion-cell, 520ms) var(--score-motion-ease)
    forwards;
  pointer-events: none;
}
@keyframes cell-pop {
  0%,
  100% {
    transform: scale(1);
  }
  38% {
    transform: scale(1.12);
  }
}
@keyframes cell-wipe {
  from {
    clip-path: inset(0);
    opacity: 1;
  }
  to {
    clip-path: inset(0 0 0 100%);
    opacity: 0;
  }
}
@media (prefers-reduced-motion: reduce) {
  .is-revealing,
  .is-unfreezing::after {
    animation: none;
  }
  .is-unfreezing::after {
    display: none;
  }
  button:active {
    transform: none;
  }
}
</style>
