import { describe, it, expect } from "vitest";
import { importEventMarkdown } from "../src/features/events/documentImport";
import {
  problemImageSources,
  replaceProblemImage,
  localImagePath,
  matchImageFile,
} from "../src/features/events/imageResources";
import {
  parsePDFPages,
  pdfLines,
  type PDFText,
} from "../src/features/events/pdfLayout";

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
  it("rewrites real image references and image links while protecting code and ordinary links", () => {
    const p = importEventMarkdown(markdown);
    p.sections.description =
      '![first](./images/a.png "caption")\n\n![ref][diagram]\n\n[diagram]: ./images/a.png\n\n[https://cdn.example.com/a.png](https://cdn.example.com/a.png)\n\n`![code](./images/a.png)`\n\n```text\n![fence](./images/a.png)\n```\n\n[来源](https://example.com/problem)';
    expect(problemImageSources(p)).toEqual([
      "./images/a.png",
      "https://cdn.example.com/a.png",
      "https://cdn.example.com/picture.png",
    ]);
    replaceProblemImage(p, "./images/a.png", "assets/abcd.png");
    replaceProblemImage(p, "https://cdn.example.com/a.png", "assets/efgh.png");
    expect(p.sections.description).toContain("![first](assets/abcd.png)");
    expect(p.sections.description).toContain("![ref](assets/abcd.png)");
    expect(p.sections.description).toContain("![题面图片](assets/efgh.png)");
    expect(p.sections.description).toContain("`![code](./images/a.png)`");
    expect(p.sections.description).toContain(
      "```text\n![fence](./images/a.png)",
    );
    expect(p.sections.description).toContain(
      "[来源](https://example.com/problem)",
    );
  });
  it("normalizes safe image paths and refuses ambiguous or escaping filenames", () => {
    expect(localImagePath("./images/%E5%9B%BE.png")).toBe("images/图.png");
    for (const path of [
      "../a.png",
      "assets/%2e%2e/a.png",
      "C:\\a.png",
      "/a.png",
      "data:x",
    ])
      expect(() => localImagePath(path)).toThrow();
    const one = new File(["a"], "图.png"),
      two = new File(["b"], "图.png");
    expect(matchImageFile("images/%E5%9B%BE.png", [one])).toBe(one);
    expect(() => matchImageFile("images/图.png", [one, two])).toThrow("同名");
  });
});
const item = (
  str: string,
  x: number,
  y: number,
  height = 10,
  width = str.length * 5,
): PDFText => ({ str, x, y, width, height });
describe("PDF layout", () => {
  it("restores exponents and subscripts instead of merging their digits", () => {
    expect(
      pdfLines([
        item("2", 10, 100, 10, 5),
        item("31", 15, 96, 7, 7),
        item("− 1", 25, 100),
      ]).map((l) => l.text),
    ).toEqual(["$2^{31}$ − 1"]);
    expect(
      pdfLines([item("D", 10, 100, 13, 9), item("1,1", 19, 102.6, 9, 12)]).map(
        (l) => l.text,
      ),
    ).toEqual(["$D_{1,1}$"]);
  });
  it("separates side-by-side sample tables and leaves the cover out", () => {
    const results = parsePDFPages([
      {
        page: 1,
        width: 600,
        height: 800,
        items: [item("试卷目录", 40, 60, 16)],
      },
      {
        page: 2,
        width: 600,
        height: 800,
        items: [
          item("Problem A. 测试题", 40, 67, 16),
          item("题目描述内容", 40, 160),
          item("输入格式", 40, 200, 13),
          item("两行", 40, 225),
          item("输出格式", 40, 250, 13),
          item("整数", 40, 275),
          item("样例", 40, 300, 13),
          item("standard input", 70, 340, 10, 70),
          item("standard output", 190, 340, 10, 75),
          item("standard input", 330, 340, 10, 70),
          item("standard output", 450, 340, 10, 75),
          item("1", 53, 365),
          item("2", 53, 377),
          item("3", 175, 365),
          item("4", 313, 365),
          item("5", 313, 377),
          item("9", 435, 365),
          item("说明", 40, 420, 13),
          item("注意边界", 40, 445),
        ],
      },
    ]);
    expect(results).toHaveLength(1);
    expect(results[0].pages).toEqual([2]);
    expect(results[0].sections.samples).toEqual([
      { input: "1\n2", output: "3" },
      { input: "4\n5", output: "9" },
    ]);
    expect(results[0].sections.notes).toBe("注意边界");
  });
});
