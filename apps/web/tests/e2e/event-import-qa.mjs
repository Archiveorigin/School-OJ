// Local browser acceptance only. The two user fixtures are supplied by env.
import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
const root = "http://127.0.0.1:4179",
  api = "http://127.0.0.1:18089",
  output = path.resolve("../../tmp/event-import-browser");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true }),
  errors = [],
  checks = [];
const markdownFile = process.env.OJ_IMPORT_MARKDOWN_FIXTURE,
  pdfFile = process.env.OJ_IMPORT_PDF_FIXTURE;
assert.ok(
  markdownFile && pdfFile,
  "Set both OJ_IMPORT_*_FIXTURE environment variables",
);
async function call(page, method, url, body) {
  const token = await page.evaluate(() =>
    localStorage.getItem("school-oj-token"),
  );
  return page.request.fetch(api + "/api/v1" + url, {
    method,
    headers: { Authorization: `Bearer ${token}` },
    data: body,
  });
}
async function shot(page, name) {
  await page.screenshot({
    path: path.join(output, name + ".png"),
    fullPage: !(await page.getByRole("dialog").count()),
  });
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 2,
    ),
    name + ": overflow",
  );
}
const test = (name, text) => ({
  name,
  mimeType: "text/plain",
  buffer: Buffer.from(text),
});
try {
  for (const [size, viewport] of [
    ["desktop", { width: 1440, height: 1000 }],
    ["mobile", { width: 390, height: 844 }],
  ]) {
    const page = await browser.newPage({ viewport });
    page.setDefaultTimeout(30000);
    page.on("pageerror", (e) => errors.push(e.message));
    page.on("response", (r) => {
      if (r.status() >= 500) errors.push(r.status() + " " + r.url());
    });
    await page.goto(root + "/login");
    await page
      .getByPlaceholder("邮箱", { exact: true })
      .fill("exam-teacher-2@example.test");
    await page
      .getByPlaceholder("密码", { exact: true })
      .fill("ExamFixture-2026!");
    await page.getByRole("button", { name: "登录", exact: true }).click();
    await page.waitForURL((u) => u.pathname != "/login");
    const created = await call(page, "POST", "/teams/1/contests", {
      title: "题面导入验收 " + size,
      description: "导入题面、图片与样例",
      starts_at: new Date(Date.now() + 3600000).toISOString(),
      duration_minutes: 60,
      scoring_rule: "ioi",
    });
    assert.equal(created.status(), 201, await created.text());
    const id = (await created.json()).id,
      base = `/event-papers/contest/${id}`;
    await page.goto(root + `/contest/${id}/description`);
    const compose = await page
        .getByRole("button", { name: "组题", exact: true })
        .boundingBox(),
      settings = await page
        .getByRole("button", { name: "比赛设置", exact: true })
        .boundingBox();
    assert.ok(
      compose.x < settings.x && Math.abs(compose.y - settings.y) < 2,
      "compose button before settings in same row",
    );
    assert.equal(await page.locator(".event-manage-card").count(), 0);
    await shot(page, size + "-contest-actions");
    await page.getByRole("button", { name: "组题", exact: true }).click();
    assert.equal(await page.getByText(/修订 \d/).count(), 0);
    await page
      .getByRole("button", { name: "导入 Markdown 单题", exact: true })
      .click();
    let dialog = page.getByRole("dialog", {
      name: "导入 Markdown 单题",
      exact: true,
    });
    await dialog
      .getByLabel("选择题面文件", { exact: true })
      .setInputFiles(markdownFile);
    await dialog
      .getByRole("heading", {
        name: "B4264 [GESP202503 四级] 二阶矩阵",
        exact: true,
      })
      .waitFor();
    const imageResponse = page.waitForResponse(
      (r) =>
        r.request().method() === "POST" && r.url().endsWith("/import-image"),
    );
    await dialog
      .getByRole("button", { name: "导入外链图片", exact: true })
      .click();
    const importedImage = await imageResponse;
    assert.equal(importedImage.status(), 200, await importedImage.text());
    await dialog.getByText("已打包", { exact: true }).waitFor();
    await page.waitForFunction(() =>
      [...document.querySelectorAll(".parsed-preview img")].some(
        (i) => i.complete && i.naturalWidth > 1,
      ),
    );
    await shot(page, size + "-markdown-resources");
    await dialog
      .getByRole("button", { name: "校对并添加测试点", exact: true })
      .click();
    const editor = page.locator(".problem-editor");
    assert.equal(
      await editor.getByLabel("样例 1 输入", { exact: true }).inputValue(),
      "3 4\n1 2 1 0\n2 4 2 1\n0 3 3 0",
    );
    assert.ok(
      (
        await editor.getByLabel("说明（可选）", { exact: true }).inputValue()
      ).includes("assets/"),
    );
    await editor
      .locator("input[type=file]")
      .last()
      .setInputFiles([
        test("sample.in", "3 4\n1 2 1 0\n2 4 2 1\n0 3 3 0\n"),
        test("sample.out", "2\n"),
      ]);
    await editor
      .locator(".el-checkbox")
      .filter({ hasText: "已对照原文核对题面、公式、图片和样例" })
      .click();
    assert.ok(
      await editor
        .getByRole("checkbox", { name: "已对照原文核对题面、公式、图片和样例" })
        .isChecked(),
    );
    let pending = page.waitForResponse(
      (r) =>
        r.request().method() === "POST" && r.url().endsWith(base + "/problems"),
    );
    await editor.getByRole("button", { name: "保存题目", exact: true }).click();
    let result = await pending;
    assert.equal(result.status(), 200, await result.text());
    await editor.waitFor({ state: "hidden" });
    let paper = await (await call(page, "GET", base)).json();
    assert.equal(paper.event.submission_version, 0);
    assert.equal(paper.problems.length, 1);
    await page
      .getByRole("button", { name: "导入 PDF 题册", exact: true })
      .click();
    // The previous Markdown queue is already saved, so switching is immediate.
    dialog = page.getByRole("dialog", { name: "导入 PDF 题册", exact: true });
    await dialog
      .getByLabel("选择题面文件", { exact: true })
      .setInputFiles(pdfFile);
    await dialog
      .locator(".import-list button")
      .nth(9)
      .waitFor({ timeout: 60000 });
    assert.equal(await dialog.locator(".import-list button").count(), 10);
    await dialog.locator(".import-list button").nth(1).click();
    await dialog.getByText("已打包", { exact: true }).waitFor();
    await page.waitForFunction(() =>
      [...document.querySelectorAll(".parsed-preview img")].some(
        (i) => i.complete && i.naturalWidth > 100,
      ),
    );
    await shot(page, size + "-pdf-parsed-image");
    await dialog.getByRole("tab", { name: "PDF 原页", exact: true }).click();
    await page.waitForFunction(
      () => document.querySelector(".pdf-source canvas")?.width > 0,
    );
    await shot(page, size + "-pdf-source");
    await dialog
      .getByRole("button", { name: "校对并添加测试点", exact: true })
      .click();
    await editor.getByLabel("题目标题", { exact: true }).waitFor();
    assert.equal(
      await editor.getByLabel("题目标题", { exact: true }).inputValue(),
      "二阶矩阵",
    );
    assert.equal(
      await editor.getByLabel("样例 2 输出", { exact: true }).inputValue(),
      "0",
    );
    assert.ok(
      (
        await editor.getByLabel("说明（可选）", { exact: true }).inputValue()
      ).includes("assets/"),
    );
    await editor.getByRole("tab", { name: "PDF 原页", exact: true }).click();
    const canvas = editor.locator(".page-canvas");
    await canvas.scrollIntoViewIfNeeded();
    await page.waitForTimeout(500);
    await editor.locator(".pdf-source summary").click();
    for (const [label, value] of [
      ["左", "5"],
      ["上", "72"],
      ["宽", "85"],
      ["高", "20"],
    ])
      await editor
        .getByRole("spinbutton", { name: "截取" + label, exact: true })
        .fill(value);
    await editor
      .getByRole("button", { name: "添加框选图片", exact: true })
      .click();
    await page.waitForFunction(
      () => document.querySelectorAll(".image-assets img").length === 2,
    );
    await editor
      .locator("input[type=file]")
      .last()
      .setInputFiles([test("one.in", "1 1\n0\n"), test("one.out", "0\n")]);
    await editor
      .locator(".el-checkbox")
      .filter({ hasText: "已对照原文核对题面、公式、图片和样例" })
      .click();
    assert.ok(
      await editor
        .getByRole("checkbox", { name: "已对照原文核对题面、公式、图片和样例" })
        .isChecked(),
    );
    await shot(page, size + "-pdf-editor");
    pending = page.waitForResponse(
      (r) =>
        r.request().method() === "POST" && r.url().endsWith(base + "/problems"),
    );
    await editor.getByRole("button", { name: "保存题目", exact: true }).click();
    result = await pending;
    assert.equal(result.status(), 200, await result.text());
    await dialog.waitFor();
    await dialog
      .getByRole("button", { name: "收起导入队列", exact: true })
      .click();
    paper = await (await call(page, "GET", base)).json();
    assert.equal(paper.event.submission_version, 0);
    assert.equal(paper.problems.length, 2);
    const submitted = await call(page, "POST", base + "/review", {
      action: "submit",
      revision: paper.event.review_revision,
    });
    assert.equal(submitted.status(), 200, await submitted.text());
    paper = await (await call(page, "GET", base)).json();
    assert.equal(paper.event.submission_version, 1);
    const conflict = await call(page, "POST", base + "/review", {
      action: "submit",
      revision: paper.event.review_revision,
    });
    assert.equal(conflict.status(), 409);
    checks.push(
      size +
        ": button order, Markdown image packaging, all 10 PDF questions, embedded raster, manual crop, exact samples, save, submission version",
    );
    await page.close();
  }
  assert.deepEqual(errors, []);
  await writeFile(
    path.join(output, "results.json"),
    JSON.stringify({ checks, errors }, null, 2),
  );
  console.log(JSON.stringify({ checks, errors }, null, 2));
} catch (e) {
  for (const context of browser.contexts())
    for (const [i, page] of context.pages().entries()) {
      console.error(
        "Page",
        page.url(),
        (await page.locator("body").innerText()).slice(-2500),
      );
      await page
        .screenshot({
          path: path.join(output, "failure-" + i + ".png"),
          fullPage: true,
        })
        .catch(() => {});
    }
  console.error("Errors", errors);
  throw e;
} finally {
  await browser.close();
}
