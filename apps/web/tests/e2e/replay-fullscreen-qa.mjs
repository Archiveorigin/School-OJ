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
  const user = {id:1,name:'QA',role:'admin',permissions:['event:review']};
  if(base.endsWith(':18094')) await page.addInitScript(user=>{
    localStorage.setItem('school-oj-token','qa-response-only');
    localStorage.setItem('school-oj-user',JSON.stringify(user));
  },user);
  await page.route('**/api/v1/**',route=>{
    const path=new URL(route.request().url()).pathname;
    const json=path.includes('/ranking/replay') ? packet : path.endsWith('/me') ? user : path.endsWith('/ranking') ? {contest:packet.contest,problems:packet.problems,rows:packet.final_rows,final_ready:true} : path.endsWith('/contests/99') ? {contest:packet.contest,team:{id:1,name:'隔离响应'},problems:[],can_read_problems:true,lifecycle:'closed'} : [];
    return route.fulfill({json});
  });
  await page.goto(base+(base.endsWith(':18094')?'/contest/99/scoreboard':'/tests/e2e/replay-fullscreen.html'));
  await page.getByRole("button", { name: "赛后滚榜", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "赛后滚榜", exact: true }),
    status = dialog.locator(".replay-tools [role=status]"),
    surface = dialog.locator(".replay-surface");
  await dialog.locator(".penalty-row").first().waitFor();
  assert.equal(await status.innerText(), "0 / 3");
  assert.deepEqual(
    await dialog.locator(".penalty-participant-name").allTextContents(),
    ["选手1", "选手2", "选手3"],
  );
  await dialog.getByRole("button", { name: "全屏显示", exact: true }).click();
  await page.waitForFunction(() =>
    document.fullscreenElement?.matches("[data-replay-surface]"),
  );
  for (const name of ["揭晓下一题", "播放", "从封榜开始"])
    assert(await dialog.getByRole("button", { name, exact: true }).isVisible());
  await page.keyboard.press("ArrowRight");
  await page.waitForTimeout(900);
  assert.equal(await status.innerText(), "1 / 3");
  await page.keyboard.down("ArrowRight");
  await page.waitForTimeout(900);
  await page.keyboard.down("ArrowRight");
  await page.waitForTimeout(900);
  await page.keyboard.up("ArrowRight");
  assert.equal(
    await status.innerText(),
    "2 / 3",
    "holding a key must not advance again",
  );
  await dialog
    .getByRole("button", { name: "揭晓下一题", exact: true })
    .press("Home");
  assert.equal(await status.innerText(), "0 / 3");
  await dialog.locator('input[aria-label="筛选姓名"]').fill("选手2");
  await page.keyboard.press("ArrowRight");
  assert.equal(await status.innerText(), "0 / 3");
  await dialog.getByRole("button", { name: "从封榜开始", exact: true }).click();
  assert.equal(await dialog.locator(".penalty-row").count(), 3);
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
  assert.equal(await status.innerText(), "0 / 3");
  await page.locator(".el-drawer__close-btn").click();
  await page.locator(".el-drawer").waitFor({ state: "hidden" });
  await surface.focus();
  await page.keyboard.press(" ");
  await page.waitForTimeout(50);
  await page.keyboard.press(" ");
  await page.waitForTimeout(1000);
  assert.equal(await status.innerText(), "1 / 3");
  await dialog.getByRole("button", { name: "从封榜开始", exact: true }).click();
  await dialog.locator(".replay-stage").dispatchEvent("click");
  await page.waitForTimeout(900);
  assert.equal(await status.innerText(), "1 / 3");
  for (let i = 0; i < 2; i++) {
    await dialog
      .getByRole("button", { name: "揭晓下一题", exact: true })
      .click();
    await page.waitForTimeout(900);
  }
  assert.equal(await status.innerText(), "3 / 3 · 滚榜结束");
  assert.deepEqual(
    await dialog.locator(".penalty-participant-name").allTextContents(),
    ["选手2", "选手1", "选手3"],
  );
  await mkdir("../../tmp/replay-fullscreen-qa", { recursive: true });
  await page.screenshot({
    path: "../../tmp/replay-fullscreen-qa/fullscreen-final.png",
  });
  await dialog.getByRole("button", { name: "全屏显示", exact: true }).click();
  await page.waitForFunction(() => !document.fullscreenElement);
  await dialog.getByRole("button", { name: "从封榜开始", exact: true }).click();
  assert.equal(await status.innerText(), "0 / 3");
  assert.deepEqual(errors, []);
  console.log(
    "PASS freeze initial order, native fullscreen controls/drawer, focused-button arrows, repeat guard, text editing, blank clicks, pause/restart and final order",
  );
} finally {
  await browser.close();
}
