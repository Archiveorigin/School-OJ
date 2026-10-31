import { chromium } from "playwright";
import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import { pathToFileURL } from "node:url";
import path from "node:path";
const out = path.resolve("../../tmp/export-qa");
await mkdir(out, { recursive: true });
const browser = await chromium.launch();
try {
  for (const viewport of [
    { width: 1440, height: 1100 },
    { width: 390, height: 844 },
  ]) {
    const page = await browser.newPage({ viewport, reducedMotion: "reduce" });
    await page.context().setOffline(true);
    await page.goto(pathToFileURL(path.join(out, "standings.html")).href);
    await page.locator(".penalty-row").first().waitFor();
    assert.equal(await page.title(), "秋季程序设计赛");
    assert.equal(await page.locator(".penalty-row").count(), 4);
    assert.equal(await page.locator(".is-first").count(), 1);
    assert.equal(
      await page.locator(".time-progress-slider").inputValue(),
      "7200",
    );
    assert(await page.getByText("最终排名", { exact: true }).isVisible());
    assert.equal(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
      true,
    );
    await page.screenshot({
      path: path.join(out, `export-${viewport.width}.png`),
      fullPage: true,
    });
    await page.getByRole("searchbox", { name: "筛选姓名" }).fill("普通选手");
    await page.waitForFunction(
      () => document.querySelectorAll(".penalty-row").length === 1,
    );
    await page.getByRole("searchbox", { name: "筛选姓名" }).fill("");
    await page.context().setOffline(false);
    await page.route("**/api/v1/**", (route) => {
      const u = route.request().url();
      const data = u.includes("/members")
        ? [
            {
              user_id: 1,
              name: "张三",
              email: "zhang@test",
              user_role: "student",
              team_role: "member",
            },
            {
              user_id: 2,
              name: "李四",
              email: "li@test",
              user_role: "teacher",
              team_role: "admin",
            },
          ]
        : u.includes("/ranking/export")
          ? "export-content"
          : u.includes("/ranking")
            ? {
                title: "秋季程序设计赛",
                scoring_rule: "acm",
                lifecycle: "completed",
                completion_checks: { judging: 0, ungraded: 0 },
                rows: [],
                problems: [],
              }
            : [];
      return route.fulfill({ json: data });
    });
    await page.goto(
      "http://127.0.0.1:4183/tests/e2e/ranking-export-members.html",
    );
    const table = page.locator(".member-panel .el-table");
    await table.getByText("张三", { exact: true }).waitFor();
    const select = page.getByRole("combobox", { name: "选择成员" });
    await select.fill("李四");
    await page.getByRole("option").filter({ hasText: "李四" }).click();
    assert.equal(await table.getByText("张三", { exact: true }).count(), 1);
    await page.getByRole("button", { name: "查找", exact: true }).click();
    assert.equal(await table.getByText("张三", { exact: true }).count(), 0);
    assert.equal(await table.getByText("李四", { exact: true }).count(), 1);
    await page.screenshot({
      path: path.join(out, `members-${viewport.width}.png`),
      fullPage: true,
    });
    for (const [label, ext] of [
      ["导出 Excel", "xlsx"],
      ["导出网页榜单", "html"],
    ]) {
      const download = page.waitForEvent("download");
      await page.getByRole("button", { name: label, exact: true }).click();
      assert.equal(
        (await download).suggestedFilename(),
        `秋季程序设计赛.${ext}`,
      );
    }
    console.log(
      `HTML export, member filter, title filenames passed ${viewport.width}x${viewport.height}`,
    );
    await page.close();
  }
} finally {
  await browser.close();
}
