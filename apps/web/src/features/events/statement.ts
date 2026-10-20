export type StatementSections = {
  description: string;
  input_format: string;
  output_format: string;
  samples: Array<{ input: string; output: string }>;
  notes: string;
};
export function emptyStatement(): StatementSections {
  return {
    description: "",
    input_format: "",
    output_format: "",
    samples: [{ input: "", output: "" }],
    notes: "",
  };
}
export function renderStatement(s: StatementSections) {
  let text = `## 题目描述\n\n${s.description.trim()}\n\n## 输入格式\n\n${s.input_format.trim()}\n\n## 输出格式\n\n${s.output_format.trim()}\n`;
  s.samples.forEach((sample, i) => {
    for (const [label, value] of [
      ["输入", sample.input],
      ["输出", sample.output],
    ]) {
      let fence = "```";
      while (value.includes(fence)) fence += "`";
      text += `\n## 样例 ${i + 1} · ${label}\n\n${fence}text\n${value.replace(/\n$/, "")}\n${fence}\n`;
    }
  });
  if (s.notes.trim()) text += `\n## 说明\n\n${s.notes.trim()}\n`;
  return text;
}
export function validateStatement(s: StatementSections) {
  if (
    !s.description.trim() ||
    !s.input_format.trim() ||
    !s.output_format.trim()
  )
    return "请填写题目描述、输入格式和输出格式";
  if (
    !s.samples.length ||
    s.samples.length > 20 ||
    s.samples.some((x) => !x.output.trim())
  )
    return "请填写 1 至 20 组样例，每组须包含输出";
  if (s.samples.some((x) => x.input.length > 65536 || x.output.length > 65536))
    return "样例过长，请将大型数据放入测试点文件";
  return "";
}
