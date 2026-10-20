// @vitest-environment node
// Optional local acceptance fixtures; the user's original booklet is not bundled
// or published with the repository. CI uses the synthetic layout tests instead.
import { it, expect } from "vitest";
import fs from "node:fs";
import { parsePDFPages } from "../src/features/events/pdfLayout";
import { importEventMarkdown } from "../src/features/events/documentImport";
it.skipIf(!process.env.OJ_IMPORT_PDF_FIXTURE)(
  "parses the supplied ten-problem booklet and all twenty samples",
  async () => {
    const { getDocument } = await import("pdfjs-dist/legacy/build/pdf.mjs");
    const task = getDocument({
      data: new Uint8Array(fs.readFileSync(process.env.OJ_IMPORT_PDF_FIXTURE!)),
      useSystemFonts: true,
    });
    try {
      const pdf = await task.promise,
        pages = [];
      for (let n = 1; n <= pdf.numPages; n++) {
        const page = await pdf.getPage(n),
          content = await page.getTextContent(),
          height = page.view[3];
        pages.push({
          page: n,
          width: page.view[2],
          height,
          items: content.items
            .filter((i: any) => "str" in i)
            .map((i: any) => ({
              str: i.str,
              x: i.transform[4],
              y: height - i.transform[5],
              width: i.width,
              height: i.height,
            })),
        });
      }
      const imported = parsePDFPages(pages);
      expect(imported.map((p) => p.title)).toEqual([
        "陶陶摘苹果",
        "二阶矩阵",
        "津津的储蓄计划",
        "小鱼的数字游戏",
        "独木桥",
        "月份天数",
        "多项式输出",
        "小书童——凯撒密码",
        "校门外的树",
        "三角函数",
      ]);
      expect(
        imported.map((p) => p.sections.samples.map((s) => s.output)),
      ).toEqual([
        ["5", "0"],
        ["2", "0"],
        ["-2", "1580"],
        ["5 1 4 1 3", "2147483647"],
        ["2 4", "0 0"],
        ["28", "29"],
        ["2x^4-3x^3+x-1", "x"],
        ["zab", "qingdao"],
        ["5", "0"],
        ["3/5", "3/5"],
      ]);
      expect(
        imported.map((p) => p.sections.samples.map((s) => s.input)),
      ).toEqual([
        [
          "100 200 150 140 129 134 167 198 200 111\n110",
          "200 200 200 200 200 200 200 200 200 200\n100",
        ],
        ["3 4\n1 2 1 0\n2 4 2 1\n0 3 3 0", "1 1\n0"],
        [
          "0\n349\n0\n0\n0\n0\n0\n0\n0\n0\n0\n0",
          "290\n230\n280\n200\n300\n170\n330\n50\n90\n80\n200\n60",
        ],
        ["3 1 4 1 5 0", "2147483647 0"],
        ["5\n2\n2 4", "1\n0"],
        ["1900 2", "2000 2"],
        ["4\n2 -3 0 1 -1", "1\n1 0"],
        ["2\nxyz", "26\nqingdao"],
        ["10 2\n2 5\n4 7", "1 1\n0 1"],
        ["3 4 5", "12 9 15"],
      ]);
      expect(imported[1].sections.description).toContain("$D_{1,1}$");
      expect(imported[3].sections.input_format).toContain("$2^{31}$");
      expect(imported[8].sections.input_format).toContain("$10^{4}$");
      expect(imported[9].sections.input_format).toContain("$10^{9}$");
      expect(
        imported.every(
          (p) => p.time_limit_ms === 1000 && p.memory_limit_mb === 256,
        ),
      ).toBe(true);
    } finally {
      await task.destroy();
    }
  },
  60000,
);
it.skipIf(!process.env.OJ_IMPORT_MARKDOWN_FIXTURE)(
  "imports the supplied Markdown with image and exact sample",
  () => {
    const p = importEventMarkdown(
      fs.readFileSync(process.env.OJ_IMPORT_MARKDOWN_FIXTURE!, "utf8"),
    );
    expect(p.title).toBe("B4264 [GESP202503 四级] 二阶矩阵");
    expect(p.sections.samples).toEqual([
      { input: "3 4\n1 2 1 0\n2 4 2 1\n0 3 3 0", output: "2" },
    ]);
    expect(p.sections.notes).toContain(
      "https://cdn.luogu.com.cn/upload/image_hosting/lcdtefnp.png",
    );
  },
);
