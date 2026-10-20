// @vitest-environment node
import { it, expect } from "vitest";
import fs from "node:fs";
import { importEventMarkdown } from "../src/features/events/documentImport";
it.skipIf(!process.env.OJ_IMPORT_MARKDOWN_FIXTURE)(
  "imports the supplied Markdown with image and exact sample",
  () => {
    const p = importEventMarkdown(
      fs.readFileSync(process.env.OJ_IMPORT_MARKDOWN_FIXTURE!, "utf8"),
    );
    expect(p.title).toContain("二阶矩阵");
    expect(p.sections.samples).toEqual([
      { input: "3 4\n1 2 1 0\n2 4 2 1\n0 3 3 0", output: "2" },
    ]);
    expect(p.sections.notes).toContain(
      "https://cdn.luogu.com.cn/upload/image_hosting/lcdtefnp.png",
    );
  },
);
