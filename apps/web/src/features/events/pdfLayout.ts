import { blankImportedProblem, type ImportedProblem } from "./documentImport";

// Layout extraction stays independent of PDF.js/the DOM so coordinates, sample
// columns and superscripts can be regression tested without a canvas or worker.
export type PDFText = {
  str: string;
  x: number;
  y: number;
  width: number;
  height: number;
};
export type PDFPageText = {
  page: number;
  width: number;
  height: number;
  items: PDFText[];
};
type Line = { y: number; height: number; items: PDFText[]; text: string };
export function pdfLines(items: PDFText[]): Line[] {
  const groups: Omit<Line, "text">[] = [];
  for (const item of [...items]
    .filter((i) => i.str.trim())
    .sort((a, b) => b.height - a.height || a.y - b.y)) {
    const candidates = groups.filter(
      (g) =>
        Math.abs(g.y - item.y) <=
        (item.height < g.height * 0.86 ? g.height * 0.62 : 1.5),
    );
    const line = candidates.sort(
      (a, b) => Math.abs(a.y - item.y) - Math.abs(b.y - item.y),
    )[0];
    if (line) line.items.push(item);
    else groups.push({ y: item.y, height: item.height, items: [item] });
  }
  return groups
    .sort((a, b) => a.y - b.y)
    .map((line) => ({
      ...line,
      items: line.items.sort((a, b) => a.x - b.x),
      text: lineText(line),
    }));
}
function lineText(line: Omit<Line, "text">): string {
  const items = [...line.items].sort((a, b) => a.x - b.x);
  let out = "",
    end = -1;
  for (let i = 0; i < items.length; i++) {
    const item = items[i],
      delta = item.y - line.y;
    if (
      item.height < line.height * 0.86 &&
      Math.abs(delta) > 1.2 &&
      /[\w)]$/.test(out)
    ) {
      let script = item.str;
      while (
        items[i + 1] &&
        Math.abs(items[i + 1].y - item.y) < 1 &&
        items[i + 1].height < line.height * 0.86
      )
        script += items[++i].str;
      const base = out.match(/(?:[0-9]+|[a-zA-Z])$/)![0];
      out =
        out.slice(0, -base.length) +
        `$${base}${delta < 0 ? "^" : "_"}{${script}}$`;
    } else {
      if (
        end >= 0 &&
        item.x - end >= line.height * 0.3 &&
        !out.endsWith(" ") &&
        !item.str.startsWith(" ")
      )
        out += " ";
      out += item.str;
    }
    end = items[i].x + items[i].width;
  }
  return out.trim();
}
const titlePattern =
  /^(?:Problem\s+([A-Z]|\d+)\s*[.．:：、-]\s*|(?:题目|题)\s*([A-Z]|\d+)\s*[.．:：、-]\s*)(.+)$/i;
const fieldPattern =
  /^(题目描述|问题描述|题目背景|Description|输入格式|输入说明|Input|输出格式|输出说明|Output|样例|示例|Examples?|Samples?|样例解释|样例说明|说明|提示|说明\/提示|数据范围)$/i;
