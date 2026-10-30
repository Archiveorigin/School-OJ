import { shallowRef, watch, onBeforeUnmount } from "vue";
import type { LeaderboardRow } from "./types";
import { MOTION } from "./motion";
export type RankDelta = { delta: number; isNew: boolean };
export function useRankDelta(
  rows: () => LeaderboardRow[],
  resetKey: () => unknown = () => undefined,
) {
  const changes = shallowRef<Record<string, RankDelta>>({});
  let previous = new Map<string, number>();
  let initialized = false;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let key = resetKey();
  let signature = "";
  watch(
    () =>
      [
        rows().map((r) => [String(r.id), r.rank, Boolean(r.unranked)] as const),
        resetKey(),
      ] as const,
    ([current, nextKey]) => {
      const nextSignature = JSON.stringify(current);
      if (nextSignature === signature && nextKey === key) return;
      signature = nextSignature;
      clearTimeout(timer);
      const next: Record<string, RankDelta> = {};
      if (nextKey !== key) {
        initialized = false;
        key = nextKey;
      }
      if (initialized)
        for (const [id, rank, unranked] of current) {
          if (unranked) continue;
          const old = previous.get(id);
          if (old === undefined) next[id] = { delta: 0, isNew: true };
          else if (old !== rank) next[id] = { delta: old - rank, isNew: false };
        }
      previous = new Map(current.filter((r) => !r[2]).map((r) => [r[0], r[1]]));
      initialized = current.length > 0;
      changes.value = next;
      if (Object.keys(next).length)
        timer = setTimeout(() => (changes.value = {}), MOTION.rankBadge);
    },
    { immediate: true },
  );
  onBeforeUnmount(() => clearTimeout(timer));
  return changes;
}
