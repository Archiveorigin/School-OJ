import { mount, flushPromises } from "@vue/test-utils";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import ContestReplay from "../src/components/events/ContestReplay.vue";
vi.mock("../src/stores/auth", () => ({
  useAuthStore: () => ({ can: () => true }),
}));
vi.mock("../src/api/client", () => ({
  generatedApi: { getContestReplay: vi.fn(async () => ({ data: packet() })) },
}));
function packet() {
  const row = (id: number, accepted = false) => ({
    user_id: id,
    name: `P${id}`,
    rank: id,
    solved: accepted ? 1 : 0,
    total_score: accepted ? 100 : 0,
    penalty_minutes: accepted ? 20 + id : 0,
    submission_count: accepted ? 1 : 0,
    problems: [
      {
        problem_id: 1,
        status: accepted ? "accepted" : "",
        attempts: accepted ? 1 : 0,
        submission_count: accepted ? 1 : 0,
        best_score: accepted ? 100 : 0,
        wrong_attempts: 0,
        elapsed_minutes: accepted ? 20 + id : 0,
      },
    ],
  });
  return {
    schema_version: 1,
    kind: "contest-replay",
    contest: { scoring_rule: "acm", duration_minutes: 120 },
    problems: [{ problem_id: 1, label: "A" }],
    initial_rows: [row(1), row(2), row(3)],
    final_rows: [row(1, true), row(2, true), row(3, true)],
    events: [],
  };
}
const stubs = {
  ElDialog: {
    props: ["modelValue"],
    emits: ["close", "update:modelValue"],
    template: '<section v-if="modelValue" role="dialog"><slot/></section>',
  },
  ElButton: {
    props: ["disabled", "loading"],
    template: '<button :disabled="disabled"><slot/></button>',
  },
  ElAlert: true,
  LeaderboardBoard: {
    props: ["data"],
    template:
      '<div class="fake-board" :data-step="data.replayStep"><slot name="replay-reload"/><strong>滚榜回放</strong><button class="cell">cell</button><input/><div class="blank">blank</div></div>',
  },
};
let wrapper: ReturnType<typeof mount>;
const button = (label: string) =>
  wrapper.findAll("button").find((b) => b.text() === label)!;
const progress = () => wrapper.get(".fake-board").attributes("data-step");
beforeEach(async () => {
  vi.useFakeTimers();
  vi.spyOn(document, "hidden", "get").mockReturnValue(false);
  wrapper = mount(ContestReplay, {
    props: { kind: "contest", eventId: 9 },
    attachTo: document.body,
    global: { stubs },
  });
  await button("赛后滚榜").trigger("click");
  await flushPromises();
});
afterEach(() => {
  wrapper.unmount();
  vi.useRealTimers();
  vi.restoreAllMocks();
});
describe("keyboard-only replay flow", () => {
  it("shows only reload, starts at zero and advances once for each physical key press", async () => {
    expect(progress()).toBe("0");
    expect(wrapper.find(".replay-tools").exists()).toBe(false);
    expect(wrapper.find("progress").exists()).toBe(false);
    await button("重新加载").trigger("keydown", { key: "ArrowRight" });
    expect(progress()).toBe("1");
    await wrapper.get(".replay-surface").trigger("keydown", { key: "Enter" });
    expect(progress()).toBe("1");
    await vi.advanceTimersByTimeAsync(900);
    await wrapper
      .get(".replay-surface")
      .trigger("keydown", { key: " ", repeat: true });
    expect(progress()).toBe("1");
    await wrapper.get(".replay-surface").trigger("keydown", { key: " " });
    expect(progress()).toBe("2");
    await vi.advanceTimersByTimeAsync(2000);
    expect(progress()).toBe("2");
  });
  it("ignores mouse clicks, editing, modifiers and native button activation keys", async () => {
    await wrapper.get("input").trigger("keydown", { key: "ArrowRight" });
    await wrapper
      .get(".replay-surface")
      .trigger("keydown", { key: "ArrowRight", ctrlKey: true });
    await wrapper.get(".cell").trigger("click");
    await wrapper.get(".blank").trigger("click");
    await button("重新加载").trigger("keydown", { key: " " });
    expect(progress()).toBe("0");
  });
  it("Home cancels the current move and reload resets the entire engine", async () => {
    await wrapper.get(".replay-surface").trigger("keydown", { key: "Enter" });
    expect(progress()).toBe("1");
    await wrapper.get(".replay-surface").trigger("keydown", { key: "Home" });
    await flushPromises();
    expect(progress()).toBe("0");
    await vi.advanceTimersByTimeAsync(2500);
    expect(progress()).toBe("0");
    await wrapper
      .get(".replay-surface")
      .trigger("keydown", { key: "ArrowRight" });
    await button("重新加载").trigger("click");
    await flushPromises();
    expect(progress()).toBe("0");
    expect(document.activeElement).toBe(wrapper.get(".replay-surface").element);
  });
});