function appendParagraph(old: string, lines: Line[]) {
  let text = "";
  lines.forEach((line, i) => {
    const previous = lines[i - 1];
    if (previous)
      text +=
        line.y - previous.y > previous.height * 1.95
          ? "\n\n"
          : /[\u3400-\u9fff，。；：、！？]$/.test(text) ||
              /^[\u3400-\u9fff，。；：、！？]/.test(line.text)
            ? ""
            : " ";
    text += line.text;
  });
  return [old, text].filter(Boolean).join("\n\n");
}
function tableSamples(lines: Line[], width: number, warnings: string[]) {
  const headers = lines.filter(
    (l) =>
      l.items.some((i) =>
        /^(standard\s+input|input|样例输入|输入)$/i.test(i.str.trim()),
      ) &&
      l.items.some((i) =>
        /^(standard\s+output|output|样例输出|输出)$/i.test(i.str.trim()),
      ),
  );
  const samples: { input: string; output: string }[] = [];
  for (let h = 0; h < headers.length; h++) {
    const header = headers[h],
      labels = header.items
        .filter((i) =>
          /^(standard\s+)?(input|output)$|^(样例)?(输入|输出)$/i.test(
            i.str.trim(),
          ),
        )
        .sort((a, b) => a.x - b.x);
    if (labels.length % 2) {
      warnings.push("样例表列数不明确，请对照原页检查");
      continue;
    }
    const items = lines
      .filter(
        (l) =>
          l.y > header.y + 2 &&
          l.y < (headers[h + 1]?.y ?? Infinity) &&
          !/^样例\s*\d/.test(l.text),
      )
      .flatMap((l) => l.items);
    // Column boundaries are the midpoints between header centres. This handles
    // both one input/output table and several side-by-side sample tables.
    const centres = labels.map((i) => i.x + i.width / 2);
    const boundaries = [
      0,
      ...centres.slice(1).map((c, i) => (c + centres[i]) / 2),
      width,
    ];
    for (let col = 0; col < labels.length; col += 2) {
      const cells = [0, 1].map((offset) =>
        pdfLines(
          items.filter(
            (i) =>
              i.x >= boundaries[col + offset] &&
              i.x < boundaries[col + offset + 1],
          ),
        ),
      );
      const rows = pdfLines(
        items.filter(
          (i) => i.x >= boundaries[col] && i.x < boundaries[col + 2],
        ),
      );
      if (!rows.length) continue;
      const cuts = [rows[0].y - 1];
      for (let r = 1; r < rows.length; r++)
        if (
          rows[r].y - rows[r - 1].y >
          Math.max(rows[r].height, rows[r - 1].height) * 1.8
        )
          cuts.push((rows[r].y + rows[r - 1].y) / 2);
      cuts.push(Infinity);
      for (let r = 0; r < cuts.length - 1; r++) {
        const pair = cells.map((column) =>
          column
            .filter((l) => l.y >= cuts[r] && l.y < cuts[r + 1])
            .map((l) => l.text)
            .join("\n"),
        );
        samples.push({ input: pair[0], output: pair[1] });
      }
    }
  }
  if (!headers.length)
    warnings.push("未识别成对的样例表头，请在编辑器补充样例");
  return samples;
}
export function parsePDFPages(pages: PDFPageText[]): ImportedProblem[] {
  const results: ImportedProblem[] = [];
  let current: ImportedProblem | undefined;
  let field:
    "description" | "input_format" | "output_format" | "notes" | "samples" =
    "description";
  for (const page of pages) {
    const lines = pdfLines(
      page.items.filter((i) => i.y > 40 && i.y < page.height - 35),
    );
    let buffer: Line[] = [];
    function flush() {
      if (current && buffer.length) {
        if (field === "samples")
          current.sections.samples.push(
            ...tableSamples(buffer, page.width, current.warnings),
          );
        else
          current.sections[field] = appendParagraph(
            current.sections[field],
            buffer,
          );
      }
      buffer = [];
    }
    for (const line of lines) {
      const title = line.text.match(titlePattern);
      if (title && line.height >= 12) {
        flush();
        current = blankImportedProblem(title[3].trim(), [page.page]);
        current.sections.samples = [];
        current.warnings.push(
          "PDF 公式、图表及样例须对照原页校对；未自动提取的插图可框选添加。",
        );
        results.push(current);
        field = "description";
        continue;
      }
      if (!current) continue; // Cover / contents preceding the first problem.
      if (!current.pages!.includes(page.page)) current.pages!.push(page.page);
      if (
        /^(输入文件|输出文件|Input file|Output file)\s*[:：]/i.test(line.text)
      )
        continue;
      const time = line.text.match(
        /(?:时间限制|Time limit)\s*[:：]\s*([\d.]+)\s*(ms|毫秒|秒|s(?:econds?)?)/i,
      );
      const memory = line.text.match(
        /(?:内存限制|Memory limit)\s*[:：]\s*(\d+)\s*(MiB|MB)/i,
      );
      if (time)
        current.time_limit_ms = Math.round(
          Number(time[1]) * (/^(ms|毫秒)$/i.test(time[2]) ? 1 : 1000),
        );
      if (memory) current.memory_limit_mb = Number(memory[1]);
      if (time || memory) continue;
      if (fieldPattern.test(line.text.trim())) {
        flush();
        const h = line.text.toLowerCase();
        field = /输入|^input$/.test(h)
          ? "input_format"
          : /输出|^output$/.test(h)
            ? "output_format"
            : /说明|解释|提示|数据范围/.test(h)
              ? "notes"
              : /样例|示例|example|sample/.test(h)
                ? "samples"
                : "description";
      } else buffer.push(line);
    }
    flush();
  }
  for (const problem of results) {
    if (!problem.sections.samples.length)
      problem.sections.samples = [{ input: "", output: "" }];
    if (problem.sections.samples.some((s) => !s.output.trim()))
      problem.warnings.push("存在空白样例输出，请核对原页");
    problem.warnings = [...new Set(problem.warnings)];
  }
  if (!results.length)
    throw new Error(
      "未识别到题目标题。支持“Problem A. 标题”或“题目 1：标题”的文字版 PDF；扫描件请先转成 Markdown 或文字版 PDF。",
    );
  if (results.length > 100) throw new Error("每次最多导入 100 道题目");
  return results;
}
