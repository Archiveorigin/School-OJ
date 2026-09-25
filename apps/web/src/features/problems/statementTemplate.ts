import MarkdownIt from "markdown-it";

export const statementSections = [
  { key: "description", title: "题目描述", required: true },
  { key: "input", title: "输入格式", required: true },
  { key: "output", title: "输出格式", required: true },
  { key: "hints", title: "数据范围与提示", required: false },
] as const;
export type StatementSection = (typeof statementSections)[number]["key"];
export interface StatementSample {
  input: string;
  output: string;
  explanation: string;
}
export interface StatementDraft {
  description: string;
  input: string;
  output: string;
  hints: string;
  samples: StatementSample[];
}
const parser = new MarkdownIt({ html: false });
const titles = ["题目描述", "输入格式", "输出格式", "样例", "数据范围与提示"];
export const emptySample = (): StatementSample => ({
  input: "",
  output: "",
  explanation: "",
});
function fence(value: string) {
  let length = 3;
  for (const match of value.matchAll(/`+/g))
    length = Math.max(length, match[0].length + 1);
  const marker = "`".repeat(length);
  return `${marker}text\n${value.replace(/\r\n?/g, "\n")}\n${marker}`;
}
export function serializeStatement(draft: StatementDraft) {
  const samples = draft.samples
    .map(
      (sample, index) =>
        `### 样例 ${index + 1}\n\n#### 输入\n\n${fence(sample.input)}\n\n#### 输出\n\n${fence(sample.output)}` +
        (sample.explanation.trim()
          ? `\n\n#### 说明\n\n${sample.explanation.trim()}`
          : ""),
    )
    .join("\n\n");
  return [draft.description, draft.input, draft.output, samples, draft.hints]
    .filter((body, index) => index !== 4 || body.trim())
    .map((body, index) => `## ${titles[index]}\n\n${body.trim()}`)
    .join("\n\n");
}
// Keep legacy content intact, demoting only headings that would compete with the fixed sections.
function demoteHeadings(source: string) {
  const lines = source.split("\n");
  for (const token of parser.parse(source, {}).reverse()) {
    if (
      token.type !== "heading_open" ||
      !["h1", "h2"].includes(token.tag) ||
      !token.map
    )
      continue;
    const [start, end] = token.map;
    if (/^ {0,3}#{1,2}(\s|$)/.test(lines[start]))
      lines[start] = lines[start].replace(/^( {0,3})#{1,2}(?=\s|$)/, "$1###");
    else
      lines.splice(
        start,
        end - start,
        `### ${lines.slice(start, end - 1).join(" ")}`,
      );
  }
  return lines.join("\n");
}
export function parseStatement(source: string): {
  draft: StatementDraft;
  legacy: boolean;
} {
  source = source.replace(/\r\n?/g, "\n").trim();
  const draft: StatementDraft = {
    description: "",
    input: "",
    output: "",
    hints: "",
    samples: [emptySample()],
  };
  if (!source) return { draft, legacy: false };
  const lines = source.split("\n"),
    tokens = parser.parse(source, {});
  const headings = tokens.flatMap((token, index) =>
    token.type === "heading_open" &&
    token.tag === "h2" &&
    token.level === 0 &&
    token.map
      ? [
          {
            title: tokens[index + 1].content,
            start: token.map[0],
            end: token.map[1],
          },
        ]
      : [],
  );
  if (
    (headings.length === 4 || headings.length === 5) &&
    headings[0].start === 0 &&
    headings.every((h, i) => h.title === titles[i])
  ) {
    const bodies = headings.map((h, i) =>
      lines
        .slice(h.end, headings[i + 1]?.start ?? lines.length)
        .join("\n")
        .trim(),
    );
    [draft.description, draft.input, draft.output, , draft.hints] = bodies;
    draft.hints ||= "";
    const sampleLines = bodies[3].split("\n"),
      sampleTokens = parser.parse(bodies[3], {});
    const starts = sampleTokens.flatMap((token, index) =>
      token.type === "heading_open" &&
      token.tag === "h3" &&
      token.level === 0 &&
      token.map &&
      /^样例 \d+$/.test(sampleTokens[index + 1].content)
        ? [token.map[0]]
        : [],
    );
    draft.samples = starts.map((start, i) => {
      const content = sampleLines
        .slice(start, starts[i + 1] ?? sampleLines.length)
        .join("\n");
      const parts = parser.parse(content, {}),
        sample = emptySample();
      for (let j = 0; j < parts.length; j++) {
        if (parts[j].type !== "heading_open" || parts[j].tag !== "h4") continue;
        const name = parts[j + 1].content,
          next = parts[j + 3];
        if ((name === "输入" || name === "输出") && next?.type === "fence")
          sample[name === "输入" ? "input" : "output"] = next.content.replace(
            /\n$/,
            "",
          );
        if (name === "说明" && parts[j].map)
          sample.explanation = content
            .split("\n")
            .slice(parts[j].map![1])
            .join("\n")
            .trim();
      }
      return sample;
    });
    // Only accept a lossless round trip; unfamiliar drafts remain available in the description field.
    const normalizedSource = draft.hints ? source : source.replace(/\n+## 数据范围与提示\s*$/, "");
    if (draft.samples.length && serializeStatement(draft).trim() === normalizedSource)
      return { draft, legacy: false };
  }
  return {
    draft: {
      description: demoteHeadings(source),
      input: "",
      output: "",
      hints: "",
      samples: [emptySample()],
    },
    legacy: true,
  };
}
export function validateStatement(draft: StatementDraft) {
  for (const section of statementSections) {
    if (section.required && !draft[section.key].trim())
      return `请填写${section.title}`;
  }
  if (!draft.samples.length) return "请添加至少一组样例";
  for (const [index, sample] of draft.samples.entries()) {
    if (!sample.input.trim() && !sample.output.trim())
      return `请填写样例 ${index + 1} 的输入或输出`;
    if (
      parser
        .parse(sample.explanation, {})
        .some(
          (t) =>
            t.type === "heading_open" &&
            ["h1", "h2", "h3", "h4"].includes(t.tag),
        )
    )
      return "样例标题由系统统一生成，样例说明内请使用五级及以下标题（#####）";
  }
  for (const content of [
    ...statementSections.map((s) => draft[s.key]),
    ...draft.samples.map((s) => s.explanation),
  ]) {
    const lines = content.replace(/\r\n?/g, "\n").split("\n");
    const bodyTokens = parser.parse(content, {});
    for (const token of bodyTokens) {
      if (token.type !== "fence" || !token.map) continue;
      const [start, end] = token.map;
      const closing = lines[end - 1]
        ?.trim()
        .replace(/^(?:>\s*)+/, "")
        .trim();
      if (
        end <= start + 1 ||
        !/^(`{3,}|~{3,})$/.test(closing) ||
        closing[0] !== token.markup[0] ||
        closing.length < token.markup.length
      )
        return "请闭合正文中的代码块（使用匹配的反引号或波浪线）";
    }
    if (
      bodyTokens.some(
        (t) => t.type === "heading_open" && ["h1", "h2"].includes(t.tag),
      )
    )
      return "栏目标题由系统统一生成，正文内请使用三级及以下标题（###）";
  }
  const markdown = serializeStatement(draft);
  if (new TextEncoder().encode(markdown).length > 200000)
    return "题面总大小不能超过 200000 字节";
  const tokens = parser.parse(markdown, {});
  const headings = tokens.flatMap((t, i) =>
    t.type === "heading_open" && t.tag === "h2" && t.level === 0
      ? [tokens[i + 1].content]
      : [],
  );
  if (
    headings.length !== (draft.hints.trim() ? 5 : 4) ||
    headings.some((title, i) => title !== titles[i])
  )
    return "题面栏目格式不完整，请检查并闭合正文中的代码块";
  return "";
}
