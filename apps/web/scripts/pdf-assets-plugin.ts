import type { Plugin } from "vite";
import { readdirSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { createRequire } from "node:module";

// All fonts/CMaps/decoders are served from this deployment. Importing a private
// booklet must never cause requests to a third-party PDF asset CDN.
export function pdfAssetsPlugin(): Plugin {
  const root = dirname(
    createRequire(import.meta.url).resolve("pdfjs-dist/package.json"),
  );
  const assets = new Map<string, string>();
  for (const dir of ["cmaps", "standard_fonts", "wasm"])
    for (const file of readdirSync(join(root, dir)))
      if (/\.(bcmap|pfb|ttf|wasm|js)$/.test(file) || file.startsWith("LICENSE"))
        assets.set(`/pdf-assets/${dir}/${file}`, join(root, dir, file));
  assets.set("/pdf-assets/LICENSE", join(root, "LICENSE"));
  return {
    name: "local-pdf-assets",
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        const path = assets.get((req.url || "").split("?")[0]);
        if (!path) return next();
        res.setHeader(
          "Content-Type",
          path.endsWith(".wasm")
            ? "application/wasm"
            : path.endsWith(".js")
              ? "text/javascript"
              : "application/octet-stream",
        );
        res.end(readFileSync(path));
      });
    },
    generateBundle() {
      for (const [name, path] of assets)
        this.emitFile({
          type: "asset",
          fileName: name.slice(1),
          source: readFileSync(path),
        });
    },
  };
}
