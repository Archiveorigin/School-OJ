import { ref, watch, nextTick, onBeforeUnmount, type Ref } from "vue";
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
  watch(
    () => [data().revealUserId, data().revealProblemId, data().replayStep],
    async () => {
      if (!data().replaying || data().revealUserId === undefined) return;
      await nextTick();
      root.value
        ?.querySelector<HTMLElement>('[data-reveal-focus="true"]')
        ?.scrollIntoView?.({
          block: "nearest",
          inline: "nearest",
          behavior: reduced.value ? "instant" : "smooth",
        });
    },
  );
  return { root, expandedId, instanceId, deltas, toggleRow };
}
