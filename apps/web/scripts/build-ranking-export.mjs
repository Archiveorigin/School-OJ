import { build } from "vite";
import vue from "@vitejs/plugin-vue";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";
import ElementPlus from "unplugin-element-plus/vite";
import { mkdir, writeFile } from "node:fs/promises";
const result = await build({
  configFile: false,
  define: { "process.env.NODE_ENV": JSON.stringify("production") },
  plugins: [
    vue(),
    Components({ dirs: [], dts: false, resolvers: [ElementPlusResolver()] }),
    ElementPlus(),
  ],
  build: {
    write: false,
    minify: true,
    cssCodeSplit: false,
    assetsInlineLimit: 10000000,
    lib: {
      entry: "src/features/leaderboard/offline.ts",
      formats: ["iife"],
      name: "OfflineRanking",
    },
    rollupOptions: { output: { inlineDynamicImports: true } },
  },
});
const outputs = (Array.isArray(result) ? result : [result]).flatMap(
  (r) => r.output,
);
const js = outputs
  .filter((o) => o.type === "chunk")
  .map((o) => o.code)
  .join("\n")
  .replace(/<\/script/gi, "<\\/script");
const css = outputs
  .filter((o) => o.type === "asset" && o.fileName.endsWith(".css"))
  .map((o) => o.source)
  .join("\n");
await mkdir("dist", { recursive: true });
await writeFile(
  "dist/offline-ranking.html",
  `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>排行榜</title><style>${css}</style></head><body><div id="app"></div><script id="ranking-data" type="application/json">__OJ_RANKING_DATA__</script><script>${js}</script></body></html>`,
);
