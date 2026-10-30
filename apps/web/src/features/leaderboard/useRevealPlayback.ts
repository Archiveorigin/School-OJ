import { ref, shallowRef, computed, onMounted, onBeforeUnmount } from "vue";
import type { LeaderboardData } from "./types";
import { stepDuration } from "./motion";
import { useReducedMotion } from "./useReducedMotion";
export interface RevealEngine {
  total: number;
  readonly step: number;
  snapshot(): LeaderboardData;
  next(): LeaderboardData;
}
export function useRevealPlayback() {
  const board = shallowRef<LeaderboardData>(),
    speed = ref(1),
    playing = ref(false),
    moving = ref(false),
    status = ref("idle"),
    step = ref(0),
    total = ref(0),
    run = ref(0),
    error = ref("");
  const reduced = useReducedMotion();
  let engine: RevealEngine | undefined;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const decorate = (data: LeaderboardData) => ({
    ...data,
    replayStep: step.value,
    replayRunId: run.value,
    motionSpeed: speed.value,
  });
  const finished = computed(
    () => Boolean(board.value) && step.value >= total.value,
  );
  function pause() {
    playing.value = false;
    if (!moving.value) {
      clearTimeout(timer);
      status.value = finished.value ? "finished" : "paused";
    }
  }
  function dispose() {
    clearTimeout(timer);
    playing.value = false;
    moving.value = false;
    engine = undefined;
    board.value = undefined;
    step.value = 0;
    total.value = 0;
    status.value = "idle";
    error.value = "";
    run.value++;
  }
  function install(value: RevealEngine) {
    dispose();
    engine = value;
    total.value = value.total;
    board.value = decorate(value.snapshot());
    error.value = board.value.consistencyError || "";
    status.value = error.value ? "error" : total.value ? "ready" : "finished";
  }
  function advance() {
    if (!engine || moving.value || finished.value) return;
    clearTimeout(timer);
    moving.value = true;
    status.value = "stepping";
    try {
      const data = engine.next();
      step.value = engine.step;
      board.value = decorate(data);
      if (data.consistencyError) {
        error.value = data.consistencyError;
        playing.value = false;
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : "滚榜失败";
      playing.value = false;
      moving.value = false;
      status.value = "error";
      return;
    }
    timer = setTimeout(
      () => {
        moving.value = false;
        status.value = error.value
          ? "error"
          : finished.value
            ? "finished"
            : playing.value
              ? "playing"
              : "paused";
        if (finished.value) playing.value = false;
        else if (playing.value) advance();
      },
      stepDuration(speed.value, board.value?.rows.length, reduced.value),
    );
  }
  function next() {
    if (playing.value || error.value) return;
    advance();
  }
  function play() {
    if (!engine || finished.value || error.value || document.hidden) return;
    playing.value = true;
    if (!moving.value) advance();
  }
  function toggle() {
    if (playing.value) pause();
    else play();
  }
  function visibility() {
    if (document.hidden) pause();
  }
  onMounted(() => document.addEventListener("visibilitychange", visibility));
  onBeforeUnmount(() => {
    dispose();
    document.removeEventListener("visibilitychange", visibility);
  });
  return {
    board,
    speed,
    playing,
    moving,
    status,
    step,
    total,
    finished,
    error,
    install,
    dispose,
    pause,
    next,
    toggle,
  };
}
