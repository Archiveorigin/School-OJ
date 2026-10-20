import {
  getDocument,
  GlobalWorkerOptions,
  Util,
  type PDFDocumentLoadingTask,
  type PDFDocumentProxy,
} from "pdfjs-dist";
import workerURL from "pdfjs-dist/build/pdf.worker.min.mjs?url";
import { parsePDFPages, type PDFPageText } from "./pdfLayout";
import { extractPDFImages } from "./pdfImages";
import { problemImageSources } from "./imageResources";
import type { ImportAsset } from "./documentImport";
GlobalWorkerOptions.workerSrc = workerURL;

export type PDFImportSession = {
  document: PDFDocumentProxy;
  task: PDFDocumentLoadingTask;
};
export async function readPDF(
  file: File,
  signal: AbortSignal,
  progress: (n: number, total: number) => void,
) {
  if (file.size > 20 * 1024 * 1024) throw new Error("PDF 不能超过 20 MiB");
  const root = import.meta.env.BASE_URL + "pdf-assets/";
  const task = getDocument({
    data: new Uint8Array(await file.arrayBuffer()),
    useSystemFonts: true,
    useWasm: false,
    maxImageSize: 16000000,
    stopAtErrors: true,
    cMapUrl: root + "cmaps/",
    cMapPacked: true,
    standardFontDataUrl: root + "standard_fonts/",
    wasmUrl: root + "wasm/",
  });
  const abort = () => {
    void task.destroy();
  };
  signal.addEventListener("abort", abort, { once: true });
  try {
    if (signal.aborted) throw new DOMException("已取消", "AbortError");
    const document = await task.promise;
    if (document.numPages > 60)
      throw new Error("每次最多读取 60 页，请拆分 PDF 后导入");
    const pages: PDFPageText[] = [];
    const images = new Map<string, ImportAsset>();
    let imageBytes = 0;
    for (let n = 1; n <= document.numPages; n++) {
      if (signal.aborted) throw new DOMException("已取消", "AbortError");
      const page = await document.getPage(n),
        viewport = page.getViewport({ scale: 1 }),
        content = await page.getTextContent();
      if (content.items.length > 20000)
        throw new Error(`第 ${n} 页文字对象过多，请拆分文档或转为 Markdown 导入`);
      pages.push({
        page: n,
        width: viewport.width,
        height: viewport.height,
        items: content.items.flatMap((item) => {
          if (!("str" in item)) return [];
          const transform = Util.transform(viewport.transform, item.transform);
          return [
            {
              str: item.str,
              x: transform[4],
              y: transform[5],
              width: item.width,
              height: Math.hypot(transform[2], transform[3]),
            },
          ];
        }),
      });
      // A preliminary text pass identifies actual question pages; cover logos
      // and contents are neither decoded nor imported as problem resources.
      if (
        pages
          .at(-1)!
          .items.some((i) =>
            /^(Problem\s+[A-Z0-9]+|题目?\s*[A-Z0-9]+)\s*[.．:：、-]/i.test(
              i.str,
            ),
          ) ||
        (pages.length > 1 &&
          pages
            .slice(0, -1)
            .some((p) =>
              p.items.some((i) =>
                /^Problem\s+[A-Z0-9]+[.．:：、-]/i.test(i.str),
              ),
            ))
      ) {
        const extracted = await extractPDFImages(page, signal);
        for (const asset of extracted.assets) {
          if (!images.has(asset.path)) {
            imageBytes += asset.data.length * 0.75;
            images.set(asset.path, asset);
          }
        }
        if (imageBytes > 40 * 1024 * 1024)
          throw new Error("本次导入图片总量超过 40 MiB，请拆分题册后导入");
        pages.at(-1)!.items.push(...extracted.items);
      }
      progress(n, document.numPages);
    }
    const problems = parsePDFPages(pages);
    for (const problem of problems) {
      problem.assets = problemImageSources(problem).flatMap((path) =>
        images.has(path) ? [images.get(path)!] : [],
      );
      if (
        problem.assets.length > 30 ||
        problem.assets.reduce((size, a) => size + a.data.length * 0.75, 0) >
          20 * 1024 * 1024
      )
        throw new Error(`“${problem.title}”的图片超过每题 30 张 / 20 MiB 限制`);
    }
    return { problems, session: { document, task } as PDFImportSession };
  } catch (error) {
    await task.destroy();
    throw error;
  } finally {
    signal.removeEventListener("abort", abort);
  }
}
