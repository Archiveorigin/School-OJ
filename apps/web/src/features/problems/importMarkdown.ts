import MarkdownIt from "markdown-it";
import {
  emptySample,
  serializeStatement,
  type StatementDraft,
  type StatementSample,
} from "./statementTemplate";

const parser = new MarkdownIt({ html: false });
const normalize = (value: string) =>
  value.replace(/[\s：:【】\[\]]/g, "").toLowerCase();

// Parse Markdown tokens, not regular expressions over raw Markdown: headings in
// code fences and formulas stay in their original content.
export function importProblemMarkdown(source: string) {
  source = source
    .replace(/^\uFEFF/, "")
    .replace(/\r\n?/g, "\n")
    .trim();
  if (new TextEncoder().encode(source).length > 200000)
    throw new Error("Markdown 文档不能超过 200000 字节");
  const tokens = parser.parse(source, {}),
    lines = source.split("\n");
  const headings = tokens.flatMap((token, i) =>
    token.type === "heading_open" && token.level === 0 && token.map
      ? [
          {
            name: tokens[i + 1].content,
            level: Number(token.tag.slice(1)),
            start: token.map[0],
            end: token.map[1],
          },
        ]
      : [],
  );
  const titleHeading = headings.find((h) => h.level === 1);
  if (!titleHeading || headings.filter((h) => h.level === 1).length !== 1)
    throw new Error("请使用一个一级标题（# 题目名称）标明题目标题");
  if (Array.from(titleHeading.name).length > 200)
    throw new Error("题目标题不能超过 200 个字");
  const draft: StatementDraft = {
    description: "",
    input: "",
    output: "",
    hints: "",
    samples: [],
  };
  const samples = new Map<number, StatementSample>();
  let sampleIndex = 1,
    active: "description" | "input" | "output" | "hints" | "explanation" =
      "description",
    matched = 0;
  const warnings: string[] = [];
  function sample(index = sampleIndex) {
    if (!samples.has(index)) samples.set(index, emptySample());
    return samples.get(index)!;
  }
  function append(body: string) {
    if (!body.trim()) return;
    if (active === "explanation")
      sample().explanation += (sample().explanation ? "\n\n" : "") + body;
    else draft[active] += (draft[active] ? "\n\n" : "") + body;
  }
  function sampleText(body: string) {
    const parts = parser.parse(body, {});
    if (
      parts.length === 1 &&
      (parts[0].type === "fence" || parts[0].type === "code_block")
    )
      return parts[0].content.replace(/\n$/, "");
    return body;
  }
  if (titleHeading.start > 0)
    append(lines.slice(0, titleHeading.start).join("\n").trim());
  for (const [i, heading] of headings.entries()) {
    const body = lines
      .slice(heading.end, headings[i + 1]?.start ?? lines.length)
      .join("\n")
      .trim();
    if (heading === titleHeading) {
      append(body);
      continue;
    }
    const name = normalize(heading.name).replace(
      /^(样例|示例)(\d+)[·・-](输入|输出)$/,
      "$1$3$2",
    );
    const field = /^(?:样例|示例)?(?:输入|input)(?:样例|示例)?#?(\d*)$/.exec(
      name,
    );
    const output = /^(?:样例|示例)?(?:输出|output)(?:样例|示例)?#?(\d*)$/.exec(
      name,
    );
    const isSampleField = /样例|示例|\d/.test(name) || heading.level >= 3;
    if ((field || output) && isSampleField) {
      const match = (field || output)!;
      if (match[1]) sampleIndex = Number(match[1]);
      if (sampleIndex < 1 || sampleIndex > 100)
        throw new Error("样例编号应为 1 至 100");
      const key = field ? "input" : "output";
      if (sample()[key])
        throw new Error(
          `样例 ${sampleIndex} 的${field ? "输入" : "输出"}重复，请检查文档`,
        );
      sample()[key] = sampleText(body);
      active = "explanation";
      matched++;
      continue;
    }
    if (
      /^(输入输出样例|输入输出示例|样例|示例|examples?|samples?)#?\d*$/.test(
        name,
      )
    ) {
      const index = name.match(/\d+$/)?.[0];
      if (index) sampleIndex = Number(index);
      active = "explanation";
      if (body) append(body);
      matched++;
      continue;
    }
    if (
      /^(样例|示例)?(说明|解释)$/.test(name) &&
      samples.size &&
      heading.level >= 3 &&
      active === "explanation"
    ) {
      append(body);
      continue;
    }
    if (/^(题目描述|问题描述|描述|description|题目背景|背景)$/.test(name))
      active = "description";
    else if (/^(输入格式|输入说明|输入|input)$/.test(name)) active = "input";
    else if (/^(输出格式|输出说明|输出|output)$/.test(name)) active = "output";
    else if (
      /^(数据范围.*|提示.*|说明(?:\/提示)?|约束.*|constraints|hints?|notes?)$/.test(
        name,
      )
    )
      active = "hints";
    else {
      append(
        `${"#".repeat(active === "explanation" ? 5 : Math.max(3, heading.level))} ${heading.name}\n\n${body}`,
      );
      warnings.push(`已保留子标题“${heading.name}”`);
      continue;
    }
    append(
      heading.level >= 3 && active === "hints"
        ? `${"#".repeat(heading.level)} ${heading.name}\n\n${body}`
        : body,
    );
    matched++;
  }
  if (!matched)
    throw new Error(
      "未识别题面栏目，请使用二级标题：题目描述、输入格式、输出格式、样例",
    );
  draft.samples = [...samples.entries()]
    .sort(([a], [b]) => a - b)
    .map(([, value]) => value);
  if (!draft.samples.length) {
    draft.samples = [emptySample()];
    warnings.push("请补充样例");
  }
  for (const [key, label] of [
    ["description", "题目描述"],
    ["input", "输入格式"],
    ["output", "输出格式"],
  ] as const)
    if (!draft[key]) warnings.push(`请补充${label}`);
  return {
    title: titleHeading.name,
    statement: serializeStatement(draft),
    draft,
    warnings,
  };
}
