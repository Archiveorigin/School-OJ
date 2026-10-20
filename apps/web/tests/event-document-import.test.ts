import { describe, it, expect } from "vitest";
import { importEventMarkdown } from "../src/features/events/documentImport";
const markdown =
  "# B4264 二阶矩阵\n\n## 题目背景\n背景\n\n## 题目描述\n满足 $D_{1,1} = D_{2,2}$。\n\n## 输入格式\n两个整数\n\n## 输出格式\n答案\n\n## 输入输出样例 #1\n### 输入 #1\n```\n  3 4\n0 1\n```\n### 输出 #1\n```\n2\n```\n## 说明/提示\n### 样例解释\n![](https://cdn.example.com/picture.png)\n### 数据范围\n$n \\le 500$";
describe("event document import", () => {
  it("imports Luogu headings without moving hints into a sample", () => {
    const p = importEventMarkdown(markdown);
    expect(p.sections.description).toContain("背景");
    expect(p.sections.samples).toEqual([{ input: "  3 4\n0 1", output: "2" }]);
    expect(p.sections.notes).toContain("### 样例解释");
    expect(p.sections.notes).toContain("$n \\le 500$");
    expect(p.sections.notes).toContain("### 数据范围");
    expect(p.sections.notes).not.toContain("说明/提示");
  });
  it("supports canonical event sample headings", () => {
    const p = importEventMarkdown(
      markdown
        .replace("## 输入输出样例 #1\n### 输入 #1", "## 样例 1 · 输入")
        .replace("### 输出 #1", "## 样例 1 · 输出"),
    );
    expect(p.sections.samples).toEqual([{ input: "  3 4\n0 1", output: "2" }]);
  });
  it("rejects multiple problems instead of merging them", () => {
    expect(() => importEventMarkdown(markdown + "\n# 第二题\n正文")).toThrow(
      "一个一级标题",
    );
  });
});
