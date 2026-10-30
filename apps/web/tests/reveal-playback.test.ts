import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { mount } from "@vue/test-utils";
import { defineComponent } from "vue";
import { useRevealPlayback } from "../src/features/leaderboard/useRevealPlayback";
import type { LeaderboardData } from "../src/features/leaderboard/types";
const data = { rows: [], problems: [] } as unknown as LeaderboardData;
function engine(total = 3) {
  let step = 0;
  return {
    total,
    get step() {
      return step;
    },
    snapshot: () => data,
    next: () => {
      step++;
      return data;
    },
  };
}
beforeEach(() => {
  vi.useFakeTimers();
  vi.spyOn(document, "hidden", "get").mockReturnValue(false);
});
afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});
function setup() {
  let player!: ReturnType<typeof useRevealPlayback>;
  const wrapper = mount(
    defineComponent({
      setup() {
        player = useRevealPlayback();
        return () => null;
      },
    }),
  );
  return { player, wrapper };
}
it("plays sequentially, guards duplicate steps, pauses and disposes timers", async () => {
  const { player, wrapper } = setup();
  player.install(engine());
  player.next();
  player.next();
  expect(player.step.value).toBe(1);
  await vi.advanceTimersByTimeAsync(821);
  player.toggle();
  expect(player.step.value).toBe(2);
  player.pause();
  await vi.advanceTimersByTimeAsync(2000);
  expect(player.step.value).toBe(2);
  player.toggle();
  await vi.advanceTimersByTimeAsync(821);
  expect(player.finished.value).toBe(true);
  expect(player.playing.value).toBe(false);
  wrapper.unmount();
  expect(vi.getTimerCount()).toBe(0);
});
it("shows zero-step finals and resets progress when a new engine is installed", () => {
  const { player, wrapper } = setup();
  expect(player.finished.value).toBe(false);
  player.install(engine(0));
  expect(player.finished.value).toBe(true);
  player.next();
  expect(player.step.value).toBe(0);
  player.install(engine(2));
  expect(player.finished.value).toBe(false);
  expect(player.step.value).toBe(0);
  wrapper.unmount();
});
it("pauses when the browser tab is hidden", async () => {
  const { player, wrapper } = setup();
  player.install(engine());
  player.toggle();
  vi.spyOn(document, "hidden", "get").mockReturnValue(true);
  document.dispatchEvent(new Event("visibilitychange"));
  await vi.advanceTimersByTimeAsync(3000);
  expect(player.step.value).toBe(1);
  expect(player.playing.value).toBe(false);
  wrapper.unmount();
});
