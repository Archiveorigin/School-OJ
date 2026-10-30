import { describe, it, expect, vi, afterEach } from "vitest";
import { mount } from "@vue/test-utils";
import { defineComponent, ref, nextTick } from "vue";
import { motionScale, stepDuration } from "../src/features/leaderboard/motion";
import { useRankDelta } from "../src/features/leaderboard/useRankDelta";
import { useAnimatedNumber } from "../src/features/leaderboard/useAnimatedNumber";
import type { LeaderboardRow } from "../src/features/leaderboard/types";
afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});
it("scales player timing and disables motion under reduced/large-board policy", () => {
  expect(stepDuration(1, 10, false)).toBe(820);
  expect(stepDuration(2, 10, false)).toBe(410);
  expect(stepDuration(0.5, 10, false)).toBe(1640);
  expect(stepDuration(1, 1001, false)).toBe(200);
  expect(stepDuration(1, 1, true)).toBe(200);
});
it("keeps first load quiet, tracks competition ranks and expires markers", async () => {
  vi.useFakeTimers();
  const rows = ref<LeaderboardRow[]>([]);
  let delta: any;
  const wrapper = mount(
    defineComponent({
      setup() {
        delta = useRankDelta(() => rows.value);
        return () => null;
      },
    }),
  );
  rows.value = [
    { id: 1, rank: 3 },
    { id: 2, rank: 1, unranked: true },
  ] as LeaderboardRow[];
  await nextTick();
  expect(delta.value).toEqual({});
  rows.value = [
    { id: 1, rank: 1 },
    { id: 2, rank: 2, unranked: true },
  ] as LeaderboardRow[];
  await nextTick();
  expect(delta.value["1"].delta).toBe(2);
  expect(delta.value["2"]).toBeUndefined();
  await vi.advanceTimersByTimeAsync(2401);
  expect(delta.value).toEqual({});
  wrapper.unmount();
  expect(vi.getTimerCount()).toBe(0);
});
it("finishes animated numbers exactly and releases the shared animation frame", async () => {
  vi.useFakeTimers();
  const target = ref(3);
  let value: any;
  const wrapper = mount(
    defineComponent({
      setup() {
        value = useAnimatedNumber(() => target.value);
        return () => null;
      },
    }),
  );
  expect(value.value).toBe(3);
  target.value = 17;
  await nextTick();
  await vi.advanceTimersByTimeAsync(700);
  expect(value.value).toBe(17);
  wrapper.unmount();
  expect(vi.getTimerCount()).toBe(0);
});
