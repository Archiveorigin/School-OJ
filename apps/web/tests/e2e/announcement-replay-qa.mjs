import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
const fixture = "http://127.0.0.1:4179",
  base = process.env.QA_WEB_BASE || fixture;
assert(
  ["http://127.0.0.1:4179", "http://127.0.0.1:18092"].includes(base),
  "QA only permits explicit loopback endpoints",
);
async function useFixture(context) {
  if (base === fixture) return;
  await context.route("**/api/**", async (route) => {
    const url = new URL(route.request().url());
    const response = await route.fetch({
      url: fixture + url.pathname + url.search,
      headers: { ...route.request().headers(), origin: fixture },
    });
    await route.fulfill({ response });
  });
}
await mkdir("../../tmp/replay-qa", { recursive: true });
const browser = await chromium.launch();
try {
  const context = await browser.newContext({
    viewport: { width: 1440, height: 1100 },
    acceptDownloads: true,
  });
  await useFixture(context);
  const p = await context.newPage();
  let announcementReads = 0;
  p.on("request", (r) => {
    if (r.method() === "GET" && r.url().endsWith("/announcements"))
      announcementReads++;
  });
  const errors = [];
  p.on("pageerror", (e) => errors.push(e.message));
  const info = await (await context.request.get(fixture + "/__qa/info")).json();
  const item = info.rankings.find(
    (x) => x.kind === "contest" && x.rule === "acm",
  );
  await p.goto(base + "/login");
  await p
    .getByPlaceholder("邮箱", { exact: true })
    .fill("exam-teacher-2@example.test");
  await p.getByPlaceholder("密码", { exact: true }).fill("ExamFixture-2026!");
  await p.getByRole("button", { name: "登录", exact: true }).click();
  await p.waitForURL((u) => u.pathname != "/login");
  await context.request.post(`${fixture}/__qa/clock/contest/${item.id}/start`);
  await p.goto(`${base}/contest/${item.id}/description`);
  const board = p.getByRole("region", { name: "赛时公告" });
  await board.waitFor();
  const boardBox = await board.boundingBox(),
    intro = await p.locator(".two-column").boundingBox();
  assert(
    boardBox.y >= intro.y + intro.height,
    "announcement board must be below both intro columns",
  );
  assert(
    Math.abs(boardBox.width - intro.width) < 2,
    "announcement board spans both columns",
  );
  await p.getByRole("button", { name: "发布公告", exact: true }).click();
  const dialog = p.getByRole("dialog", { name: "发布赛时公告" });
  assert.equal(
    await board.getByRole("dialog").count(),
    0,
    "publisher is separate from announcement board",
  );
  await dialog
    .getByLabel("公告正文")
    .fill("**输入范围更正**：保证 $x \\le 100$。\n\n请查看解释文档。");
  await dialog.getByRole("button", { name: "预览正文", exact: true }).click();
  await dialog.locator(".announcement-preview strong").waitFor();
  await dialog.locator(".announcement-preview .katex").first().waitFor();
  await p.setViewportSize({ width: 390, height: 844 });
  assert(
    await p.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  await p.screenshot({
    path: "../../tmp/replay-qa/announcement-publisher-mobile.png",
    fullPage: true,
  });
  await p.setViewportSize({ width: 1440, height: 1100 });
  await dialog
    .locator("input[type=file]")
    .setInputFiles({
      name: "解释.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("边界条件解释"),
    });
  await dialog.getByRole("button", { name: "发布公告", exact: true }).click();
  await dialog.waitFor({ state: "hidden" });
  await board
    .getByRole("button", { name: "解释.txt", exact: true })
    .first()
    .waitFor();
  await board
    .locator("strong")
    .filter({ hasText: "输入范围更正" })
    .first()
    .waitFor();
  await board.locator(".katex").first().waitFor();
  const reads = announcementReads;
  await p.waitForTimeout(16000);
  assert.equal(announcementReads, reads, "announcement must not poll");
  for (const width of [1440, 390]) {
    await p.setViewportSize({ width, height: width === 390 ? 844 : 1100 });
    assert(
      await p.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth + 1,
      ),
    );
    for (const dark of [false, true]) {
      await p.evaluate((dark) => {
        document.documentElement.dataset.theme = dark ? "dark" : "light";
        document.documentElement.classList.toggle("dark", dark);
      }, dark);
      await p.screenshot({
        path: `../../tmp/replay-qa/announcement-${width}-${dark ? "dark" : "light"}.png`,
        fullPage: true,
      });
    }
  }
  const studentContext = await browser.newContext({
    viewport: { width: 1440, height: 1100 },
  });
  await useFixture(studentContext);
  const student = await studentContext.newPage();
  student.on("pageerror", (e) => errors.push(e.message));
  await student.goto(base + "/login");
  await student
    .getByPlaceholder("邮箱", { exact: true })
    .fill("exam-student-0@example.test");
  await student
    .getByPlaceholder("密码", { exact: true })
    .fill("ExamFixture-2026!");
  await student.getByRole("button", { name: "登录", exact: true }).click();
  await student.waitForURL((u) => u.pathname != "/login");
  await student.goto(`${base}/contest/${item.id}/description`);
  const studentBoard = student.getByRole("region", { name: "赛时公告" });
  await studentBoard
    .locator("strong")
    .filter({ hasText: "输入范围更正" })
    .first()
    .waitFor();
  assert.equal(
    await student
      .getByRole("button", { name: "发布公告", exact: true })
      .count(),
    0,
  );
  await studentBoard.getByRole("button", { name: "提问", exact: true }).click();
  await student
    .getByRole("dialog", { name: "工单修改", exact: true })
    .waitFor();
  if (base === fixture) {
    const submissionCase = info.rankings.find(
      (x) => x.kind === "contest" && x.rule === "ioi",
    );
    await context.request.post(
      `${fixture}/__qa/clock/contest/${submissionCase.id}/start`,
    );
    await student.goto(`${base}/contest/${submissionCase.id}/problems/A`);
    await student
      .getByRole("button", { name: "提交代码", exact: true })
      .click();
    const composer = student.getByRole("dialog", { name: /^提交 A/ });
    await composer.locator(".monaco-editor").waitFor();
    await composer.locator(".monaco-editor textarea").first().focus();
    await student.keyboard.press("ControlOrMeta+A");
    await student.keyboard.insertText("int main(){return 0;}");
    await student.waitForTimeout(100);
    let requests = 0;
    student.on("request", (r) => {
      if (r.method() === "POST" && r.url().endsWith("/submissions")) requests++;
    });
    const accepted = student.waitForResponse(
      (r) =>
        r.request().method() === "POST" && r.url().endsWith("/submissions"),
    );
    await composer
      .getByRole("button", { name: "提交评测", exact: true })
      .evaluate((button) => {
        button.click();
        button.click();
        button.click();
      });
    assert.equal((await accepted).status(), 201);
    await composer.waitFor({ state: "hidden" });
    assert.equal(requests, 1, "rapid clicks must issue a single submission");
  }
  await studentContext.unrouteAll({ behavior: "wait" });
  await studentContext.close();
  const examCase = info.rankings.find(
    (x) => x.kind === "exam" && x.rule === "acm",
  );
  await p.goto(`${base}/exams/${examCase.id}/description`);
  const examBoard = p.getByRole("region", { name: "赛时公告" });
  await examBoard.waitFor();
  const examBox = await examBoard.boundingBox(),
    examInfo = await p.locator(".exam-info").boundingBox(),
    examDescription = await p.locator(".description-panel").boundingBox();
  assert(
    examBox.y >=
      Math.max(
        examInfo.y + examInfo.height,
        examDescription.y + examDescription.height,
      ),
    "exam board below both columns",
  );
  await p.getByRole("button", { name: "发布公告", exact: true }).click();
  const examPublisher = p.getByRole("dialog", { name: "发布赛时公告" });
  await examPublisher.getByLabel("公告正文").fill("考试 **公告** 排版验证。");
  await examPublisher
    .getByRole("button", { name: "发布公告", exact: true })
    .click();
  await examPublisher.waitFor({ state: "hidden" });
  await examBoard
    .locator("strong")
    .filter({ hasText: "公告" })
    .first()
    .waitFor();
  await context.request.post(`${fixture}/__qa/clock/contest/${item.id}/end`);
  await context.request.post(
    `${fixture}/__qa/rankings/contest/${item.id}/resolve`,
  );

  await p.goto(`${base}/contest/${item.id}/scoreboard`);
  assert.equal(
    await p.getByRole("button", { name: "赛后滚榜", exact: true }).count(),
    0,
    "teacher cannot roll standings",
  );
  const adminContext = await browser.newContext({
    viewport: { width: 1440, height: 1100 },
  });
  await useFixture(adminContext);
  const admin = await adminContext.newPage();
  admin.on("pageerror", (e) => errors.push(e.message));
  await admin.goto(base + "/login");
  await admin
    .getByPlaceholder("邮箱", { exact: true })
    .fill("exam-admin-3@example.test");
  await admin
    .getByPlaceholder("密码", { exact: true })
    .fill("ExamFixture-2026!");
  await admin.getByRole("button", { name: "登录", exact: true }).click();
  await admin.waitForURL((u) => u.pathname != "/login");
  await admin.goto(`${base}/contest/${item.id}/scoreboard`);
  await admin.getByRole("button", { name: "赛后滚榜", exact: true }).click();
  const replay = admin.getByRole("dialog", { name: "赛后滚榜", exact: true });
  await replay.locator(".scoreboard-core").waitFor();
  const initialLeader = await replay
    .locator(".penalty-participant-name")
    .first()
    .innerText();
  assert.equal(
    await replay.getByText("导入离线数据包", { exact: true }).count(),
    0,
  );
  assert.equal(
    await replay.getByText("导出滚榜数据包", { exact: true }).count(),
    0,
  );
  await replay.getByRole("button", { name: "揭晓下一题", exact: true }).click();
  await admin.waitForTimeout(1000);
  while (
    await replay
      .getByRole("button", { name: "揭晓下一题", exact: true })
      .isEnabled()
  ) {
    await replay
      .getByRole("button", { name: "揭晓下一题", exact: true })
      .click();
    await admin.waitForTimeout(1000);
  }
  await replay.getByRole("status").filter({ hasText: "滚榜结束" }).waitFor();
  assert.notEqual(
    await replay.locator(".penalty-participant-name").first().innerText(),
    initialLeader,
    "actual ranking order must change",
  );
  await admin.screenshot({
    path: "../../tmp/replay-qa/admin-direct-replay.png",
    fullPage: true,
  });
  await admin.setViewportSize({ width: 390, height: 844 });
  assert(
    await admin.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  await admin.screenshot({
    path: "../../tmp/replay-qa/admin-direct-replay-mobile.png",
    fullPage: false,
  });
  await adminContext.unrouteAll({ behavior: "wait" });
  await adminContext.close();
  await context.unrouteAll({ behavior: "wait" });
  assert.deepEqual(errors, []);
  console.log(
    "PASS separated announcements, Markdown/math, question tickets, admin-only direct click replay, no import/export UI, responsive layouts",
  );
} finally {
  await browser.close();
}
