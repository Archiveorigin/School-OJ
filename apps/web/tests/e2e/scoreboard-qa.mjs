import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
const fixture = "http://127.0.0.1:4179",
  base = process.env.QA_WEB_BASE || fixture;
assert([fixture, "http://127.0.0.1:18092"].includes(base), "loopback QA only");
const output = "../../tmp/icpc-scoreboard-qa";
await mkdir(output, { recursive: true });
const browser = await chromium.launch();
const failures = [];
const metrics = {};
async function context() {
  const c = await browser.newContext({
    viewport: { width: 1440, height: 1100 },
  });
  if (base !== fixture)
    await c.route("**/api/**", async (route) => {
      const u = new URL(route.request().url());
      await route.fulfill({
        response: await route.fetch({
          url: fixture + u.pathname + u.search,
          headers: { ...route.request().headers(), origin: fixture },
        }),
      });
    });
  return c;
}
async function login(c, email) {
  const p = await c.newPage();
  p.on("pageerror", (e) => failures.push(e.message));
  await p.goto(base + "/login");
  await p.getByPlaceholder("邮箱", { exact: true }).fill(email);
  await p.getByPlaceholder("密码", { exact: true }).fill("ExamFixture-2026!");
  await p.getByRole("button", { name: "登录", exact: true }).click();
  await p.waitForURL((u) => u.pathname != "/login");
  return p;
}
async function closeDrawer(p) {
  await p.locator(".el-drawer__close-btn").last().click();
  await p.locator(".el-drawer").waitFor({ state: "hidden" });
}
try {
  const studentContext = await context(),
    student = await login(studentContext, "exam-student-0@example.test");
  const info = await (
      await studentContext.request.get(fixture + "/__qa/info")
    ).json(),
    contest = info.rankings.find(
      (x) => x.kind === "contest" && x.rule === "acm",
    );
  await student.goto(`${base}/contest/${contest.id}/scoreboard`);
  await student.locator(".penalty-row").first().waitFor();
  assert.equal(
    await student
      .getByRole("button", { name: "赛后滚榜", exact: true })
      .count(),
    0,
  );
  await student.locator(".row-summary-toggle").first().click();
  await student.locator(".row-detail-inner").waitFor();
  assert.equal(await student.locator("[data-expanded=true]").count(), 1);
  const ownCell = student
    .locator(".penalty-row")
    .filter({ hasText: "考生甲" })
    .locator(".penalty-result")
    .first();
  await ownCell.click();
  await student.locator(".cell-timeline li").first().waitFor();
  assert(await student.locator(".cell-timeline").innerText());
  await closeDrawer(student);
  await ownCell.focus();
  await ownCell.press("Enter");
  await student.locator(".cell-timeline li").first().waitFor();
  await closeDrawer(student);
  await student
    .locator(".penalty-row")
    .filter({ hasText: "考生乙" })
    .locator(".penalty-result")
    .first()
    .click();
  await student
    .getByText("仅显示榜单汇总；提交明细只向本人或具有查看权限的管理员开放。", {
      exact: true,
    })
    .waitFor();
  assert.equal(await student.locator(".cell-timeline li").count(), 0);
  await closeDrawer(student);
  for (const width of [1440, 390]) {
    await student.setViewportSize({
      width,
      height: width === 390 ? 844 : 1100,
    });
    await student.waitForTimeout(250);
    for (const dark of [false, true]) {
      await student.evaluate(
        (d) => document.documentElement.classList.toggle("dark", d),
        dark,
      );
      if (
        !(await student.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ))
      ) {
        await student.screenshot({
          path: output + "/overflow.png",
          fullPage: true,
        });
        console.log(
          await student.evaluate(() =>
            [...document.querySelectorAll("body *")]
              .map((el) => ({
                tag: el.tagName,
                cls: el.className,
                x: el.getBoundingClientRect().x,
                w: el.getBoundingClientRect().width,
              }))
              .filter((r) => r.x < 0 || r.x + r.w > innerWidth + 1)
              .slice(0, 18),
          ),
        );
        throw new Error("page overflow at " + width);
      }
      await student.screenshot({
        path: `${output}/student-${width}-${dark ? "dark" : "light"}.png`,
        fullPage: true,
      });
    }
  }
  await student.setViewportSize({ width: 1440, height: 1100 });
  await student.getByRole("button", { name: "全屏显示", exact: true }).click();
  await student.waitForFunction(() => Boolean(document.fullscreenElement));
  await ownCell.click();
  await student.locator(".cell-timeline li").first().waitFor();
  assert(
    await student
      .locator(".el-drawer")
      .evaluate((el) => document.fullscreenElement.contains(el)),
  );
  await closeDrawer(student);
  await student.evaluate(() => document.exitFullscreen());
  await student.emulateMedia({ reducedMotion: "reduce" });
  await student.waitForFunction(
    () =>
      document
        .querySelector(".scoreboard-core")
        ?.style.getPropertyValue("--score-motion-row") === "0ms",
  );
  assert.equal(
    await student
      .locator(".scoreboard-core")
      .evaluate((el) => el.style.getPropertyValue("--score-motion-row")),
    "0ms",
  );
  await studentContext.unrouteAll({ behavior: "wait" });
  await studentContext.close();
  const adminContext = await context(),
    admin = await login(adminContext, "exam-admin-3@example.test");
  await adminContext.request.post(
    `${fixture}/__qa/rankings/contest/${contest.id}/resolve`,
  );
  await admin.goto(`${base}/contest/${contest.id}/scoreboard`);
  const replayResponse = admin.waitForResponse(
    (r) => r.url().includes("/ranking/replay") && r.status() === 200,
  );
  await admin.getByRole("button", { name: "赛后滚榜", exact: true }).click();
  const replay = admin.getByRole("dialog", { name: "赛后滚榜", exact: true });
  await replay.locator(".penalty-row").first().waitFor();
  const firstLeader = await replay
    .locator(".penalty-participant-name")
    .first()
    .innerText();
  const replayPacket = await (await replayResponse).json();
  const countSteps = replayPacket.initial_rows.reduce(
    (sum, r) =>
      sum +
      r.problems.filter(
        (c) =>
          JSON.stringify(c) !==
          JSON.stringify(
            replayPacket.final_rows
              .find((f) => f.user_id === r.user_id)
              .problems.find((f) => f.problem_id === c.problem_id),
          ),
      ).length,
    0,
  );
  await replay.locator(".replay-surface").focus();
  for (let i = 0; i < countSteps; i++) {
    await admin.keyboard.press("ArrowRight");
    await admin.waitForTimeout(900);
  }
  assert.notEqual(
    await replay.locator(".penalty-participant-name").first().innerText(),
    firstLeader,
  );
  for (const width of [1440, 390]) {
    await admin.setViewportSize({ width, height: width === 390 ? 844 : 1100 });
    await admin.screenshot({
      path: `${output}/replay-${width}.png`,
      fullPage: false,
    });
  }
  await replay.locator(".replay-surface").focus();
  await admin.keyboard.press("Home");
  await admin.keyboard.press("Escape");
  await replay.waitFor({ state: "hidden" });
  // Synthetic response-only performance fixture; no account, score or submission is written.
  const makePacket = (count, steps) => {
    const problems = Array.from({ length: 4 }, (_, i) => ({
      problem_id: i + 1,
      label: String.fromCharCode(65 + i),
      score: 100,
    }));
    const initial = Array.from({ length: count }, (_, i) => ({
      user_id: 10000 + i,
      name: `性能测试${String(i).padStart(4, "0")}`,
      rank: 1,
      solved: 0,
      total_score: 0,
      penalty_minutes: 0,
      submission_count: 0,
      problems: problems.map((p) => ({
        problem_id: p.problem_id,
        status: "",
        attempts: 0,
        wrong_attempts: 0,
        best_score: 0,
        max_score: 100,
        submission_count: 0,
      })),
    }));
    const final = structuredClone(initial);
    for (let i = 0; i < steps; i++) {
      const r = final[count - 1 - i],
        time = steps - i + 30;
      r.solved = 1;
      r.total_score = 100;
      r.penalty_minutes = time;
      r.submission_count = 1;
      Object.assign(r.problems[0], {
        status: "accepted",
        attempts: 1,
        best_score: 100,
        elapsed_minutes: time,
        submission_count: 1,
        solved_at: new Date(Date.UTC(2026, 9, 30, 10, time)).toISOString(),
      });
    }
    final.sort(
      (a, b) =>
        b.solved - a.solved ||
        a.penalty_minutes - b.penalty_minutes ||
        a.user_id - b.user_id,
    );
    for (let i = 0; i < final.length; i++)
      final[i].rank = final[i].solved ? i + 1 : steps + 1;
    return {
      schema_version: 1,
      kind: "contest-replay",
      contest: {
        id: contest.id,
        title: "500 人 · 60 步性能验收",
        scoring_rule: "acm",
        duration_minutes: 120,
        starts_at: "2026-10-30T10:00:00Z",
      },
      problems,
      initial_rows: initial,
      final_rows: final,
      events: [],
      freeze_at: "2026-10-30T10:30:00Z",
    };
  };
  let packet = makePacket(500, 60);
  await adminContext.route(
    "**/event-papers/contest/*/ranking/replay",
    (route) => route.fulfill({ json: packet }),
  );
  await admin.setViewportSize({ width: 1440, height: 1100 });
  await admin.getByRole("button", { name: "赛后滚榜", exact: true }).click();
  await replay.locator(".penalty-row").first().waitFor();
  assert.equal(await replay.locator(".penalty-row").count(), 100);
  await admin.evaluate(() => {
    window.qaLongTasks = [];
    window.qaPerfObserver = new PerformanceObserver((list) =>
      window.qaLongTasks.push(...list.getEntries().map((e) => e.duration)),
    );
    window.qaPerfObserver.observe({ entryTypes: ["longtask"] });
  });
  await admin.emulateMedia({ reducedMotion: "reduce" });
  await replay.locator(".replay-surface").focus();
  for (let i = 0; i < 60; i++) {
    await admin.keyboard.press("ArrowRight");
    await admin.waitForTimeout(230);
  }
  metrics.performance = await admin.evaluate(() => {
    window.qaPerfObserver.disconnect();
    return {
      longTasks: window.qaLongTasks,
      maxLongTask: Math.max(0, ...window.qaLongTasks),
      hardwareConcurrency: navigator.hardwareConcurrency,
      userAgent: navigator.userAgent,
    };
  });
  metrics.performance.rows = 500;
  metrics.performance.steps = 60;
  metrics.performance.domRows = await replay.locator(".penalty-row").count();
  await replay.locator(".replay-surface").focus();
  await admin.keyboard.press("Escape");
  await replay.waitFor({ state: "hidden" });
  packet = makePacket(1000, 0);
  await admin.getByRole("button", { name: "赛后滚榜", exact: true }).click();
  await replay.locator(".penalty-row").first().waitFor();
  assert.equal(await replay.locator(".penalty-row").count(), 100);
  await replay.locator(".replay-surface").focus();
  await admin.keyboard.press("Escape");
  await adminContext.unrouteAll({ behavior: "wait" });
  await adminContext.close();
  assert.deepEqual(failures, []);
  await writeFile(`${output}/metrics.json`, JSON.stringify(metrics, null, 2));
  console.log(
    "PASS rows, cell timeline/permissions, motion/reduced motion, keyboard-only replay, keyboard, zero-step, 500x60 and 1000-row bounds",
  );
  console.log(JSON.stringify(metrics));
} finally {
  await browser.close();
}
