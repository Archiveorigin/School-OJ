import { ref, watch, nextTick, onBeforeUnmount, onMounted } from "vue";
import type { LeaderboardData, LeaderboardRow } from "./types";
import { useRankDelta } from "./useRankDelta";
import { useReducedMotion } from "./useReducedMotion";
let instance = 0;
export function useTableInteraction(
  data: () => LeaderboardData,
  rows: () => LeaderboardRow[],
) {
  const root = ref<HTMLElement>(),
    expandedId = ref<number | string>(),
    instanceId = `score-row-${++instance}`,
    reduced = useReducedMotion();
  const deltas = useRankDelta(
    () => data().rows,
    () => data().replayRunId,
  );
  function toggleRow(row: LeaderboardRow) {
    if (data().replaying) return;
    expandedId.value = expandedId.value === row.id ? undefined : row.id;
  }
  watch(
    [rows, () => data().replaying, () => data().replayRunId],
    () => (expandedId.value = undefined),
  );
  function centerCurrentRow() {
    if (!data().replaying) return;
    const row = root.value?.querySelector<HTMLElement>(
      '[data-reveal-focus="true"]',
    );
    const viewport = root.value?.closest<HTMLElement>("[data-replay-surface]");
    if (!row || !viewport) return;
    const bounds = row.getBoundingClientRect();
    const view = viewport.getBoundingClientRect();
    viewport.scrollTo({
      top: Math.max(
        0,
        viewport.scrollTop +
          bounds.top -
          view.top -
          viewport.clientTop +
          bounds.height / 2 -
          viewport.clientHeight / 2,
      ),
      behavior: reduced.value ? "instant" : "smooth",
    });
  }
  watch(
    () => [data().revealUserId, data().revealProblemId, data().replayStep],
    async () => {
      await nextTick();
      centerCurrentRow();
    },
    { immediate: true, flush: "post" },
  );
  onMounted(() => {
    window.addEventListener("resize", centerCurrentRow);
    root.value?.addEventListener("transitionend", centerCurrentRow);
  });
  onBeforeUnmount(() => {
    window.removeEventListener("resize", centerCurrentRow);
    root.value?.removeEventListener("transitionend", centerCurrentRow);
  });
  return { root, expandedId, instanceId, deltas, toggleRow };
}
