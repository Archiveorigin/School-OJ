import { OPS, Util, type PDFPageProxy } from "pdfjs-dist";
import { imageFileAsset } from "./imageResources";
import type { ImportAsset } from "./documentImport";
import type { PDFText } from "./pdfLayout";

// Raster illustrations are cropped from the rendered page at their PDF transform
// bounds. This preserves masks, colour spaces and rotations without guessing the
// image's raw encoding. Vector illustrations remain available via manual crop.
export async function extractPDFImages(
  page: PDFPageProxy,
  signal: AbortSignal,
): Promise<{ assets: ImportAsset[]; items: PDFText[] }> {
  const list = await page.getOperatorList(),
    base = page.getViewport({ scale: 1 }),
    boxes: { x: number; y: number; w: number; h: number }[] = [];
  let matrix = [1, 0, 0, 1, 0, 0];
  const stack: number[][] = [];
  for (let i = 0; i < list.fnArray.length; i++) {
    const op = list.fnArray[i],
      args = list.argsArray[i];
    if (op === OPS.save) stack.push([...matrix]);
    else if (op === OPS.restore) matrix = stack.pop() || [1, 0, 0, 1, 0, 0];
    else if (op === OPS.transform) matrix = Util.transform(matrix, args);
    else if (
      op === OPS.paintImageXObject ||
      op === OPS.paintInlineImageXObject
    ) {
      const m = Util.transform(base.transform, matrix),
        points = [
          [0, 0],
          [0, 1],
          [1, 0],
          [1, 1],
        ].map(([x, y]) => [
          m[0] * x + m[2] * y + m[4],
          m[1] * x + m[3] * y + m[5],
        ]);
      const x = Math.max(0, Math.min(...points.map((p) => p[0]))),
        y = Math.max(0, Math.min(...points.map((p) => p[1])));
      const w = Math.min(base.width, Math.max(...points.map((p) => p[0]))) - x,
        h = Math.min(base.height, Math.max(...points.map((p) => p[1]))) - y;
      if (w > 12 && h > 12 && w * h < base.width * base.height * 0.9)
        boxes.push({ x, y, w, h });
    }
  }
  if (!boxes.length) return { assets: [], items: [] };
  if (boxes.length > 30)
    throw new Error(
      `第 ${page.pageNumber} 页图片过多，请拆分文档或使用 Markdown 导入`,
    );
  const scale = Math.min(1.8, 1600 / base.width, 2000 / base.height),
    viewport = page.getViewport({ scale }),
    canvas = document.createElement("canvas");
  canvas.width = Math.ceil(viewport.width);
  canvas.height = Math.ceil(viewport.height);
  const task = page.render({ canvas, viewport }),
    abort = () => task.cancel();
  signal.addEventListener("abort", abort, { once: true });
  try {
    if (signal.aborted) throw new DOMException("已取消", "AbortError");
    await task.promise;
    const assets: ImportAsset[] = [],
      items: PDFText[] = [];
    for (const [index, box] of boxes.entries()) {
      if (signal.aborted) throw new DOMException("已取消", "AbortError");
      const crop = document.createElement("canvas");
      crop.width = Math.ceil(box.w * scale);
      crop.height = Math.ceil(box.h * scale);
      crop
        .getContext("2d")!
        .drawImage(
          canvas,
          box.x * scale,
          box.y * scale,
          box.w * scale,
          box.h * scale,
          0,
          0,
          crop.width,
          crop.height,
        );
      const blob = await new Promise<Blob | null>((resolve) =>
        crop.toBlob(resolve, "image/png"),
      );
      crop.width = crop.height = 0;
      if (!blob) throw new Error("PDF 图片提取失败，请使用原页框选图片");
      const asset = await imageFileAsset(
        new File([blob], `第${page.pageNumber}页插图${index + 1}.png`, {
          type: "image/png",
        }),
      );
      assets.push(asset);
      items.push({
        str: `![原文插图](${asset.path})`,
        x: box.x,
        y: box.y + Math.min(box.h / 2, 10),
        width: box.w,
        height: 10,
      });
    }
    return { assets, items };
  } finally {
    signal.removeEventListener("abort", abort);
    canvas.width = canvas.height = 0;
  }
}
