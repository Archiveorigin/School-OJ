import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import PenaltyLeaderboardTable from "../src/components/PenaltyLeaderboardTable.vue";
import ScoreLeaderboardTable from "../src/components/ScoreLeaderboardTable.vue";
import type {
  LeaderboardData,
  LeaderboardRow,
} from "../src/features/leaderboard/types";

function participant(overrides: Partial<LeaderboardRow> = {}): LeaderboardRow {
  return {
    id: 17,
    rank: 1,
    name: "章佳荣",
    studentNo: "STUDENT_NO_SENTINEL",
    meta: "META_SENTINEL avatar@example.com 团队 组织 班级 教练 成员",
    solved: 1,
    metric: 35,
    metricDisplay: "35",
    submissions: 2,
    results: {
      "1": {
        status: "accepted",
        attempts: 2,
        primary: "2",
        secondary: "35'",
        firstBlood: true,
      },
    },
    ...overrides,
  } as LeaderboardRow;
}

function boardData(
  scoringRule: LeaderboardData["scoringRule"],
  rows: LeaderboardRow[],
): LeaderboardData {
  return {
    scoringRule,
    title: scoringRule === "score" ? "总分榜" : "罚时榜",
    durationSeconds: 7200,
    currentTimeSeconds: 3600,
    solvedLabel: scoringRule === "score" ? "满分" : "题数",
    metricLabel: scoringRule === "score" ? "总分" : "罚时",
    metricDirection: scoringRule === "score" ? "descending" : "ascending",
    problems: [
      { id: 1, label: "A", title: "第一题", color: "#ff46a0", maxScore: 100 },
    ],
    rows,
  };
}

describe("live scoreboard tables", () => {
  it("renders penalty rows with name-only identity and percentage award mappings", () => {
    const row = participant();
    const wrapper = mount(PenaltyLeaderboardTable, {
      props: {
        data: boardData("penalty", [row]),
        rows: [row],
        mode: "published",
        rankByID: { "17": 7 },
        awardByID: { "17": "gold" },
      },
    });

    expect(wrapper.text()).toContain("章佳荣");
    expect(wrapper.text()).toContain("排名Rank");
    expect(wrapper.text()).toContain("题数Solved");
    expect(wrapper.text()).toContain("罚时Penalty");
    expect(wrapper.text()).not.toContain("STUDENT_NO_SENTINEL");
    expect(wrapper.text()).not.toContain("META_SENTINEL");
    expect(wrapper.text()).not.toContain("avatar@example.com");
    expect(wrapper.text()).not.toContain("团队");
    expect(wrapper.find(".student-identity").exists()).toBe(false);
    expect(wrapper.find(".student-avatar").exists()).toBe(false);
    expect(wrapper.find(".penalty-identity-header").exists()).toBe(false);
    expect(wrapper.get(".penalty-participant-name").text()).toBe("章佳荣");
    expect(wrapper.get(".penalty-row").classes()).toContain("award-gold");
    expect(wrapper.get(".penalty-row").attributes("data-award")).toBe("gold");
    expect(wrapper.get(".penalty-rank-stat").classes()).toContain("is-gold");
    expect(wrapper.get(".penalty-rank-stat").text()).toBe("7");
  });

  it("renders score rows in the same native language without a separate identity column", () => {
    const row = participant({
      metric: 86,
      metricDisplay: "86/100",
      maxScore: 100,
      results: {
        "1": {
          status: "wrong",
          attempts: 2,
          primary: "86/100",
          secondary: "未满分",
        },
      },
    });
    const wrapper = mount(ScoreLeaderboardTable, {
      props: {
        data: boardData("score", [row]),
        rows: [row],
        mode: "performance",
        rankByID: { "17": 4 },
        awardByID: { "17": "bronze" },
      },
    });

    expect(wrapper.text()).toContain("章佳荣");
    expect(wrapper.text()).toContain("排名Rank");
    expect(wrapper.text()).toContain("满分Full");
    expect(wrapper.text()).toContain("总分Score");
    expect(wrapper.text()).not.toContain("STUDENT_NO_SENTINEL");
    expect(wrapper.text()).not.toContain("META_SENTINEL");
    expect(wrapper.find(".student-identity").exists()).toBe(false);
    expect(wrapper.find(".student-avatar").exists()).toBe(false);
    expect(wrapper.find(".score-identity-header").exists()).toBe(false);
    expect(wrapper.get(".score-participant-name").text()).toBe("章佳荣");
    expect(wrapper.get(".score-row").classes()).toContain("award-bronze");
    expect(wrapper.get(".score-rank-stat").classes()).toContain("is-bronze");
    expect(wrapper.get(".score-rank-stat").text()).toBe("4");
    expect(wrapper.get(".score-result").classes()).toContain("result-wrong");
  });

  it.each([
    [PenaltyLeaderboardTable, "penalty", ".penalty-header-track"],
    [ScoreLeaderboardTable, "score", ".score-header-track"],
  ] as const)(
    "keeps the %s sticky header as the horizontal scroller sibling and synchronizes it",
    async (component, rule, trackSelector) => {
      const row = participant();
      const wrapper = mount(component, {
        props: {
          data: boardData(rule, [row]),
          rows: [row],
          mode: "published",
          rankByID: { "17": 1 },
          awardByID: { "17": "none" },
        },
      });
      const branch = wrapper.get(`[data-scoreboard-branch="${rule}"]`);
      const stickyHeader = wrapper.get(`[data-sticky-header="${rule}"]`);
      const bodyScroller = wrapper.get(`[data-horizontal-scroll="${rule}"]`);

      expect(stickyHeader.element.parentElement).toBe(branch.element);
      expect(bodyScroller.element.parentElement).toBe(branch.element);
      expect(stickyHeader.element.contains(bodyScroller.element)).toBe(false);

      Object.defineProperty(bodyScroller.element, "scrollLeft", {
        configurable: true,
        value: 420,
      });
      await bodyScroller.trigger("scroll");

      expect(wrapper.get(trackSelector).attributes("style")).toContain(
        "translate3d(-420px",
      );
    },
  );
});
