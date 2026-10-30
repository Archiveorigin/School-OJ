import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
const base = process.env.QA_WEB_BASE || "http://127.0.0.1:4183";
assert(["http://127.0.0.1:4183", "http://127.0.0.1:18094"].includes(base));
const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } }),
  errors = [];
page.on("pageerror", (e) => errors.push(e.message));
const empty = (id) => ({
  problem_id: id,
  status: "",
  attempts: 0,
  submission_count: 0,
  best_score: 0,
  wrong_attempts: 0,
  elapsed_minutes: 0,
});
const ac = (id, time) => ({
  ...empty(id),
  status: "accepted",
  attempts: 1,
  submission_count: 1,
  best_score: 100,
  elapsed_minutes: time,
  solved_at: `2026-10-31T${time < 60 ? "10" : "11"}:${String(time % 60).padStart(2, "0")}:00Z`,
});
const row = (id, rank, cells) => ({
  user_id: id,
  name: `选手${id}`,
  rank,
  solved: cells.filter((c) => c.status === "accepted").length,
  total_score: cells.reduce((s, c) => s + c.best_score, 0),
  penalty_minutes: cells
    .filter((c) => c.status === "accepted")
    .reduce((s, c) => s + c.elapsed_minutes, 0),
  submission_count: cells.reduce((s, c) => s + c.submission_count, 0),
  problems: cells,
});
const initial = [
  row(1, 1, [ac(1, 10), empty(2)]),
  row(2, 2, [empty(1), empty(2)]),
  row(3, 2, [empty(1), empty(2)]),
];
const packet = {
  schema_version: 1,
  kind: "contest-replay",
  contest: {
    id: 99,
    title: "封榜与全屏事件回归",
    scoring_rule: "acm",
    duration_minutes: 120,
    starts_at: "2026-10-31T10:00:00Z",
  },
  freeze_at: "2026-10-31T11:30:00Z",
  problems: [
    { problem_id: 1, label: "A", score: 100 },
    { problem_id: 2, label: "B", score: 100 },
  ],
  initial_rows: initial,
  final_rows: [
    row(2, 1, [ac(1, 95), ac(2, 100)]),
    initial[0],
    row(3, 3, [
      { ...empty(1), status: "wrong_answer", attempts: 1, submission_count: 1 },
      empty(2),
    ]),
  ],
  events: [],
};
packet.final_rows[1] = { ...packet.final_rows[1], rank: 2 };
try {
  const user = {
    id: 1,
    name: "QA",
    role: "admin",
    permissions: ["event:review"],
  };
  if (base.endsWith(":18094"))
    await page.addInitScript((user) => {
      localStorage.setItem("school-oj-token", "qa-response-only");
      localStorage.setItem("school-oj-user", JSON.stringify(user));
    }, user);
  await page.route("**/api/v1/**", (route) => {
    const path = new URL(route.request().url()).pathname;
    const json = path.includes("/ranking/replay")
      ? packet
      : path.endsWith("/me")
        ? user
        : path.endsWith("/ranking")
          ? {
              contest: packet.contest,
              problems: packet.problems,
              rows: packet.final_rows,
              final_ready: true,
            }
          : path.endsWith("/contests/99")
            ? {
                contest: packet.contest,
                team: { id: 1, name: "隔离响应" },
                problems: [],
                can_read_problems: true,
                lifecycle: "closed",
              }
            : [];
    return route.fulfill({ json });
  });
  await page.goto(
    base +
      (base.endsWith(":18094")
        ? "/contest/99/scoreboard"
        : "/tests/e2e/replay-fullscreen.html"),
  );
  await page.getByRole("button", { name: "赛后滚榜", exact: true }).click();

  const dialog = page.getByRole("dialog", { name: "赛后滚榜", exact: true }),
    surface = dialog.locator(".replay-surface");
  const ids = () =>
    dialog
      .locator(".penalty-row")
      .evaluateAll((rows) => rows.map((row) => Number(row.dataset.rowId)));
  await dialog.locator(".penalty-row").first().waitFor();
  assert.deepEqual(await ids(), [1, 2, 3]);
  assert.equal(
    await dialog
      .locator(
        ".replay-tools,progress,.replay-announcement,.controls-time-section",
      )
      .count(),
    0,
  );
  assert.equal(
    await dialog.getByRole("button", { name: "重新加载", exact: true }).count(),
    1,
  );
  await surface.focus();
  await page.keyboard.press("f");
  await page.waitForFunction(() =>
    document.fullscreenElement?.matches("[data-replay-surface]"),
  );
  assert(
    await dialog
      .getByRole("button", { name: "重新加载", exact: true })
      .isVisible(),
  );
  await page.keyboard.press("ArrowRight");
  await page.waitForTimeout(900);
  await page.keyboard.down("ArrowRight");
  await page.waitForTimeout(900);
  await page.keyboard.down("ArrowRight");
  await page.waitForTimeout(900);
  await page.keyboard.up("ArrowRight");
  // The second acceptance is still hidden: repeat keydown must not skip it.
  assert.deepEqual(await ids(), [1, 2, 3]);
  assert.equal(
    await dialog.locator('[data-row-id="2"] .result-accepted').count(),
    1,
  );
  const before = await page.evaluate(() => {
    const first = document.querySelector(
        '[data-replay-surface] [data-row-id="1"]',
      ),
      second = document.querySelector(
        '[data-replay-surface] [data-row-id="2"]',
      );
    window.qaMovingRow = second;
    return {
      first: first.getBoundingClientRect().top,
      second: second.getBoundingClientRect().top,
    };
  });
  await page.keyboard.press("Enter");
  await page.waitForTimeout(100);
  const during = await page.evaluate(() => {
    const first = document.querySelector(
        '[data-replay-surface] [data-row-id="1"]',
      ),
      second = document.querySelector(
        '[data-replay-surface] [data-row-id="2"]',
      );
    return {
      same: second === window.qaMovingRow,
      first: first.getBoundingClientRect().top,
      second: second.getBoundingClientRect().top,
      transform: getComputedStyle(second).transform,
      transition: getComputedStyle(second).transitionProperty,
    };
  });
  assert(
    during.same,
    "the moving contestant must retain the same article element",
  );
  assert.notEqual(during.transform, "none", "whole row must be moving");
  assert(during.transition.includes("transform"));
  assert(
    during.second < before.second && during.second > before.first,
    "AC row moves upward between its old and new slots",
  );
  assert(
    during.first > before.first && during.first < before.second,
    "displaced row moves downward",
  );
  await mkdir("../../tmp/replay-row-motion-qa", { recursive: true });
  await page.screenshot({ path: "../../tmp/replay-row-motion-qa/moving.png" });
  await page.waitForTimeout(900);
  assert.deepEqual(await ids(), [2, 1, 3]);
  await page.screenshot({ path: "../../tmp/replay-row-motion-qa/final.png" });
  await page.keyboard.press("Home");
  await page.waitForTimeout(50);
  assert.deepEqual(await ids(), [1, 2, 3]);
  await dialog
    .locator(".penalty-row")
    .first()
    .locator(".penalty-result")
    .first()
    .click();
  await page.locator(".el-drawer").waitFor();
  assert(
    await page
      .locator(".el-drawer")
      .evaluate((el) => document.fullscreenElement.contains(el)),
  );
  await page.locator(".el-drawer__close-btn").click();
  await page.locator(".el-drawer").waitFor({ state: "hidden" });
  await dialog.locator(".replay-stage").dispatchEvent("click");
  await page.waitForTimeout(900);
  assert.equal(
    await dialog.locator('[data-row-id="3"] .result-wrong').count(),
    0,
    "mouse no longer advances replay",
  );
  await surface.focus();
  await page.keyboard.press(" ");
  await page.waitForTimeout(900);
  assert.equal(
    await dialog.locator('[data-row-id="3"] .result-wrong').count(),
    1,
  );
  await dialog.getByRole("button", { name: "重新加载", exact: true }).click();
  await page.waitForTimeout(100);
  assert.deepEqual(await ids(), [1, 2, 3]);
  assert.equal(
    await dialog.locator('[data-row-id="3"] .result-wrong').count(),
    0,
  );
  await surface.focus();
  await page.keyboard.press("f");
  await page.waitForFunction(() => !document.fullscreenElement);
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.waitForTimeout(100);
  for (let i = 0; i < 3; i++) {
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(250);
  }
  assert.deepEqual(await ids(), [2, 1, 3]);
  assert.deepEqual(errors, []);

  // The score-table variant must also move an entire row, not just its labels.
  packet.contest.scoring_rule = "ioi";
  await dialog.getByRole("button", { name: "重新加载", exact: true }).click();
  await dialog.locator(".score-row").first().waitFor();
  await page.emulateMedia({ reducedMotion: "no-preference" });
  await page.waitForTimeout(100);
  await surface.focus();
  for (let i = 0; i < 2; i++) {
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(900);
  }
  await page.keyboard.press("ArrowRight");
  await page.waitForTimeout(80);
  assert.notEqual(
    await dialog
      .locator('.score-row[data-row-id="2"]')
      .evaluate((el) => getComputedStyle(el).transform),
    "none",
  );
  await page.waitForTimeout(900);
  assert.deepEqual(
    await dialog
      .locator(".score-row")
      .evaluateAll((rows) => rows.map((r) => Number(r.dataset.rowId))),
    [2, 1, 3],
  );
  // Bounded 500-person / 60-step manual replay and a zero-step 1000-person board.
  packet.contest.scoring_rule = "acm";
  packet.contest.duration_minutes = 240;
  packet.initial_rows = Array.from({ length: 500 }, (_, i) =>
    row(10000 + i, 1, [empty(1), empty(2)]),
  );
  packet.final_rows = packet.initial_rows.map((r, i) =>
    i >= 440
      ? row(r.user_id, 1, [ac(1, 95 + i - 440), empty(2)])
      : structuredClone(r),
  );
  packet.final_rows.sort(
    (a, b) =>
      b.solved - a.solved ||
      a.penalty_minutes - b.penalty_minutes ||
      a.user_id - b.user_id,
  );
  packet.final_rows.forEach((r, i) => (r.rank = r.solved ? i + 1 : 61));
  await page.emulateMedia({ reducedMotion: "reduce" });
  await dialog.getByRole("button", { name: "重新加载", exact: true }).click();
  await dialog.locator(".penalty-row").first().waitFor();
  assert.equal(await dialog.locator(".penalty-row").count(), 100);
  await surface.focus();
  for (let i = 0; i < 60; i++) {
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(230);
  }
  assert.equal(await dialog.locator(".penalty-row").count(), 100);
  assert.equal(
    await dialog.locator(".penalty-row").first().getAttribute("data-row-id"),
    String(packet.final_rows[0].user_id),
  );
  packet.initial_rows = Array.from({ length: 1000 }, (_, i) =>
    row(20000 + i, 1, [empty(1), empty(2)]),
  );
  packet.final_rows = structuredClone(packet.initial_rows);
  await dialog.getByRole("button", { name: "重新加载", exact: true }).click();
  await page.waitForTimeout(100);
  assert.equal(await dialog.locator(".penalty-row").count(), 100);
  await surface.focus();
  await page.keyboard.press("ArrowRight");
  assert.equal(await dialog.locator(".result-accepted").count(), 0);
  assert.deepEqual(errors, []);
  console.log(
    "PASS clean keyboard-only UI, native fullscreen, physical-key guard, whole-row identity and measured upward/downward intermediate geometry, reload/restart, drawer isolation and reduced motion",
  );
  console.log(JSON.stringify({ before, during }));
} finally {
  await browser.close();
}
