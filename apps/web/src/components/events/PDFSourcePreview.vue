<template>
  <section class="pdf-source">
    <div class="source-tools">
      <el-select v-model="pageNumber" aria-label="PDF 原页"
        ><el-option
          v-for="p in pages"
          :key="p"
          :value="p"
          :label="`原文第 ${p} 页`"
      /></el-select>
      <el-button :disabled="loading || !hasSelection" @click="crop"
        >添加框选图片</el-button
      >
    </div>
    <p class="muted">
      在原页上拖动框选插图或复杂公式，图片会插入当前选择的题面栏目。
    </p>
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <div
      class="page-canvas"
      v-loading="loading"
      @pointerdown="start"
      @pointermove="move"
      @pointerup="stop"
      @pointercancel="stop"
    >
      <canvas ref="canvas" aria-label="PDF 原页，可拖动框选图片" />
      <div
        v-if="hasSelection"
        class="selection"
        :style="{
          left: box.x + '%',
          top: box.y + '%',
          width: box.w + '%',
          height: box.h + '%',
        }"
      />
    </div>
    <details>
      <summary>精确设置截取区域（百分比）</summary>
      <div class="crop-fields">
        <label
          v-for="(label, key) in { x: '左', y: '上', w: '宽', h: '高' }"
          :key="key"
          >{{ label
          }}<el-input-number
            v-model="box[key]"
            :min="0"
            :max="100"
            :precision="1"
            :aria-label="`截取${label}`"
        /></label>
      </div>
    </details>
  </section>
</template>
<script setup lang="ts">
import { ref, reactive, computed, watch, onBeforeUnmount, nextTick } from "vue";
import type { PDFImportSession } from "../../features/events/pdfImport";
import type { RenderTask } from "pdfjs-dist";
const props = defineProps<{ session: PDFImportSession; pages: number[] }>();
const emit = defineEmits<{ image: [File] }>();
const pageNumber = ref(props.pages[0]),
  canvas = ref<HTMLCanvasElement>(),
  loading = ref(false),
  error = ref("");
const box = reactive({ x: 0, y: 0, w: 0, h: 0 }),
  hasSelection = computed(() => box.w > 0.2 && box.h > 0.2);
let task: RenderTask | undefined,
  sequence = 0,
  anchor: { x: number; y: number } | undefined;
watch(
  pageNumber,
  async () => {
    const current = ++sequence;
    task?.cancel();
    loading.value = true;
    error.value = "";
    Object.assign(box, { x: 0, y: 0, w: 0, h: 0 });
    try {
      const page = await props.session.document.getPage(pageNumber.value);
      await nextTick();
      if (current !== sequence || !canvas.value) return;
      const base = page.getViewport({ scale: 1 }),
        viewport = page.getViewport({
          scale: Math.min(1.8, 1400 / base.width, 2000 / base.height),
        });
      canvas.value.width = viewport.width;
      canvas.value.height = viewport.height;
      task = page.render({ canvas: canvas.value, viewport });
      await task.promise;
    } catch (e) {
      if (
        current === sequence &&
        (e as Error).name !== "RenderingCancelledException"
      )
        error.value = "原页显示失败，请重新导入 PDF";
    } finally {
      if (current === sequence) loading.value = false;
    }
  },
  { immediate: true },
);
onBeforeUnmount(() => {
  sequence++;
  task?.cancel();
});
function point(e: PointerEvent) {
  const r = canvas.value!.getBoundingClientRect();
  return {
    x: Math.max(0, Math.min(100, ((e.clientX - r.left) / r.width) * 100)),
    y: Math.max(0, Math.min(100, ((e.clientY - r.top) / r.height) * 100)),
  };
}
function start(e: PointerEvent) {
  if (loading.value || !canvas.value || e.button !== 0) return;
  anchor = point(e);
  (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
  Object.assign(box, { ...anchor, w: 0, h: 0 });
}
function move(e: PointerEvent) {
  if (!anchor) return;
  const p = point(e);
  Object.assign(box, {
    x: Math.min(p.x, anchor.x),
    y: Math.min(p.y, anchor.y),
    w: Math.abs(p.x - anchor.x),
    h: Math.abs(p.y - anchor.y),
  });
}
function stop() {
  anchor = undefined;
}
async function crop() {
  if (!canvas.value || !hasSelection.value) return;
  const source = canvas.value,
    x = (source.width * box.x) / 100,
    y = (source.height * box.y) / 100,
    w = Math.min(source.width - x, (source.width * box.w) / 100),
    h = Math.min(source.height - y, (source.height * box.h) / 100);
  if (w < 2 || h < 2) {
    error.value = "请选择有效的截取区域";
    return;
  }
  const result = document.createElement("canvas");
  result.width = Math.ceil(w);
  result.height = Math.ceil(h);
  result.getContext("2d")!.drawImage(source, x, y, w, h, 0, 0, w, h);
  const blob = await new Promise<Blob | null>((resolve) =>
    result.toBlob(resolve, "image/png"),
  );
  if (blob)
    emit(
      "image",
      new File([blob], `原文第${pageNumber.value}页.png`, {
        type: "image/png",
      }),
    );
}
</script>
<style scoped>
.source-tools {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.source-tools .el-select {
  width: 150px;
}
.muted {
  color: var(--muted);
  font-size: 13px;
  line-height: 1.7;
}
.page-canvas {
  position: relative;
  touch-action: none;
  background: white;
  line-height: 0;
}
.page-canvas canvas {
  display: block;
  width: 100%;
  height: auto;
}
.selection {
  position: absolute;
  border: 2px solid var(--accent);
  background: color-mix(in srgb, var(--accent) 12%, transparent);
  pointer-events: none;
  box-sizing: border-box;
}
.crop-fields {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  padding: 12px 0;
}
.crop-fields label {
  display: grid;
  gap: 6px;
}
.crop-fields .el-input-number {
  width: 120px;
}
details {
  margin-top: 12px;
  font-size: 13px;
}
summary {
  cursor: pointer;
  color: var(--muted);
}
</style>
