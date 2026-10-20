import MarkdownIt from "markdown-it";
import imageRule from "markdown-it/lib/rules_inline/image.mjs";
import linkRule from "markdown-it/lib/rules_inline/link.mjs";
import autolinkRule from "markdown-it/lib/rules_inline/autolink.mjs";
import type { RuleInline } from "markdown-it/lib/parser_inline.mjs";
import type { ImportedProblem, ImportAsset } from "./documentImport";

export const statementImageFields = [
  "description",
  "input_format",
  "output_format",
  "notes",
] as const;
type Occurrence = { url: string; alt: string; start: number; end: number };
const imageLink = (url: string) =>
  /\.(png|jpe?g|gif|webp)(?:[?#].*)?$/i.test(url);
function occurrences(source: string): Occurrence[] {
  const md = new MarkdownIt({ html: false }),
    found: Occurrence[] = [];
  // The real Markdown rules handle escaped brackets, reference images, titles,
  // code spans and nested parentheses. Only their recognized spans are changed.
  for (const [name, original] of [
    ["image", imageRule],
    ["link", linkRule],
    ["autolink", autolinkRule],
  ] as const) {
    md.inline.ruler.at(name, ((state, silent) => {
      const start = state.pos,
        before = state.tokens.length,
        success = original(state, silent);
      if (success && !silent) {
        const token = state.tokens[before],
          url = token?.attrGet(name === "image" ? "src" : "href") || "";
        if (url && (name === "image" || imageLink(url))) {
          state.env.importSpans?.push({
            url,
            alt: name === "image" ? token.content : "题面图片",
            start,
            end: state.pos,
          });
        }
      }
      return success;
    }) as RuleInline);
  }
  const env: { importSpans: Occurrence[]; references?: unknown } = {
    importSpans: [],
  };
  const blocks = md.parse(source, env);
  const lines = source.split("\n"),
    starts = [0];
  for (const line of lines) starts.push(starts.at(-1)! + line.length + 1);
  for (const token of blocks) {
    if (token.type !== "inline" || !token.map) continue;
    const spans: Occurrence[] = [];
    md.inline.parse(token.content, md, { ...env, importSpans: spans }, []);
    const from = starts[token.map[0]],
      to = starts[token.map[1]] ?? source.length;
    const offset = source.indexOf(token.content, from);
    if (offset < 0 || offset + token.content.length > to) {
      if (spans.length)
        throw new Error(
          "图片所在的列表或引用跨行较复杂，请将图片 Markdown 单独放在一行后重试",
        );
      continue;
    }
    for (const span of spans)
      found.push({
        ...span,
        start: offset + span.start,
        end: offset + span.end,
      });
  }
  return found
    .sort((a, b) => a.start - b.start || b.end - a.end)
    .filter(
      (span, i, all) =>
        !all
          .slice(0, i)
          .some((p) => p.start <= span.start && p.end >= span.end),
    );
}
export function problemImageSources(problem: ImportedProblem): string[] {
  return [
    ...new Set(
      statementImageFields.flatMap((f) =>
        occurrences(problem.sections[f]).map((o) => o.url),
      ),
    ),
  ];
}
export function replaceProblemImage(
  problem: ImportedProblem,
  url: string,
  path: string,
) {
  for (const field of statementImageFields) {
    let source = problem.sections[field];
    for (const item of occurrences(source)
      .filter((o) => o.url === url)
      .reverse()) {
      const alt = item.alt.replace(/[\[\]\\\n]/g, " ").trim() || "题面图片";
      source =
        source.slice(0, item.start) +
        `![${alt}](${path})` +
        source.slice(item.end);
    }
    problem.sections[field] = source;
  }
}
export function localImagePath(value: string): string {
  try {
    value = decodeURIComponent(value);
  } catch {
    throw new Error("图片路径的 URL 编码无效");
  }
  value = value.replace(/\\/g, "/").replace(/^(\.\/)+/, "");
  if (
    !value ||
    value.startsWith("/") ||
    /^[a-z][a-z\d+.-]*:/i.test(value) ||
    value.split("/").some((s) => s === "..") ||
    /[\u0000-\u001f]/.test(value)
  )
    throw new Error("图片请使用相对路径，不支持绝对路径或上级目录");
  return value;
}
export function matchImageFile(url: string, files: File[]): File | undefined {
  const path = localImagePath(url);
  const exact = files.filter(
    (f) => localImagePath(f.webkitRelativePath || f.name) === path,
  );
  if (exact.length === 1) return exact[0];
  const base = path.split("/").at(-1);
  const matches = files.filter((f) => f.name === base);
  if (matches.length > 1)
    throw new Error(`存在多个同名图片“${base}”，请逐项指定文件`);
  return matches[0];
}
export async function imageFileAsset(file: File): Promise<ImportAsset> {
  if (!file.size || file.size > 5 * 1024 * 1024)
    throw new Error("单张图片须为 1 字节至 5 MiB");
  const buffer = await file.arrayBuffer(),
    bytes = new Uint8Array(buffer),
    h = [...bytes.slice(0, 12)];
  const kind =
    h.slice(0, 8).join(",") === "137,80,78,71,13,10,26,10"
      ? "png"
      : h[0] === 255 && h[1] === 216 && h[2] === 255
        ? "jpeg"
        : String.fromCharCode(...h.slice(0, 6)).match(/^GIF8[79]a$/)
          ? "gif"
          : String.fromCharCode(...h.slice(0, 4)) === "RIFF" &&
              String.fromCharCode(...h.slice(8, 12)) === "WEBP"
            ? "webp"
            : "";
  if (!kind)
    throw new Error("图片内容须为 PNG、JPEG、GIF 或 WebP；不支持 SVG/HTML");
  const hash = [
    ...new Uint8Array(await crypto.subtle.digest("SHA-256", buffer)),
  ]
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
  const data = await new Promise<string>((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(String(r.result));
    r.onerror = () => reject(new Error("图片读取失败"));
    r.readAsDataURL(new Blob([buffer], { type: `image/${kind}` }));
  });
  return {
    name: file.name,
    path: `assets/${hash}.${kind === "jpeg" ? "jpg" : kind}`,
    content_type: `image/${kind}`,
    data,
  };
}
export function attachImportedImage(
  problem: ImportedProblem,
  url: string,
  asset: ImportAsset,
) {
  if (!problem.assets.some((a) => a.path === asset.path)) {
    if (
      problem.assets.length >= 30 ||
      problem.assets.reduce(
        (size, a) => size + (a.data.split(",")[1]?.length || 0) * 0.75,
        0,
      ) +
        (asset.data.split(",")[1]?.length || 0) * 0.75 >
        20 * 1024 * 1024
    )
      throw new Error("每题最多 30 张图片，图片总大小不能超过 20 MiB");
    problem.assets.push(asset);
  }
  replaceProblemImage(problem, url, asset.path);
}
