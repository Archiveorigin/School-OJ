import { describe, expect, it } from "vitest";
import MarkdownIt from "markdown-it";
import {
  parseStatement,
  serializeStatement,
  validateStatement,
  type StatementDraft,
} from "../src/features/problems/statementTemplate";
const draft: StatementDraft = {
  description: "计算 $a+b$。",
  input: "两个整数。",
  output: "输出和。",
  hints: "0 ≤ a,b ≤ 100",
  samples: [{ input: "1 2", output: "3", explanation: "求和。" }],
};
describe("structured statement editor", () => {
  it("refuses an unclosed code block or sample headings that hide later sections", () => {
    expect(validateStatement({ ...draft, hints: "~~~text\n未结束" })).toContain("闭合");
    expect(validateStatement({ ...draft, hints: "> ~~~text\n> 正文\n> ~~~" })).toBe("");
    expect(
      validateStatement({ ...draft, description: "```cpp\nreturn 0;" }),
    ).toContain("闭合");
    expect(
      validateStatement({
        ...draft,
        samples: [{ ...draft.samples[0], explanation: "### 样例 2" }],
      }),
    ).toContain("五级");
  });
  it("generates exactly the prescribed section headings and round trips multiple samples", () => {
    const value = {
      ...draft,
      samples: [
        ...draft.samples,
        {
          input: "  2 3\n\n",
          output: "5\n",
          explanation: "![示意](attachments/a.png)",
        },
      ],
    };
    const markdown = serializeStatement(value),
      tokens = new MarkdownIt().parse(markdown, {});
    expect(
      tokens.filter((t) => t.type === "heading_open" && t.tag === "h2"),
    ).toHaveLength(5);
    expect(parseStatement(markdown)).toEqual({ draft: value, legacy: false });
    expect(validateStatement(value)).toBe("");
  });
  it("keeps headings inside sample text inert, including code fences", () => {
    const value = {
      ...draft,
      samples: [
        {
          input: "```\n## 输入格式\n~~~",
          output: "````\n# 样例",
          explanation: "",
        },
      ],
    };
    const markdown = serializeStatement(value);
    expect(parseStatement(markdown).draft).toEqual(value);
    expect(
      new MarkdownIt()
        .parse(markdown, {})
        .filter((t) => t.type === "heading_open" && t.tag === "h2"),
    ).toHaveLength(5);
  });
  it("requires core sections and a usable sample while permitting a program with no input", () => {
    expect(validateStatement({ ...draft, input: " " })).toContain("输入格式");
    expect(validateStatement({ ...draft, samples: [] })).toContain("样例");
    expect(
      validateStatement({
        ...draft,
        samples: [{ input: "", output: "", explanation: "" }],
      }),
    ).toContain("样例 1");
    expect(
      validateStatement({
        ...draft,
        samples: [{ input: "", output: "hello", explanation: "" }],
      }),
    ).toBe("");
  });
  it("rejects competing ATX and setext headings, but permits subheadings and code", () => {
    for (const content of ["# 外部标题", "## 输出格式", "输出格式\n---"])
      expect(validateStatement({ ...draft, description: content })).toContain(
        "三级",
      );
    expect(
      validateStatement({
        ...draft,
        description: "### 子问题\n\n```python\n# comment\n```",
      }),
    ).toBe("");
  });
  it("preserves unfamiliar legacy text and code while demoting its top level headings", () => {
    const source =
      "旧题面\n===\n\n内容 $a+b$\n\n## 自定义栏目\n\n```text\n## 原始数据\n```";
    const parsed = parseStatement(source);
    expect(parsed.legacy).toBe(true);
    expect(parsed.draft.description).toBe(
      "### 旧题面\n\n内容 $a+b$\n\n### 自定义栏目\n\n```text\n## 原始数据\n```",
    );
    expect(parsed.draft.input).toBe("");
  });
  it("measures the server UTF-8 byte limit including generated headings", () => {
    expect(
      validateStatement({ ...draft, hints: "中".repeat(70000) }),
    ).toContain("200000");
  });
});
