<template>
  <el-dialog
    v-model="visible"
    :title="importType === 'pdf' ? '导入 PDF 题册' : '导入 Markdown 单题'"
    width="min(1120px, calc(100vw - 24px))"
    top="4vh"
    :close-on-click-modal="false"
    :before-close="close"
  >
    <p class="intro">
      导入 → 校对题面与图片 → 添加测试点。题目保存到本场后，再统一提交审核。
    </p>
    <div v-if="!problems.length" class="import-start">
      <el-tabs v-if="importType !== 'pdf'" v-model="mode"
        ><el-tab-pane label="选择文件" name="file" /><el-tab-pane
          label="粘贴 Markdown"
          name="paste"
      /></el-tabs>
      <label v-if="mode === 'file'" class="document-picker"
        >{{ importType === "pdf" ? "选择 PDF 题册" : "选择 Markdown 题面" }}
        <span>{{
          importType === "pdf"
            ? "整套题面自动分题 · 最大 20 MiB / 60 页"
            : "保留公式与图片引用 · UTF-8 格式 · 最大 200 KB"
        }}</span>
        <input
          type="file"
          :accept="importType === 'pdf' ? '.pdf' : '.md,.markdown'"
          :disabled="busy"
          aria-label="选择题面文件"
          @change="pickDocument"
        />
      </label>
      <template v-else
        ><el-input
          v-model="source"
          type="textarea"
          :rows="12"
          maxlength="200000"
          placeholder="# 题目名称&#10;&#10;## 题目描述&#10;…&#10;&#10;## 输入格式&#10;…&#10;&#10;## 输出格式&#10;…"
          aria-label="Markdown 题面"
        /><el-button
          class="parse-button"
          type="primary"
          :loading="busy"
          :disabled="!source.trim()"
          @click="parseMarkdown"
          >解析 Markdown</el-button
        ></template
      >
    </div>
    <div v-else class="import-layout">
      <nav class="import-list" aria-label="待导入题目">
        <p class="list-caption">{{ fileName }} · {{ problems.length }} 道题</p>
        <button
          v-for="(p, i) in problems"
          :key="p.key"
          type="button"
          :class="{ selected: selectedKey === p.key }"
          @click="selectedKey = p.key"
        >
          <span>{{ String(i + 1).padStart(2, "0") }}</span>
          <div>
            <strong>{{ p.title }}</strong
            ><small
              >{{ saved.has(p.key) ? "已保存到本场" : "待校对"
              }}<template v-if="p.pages">
                · 原文 {{ p.pages.join("、") }} 页</template
              ></small
            >
          </div>
        </button>
      </nav>
      <section v-if="selected" class="import-detail">
        <h2>{{ selected.title }}</h2>
        <ul v-if="selected.warnings.length" class="warnings">
          <li v-for="warning in selected.warnings" :key="warning">
            {{ warning }}
          </li>
        </ul>
        <section v-if="resources.length" class="resource-section">
          <div class="resource-heading">
            <h3>
              题面图片 ·
              {{
                unresolved.length
                  ? "待处理 " + unresolved.length + " 张"
                  : "已就绪"
              }}
            </h3>
            <label class="file-button"
              >匹配本地图片<input
                type="file"
                accept="image/png,image/jpeg,image/gif,image/webp"
                multiple
                :disabled="busy || saved.has(selected.key)"
                aria-label="匹配本地图片"
                @change="matchFiles"
            /></label>
          </div>
          <div v-for="url in resources" :key="url" class="resource-row">
            <span class="resource-path" :title="url">{{
              selected.assets.find((a) => a.path === url)?.name || url
            }}</span>
            <span
              v-if="selected.assets.some((a) => a.path === url)"
              class="resource-status"
              >已打包</span
            >
            <template v-else
              ><el-button
                v-if="url.startsWith('https://')"
                size="small"
                :loading="activeURL === url"
                :disabled="busy || saved.has(selected.key)"
                @click="fetchImage(url)"
                >导入外链图片</el-button
              ><label class="file-button"
                >指定图片<input
                  type="file"
                  accept="image/png,image/jpeg,image/gif,image/webp"
                  :disabled="busy || saved.has(selected.key)"
                  :aria-label="`为 ${url} 指定图片`"
                  @change="bindFile($event, url)" /></label
            ></template>
          </div>
          <p class="muted">
            相对路径按目录匹配，或匹配唯一同名文件。图片随题目保存，单张最大 5
            MiB，每题合计 20 MiB。
          </p>
        </section>
        <el-tabs v-model="previewTab"
          ><el-tab-pane label="解析预览" name="parsed" /><el-tab-pane
            v-if="pdfSession"
            label="PDF 原页"
            name="pdf"
        /></el-tabs>
        <PDFSourcePreview
          v-if="previewTab === 'pdf' && pdfSession && selected.pages"
          :key="selected.key"
          :session="pdfSession"
          :pages="selected.pages"
          @image="attachCrop"
        />
        <div v-else class="parsed-preview">
          <MarkdownRenderer
            :source="renderStatement(selected.sections)"
            :asset-urls="previewImages"
          />
        </div>
        <div class="import-action">
          <el-button
            type="primary"
            :disabled="busy || !!unresolved.length || saved.has(selected.key)"
            @click="choose"
            >{{
              saved.has(selected.key) ? "已保存到本场" : "校对并添加测试点"
            }}</el-button
          ><span v-if="unresolved.length" class="muted"
            >先补齐图片，再进入编辑器。</span
          >
        </div>
      </section>
    </div>
    <div v-if="busy" class="parse-progress">
      <el-progress :percentage="progress" :indeterminate="!progress" /><span
        role="status"
        >{{ status }}</span
      ><el-button size="small" @click="cancel">取消读取</el-button>
    </div>
    <el-alert
      v-if="error"
      class="import-error"
      type="error"
      :closable="false"
      :title="error"
    />
    <details class="import-rules">
      <summary>查看支持的题面规范</summary>
      <p>
        Markdown：一个一级标题作为题名；二级标题支持题目背景、题目描述、输入格式、输出格式、输入输出样例、说明/提示。样例输入与输出放在代码块中，支持多组编号。未知子标题和公式保留原文。
      </p>
      <p>
        图片：支持 ![说明](images/example.png)、引用式图片，以及指向
        PNG/JPEG/GIF/WebP 的链接。上传匹配图片，或导入公开 HTTPS
        图片。外链失败时可手动指定本地文件。
      </p>
      <p>
        PDF：支持文字版、单栏正文，以“Problem A. 标题”或“题目
        1：标题”分题；识别中英文章节和成对样例表头。复杂公式、插图可从原页框选添加；扫描件请先转成文字版
        PDF 或 Markdown。导入队列仅保留在当前页面，离开前请逐题保存。
      </p>
    </details>
    <template #footer
      ><el-button v-if="problems.length" :disabled="busy" @click="reset"
        >导入另一份文件</el-button
      ><el-button @click="close()">{{
        problems.length ? "收起导入队列" : "关闭"
      }}</el-button></template
    >
  </el-dialog>
</template>
<script setup lang="ts">
import { computed, markRaw, onBeforeUnmount, ref, shallowRef } from "vue";
import { ElMessageBox } from "element-plus";
import { generatedApi } from "../../api/client";
import { apiErrorMessage } from "../../api/errors";
import { eventPath, type EventKind } from "../../api/modules/events";
import {
  importEventMarkdown,
  type ImportedProblem,
  type ImportAsset,
} from "../../features/events/documentImport";
import {
  problemImageSources,
  attachImportedImage,
  imageFileAsset,
  matchImageFile,
} from "../../features/events/imageResources";
import { renderStatement } from "../../features/events/statement";
import type { PDFImportSession } from "../../features/events/pdfImport";
import MarkdownRenderer from "../MarkdownRenderer.vue";
import PDFSourcePreview from "./PDFSourcePreview.vue";
const props = defineProps<{
  kind: EventKind;
  eventId: number;
  revision: number;
}>();
const emit = defineEmits<{
  select: [ImportedProblem, PDFImportSession | undefined];
  pending: [boolean];
}>();
const importType = ref<"markdown" | "pdf">("markdown");
const visible = ref(false),
  mode = ref("file"),
  source = ref(""),
  problems = ref<ImportedProblem[]>([]),
  selectedKey = ref(""),
  saved = ref(new Set<string>()),
  fileName = ref(""),
  error = ref(""),
  busy = ref(false),
  status = ref(""),
  progress = ref(0),
  activeURL = ref(""),
  previewTab = ref("parsed");
const pdfSession = shallowRef<PDFImportSession>();
let controller: AbortController | undefined;
const selected = computed(() =>
  problems.value.find((p) => p.key === selectedKey.value),
);
const resources = computed(() =>
  selected.value ? problemImageSources(selected.value) : [],
);
const unresolved = computed(() =>
  resources.value.filter(
    (url) => !selected.value?.assets.some((a) => a.path === url),
  ),
);
const previewImages = computed(() =>
  Object.fromEntries(
    resources.value.map((url) => [
      url,
      selected.value?.assets.find((a) => a.path === url)?.data ||
        "data:image/gif;base64,R0lGODlhAQABAAD/ACwAAAAAAQABAAACADs=",
    ]),
  ),
);
function cancel() {
  controller?.abort();
}
function close(done?: () => void) {
  cancel();
  visible.value = false;
  done?.();
}
onBeforeUnmount(() => {
  cancel();
  void pdfSession.value?.task.destroy();
});
function updatePending() {
  emit(
    "pending",
    problems.value.some((p) => !saved.value.has(p.key)),
  );
}
function setProblems(list: ImportedProblem[]) {
  // Validate resource spans while still inside the import error boundary, before
  // reactive previews run. Unsupported nested Markdown must not break the page.
  for (const problem of list) problemImageSources(problem);
  problems.value = list;
  selectedKey.value = list[0]?.key || "";
  saved.value = new Set();
  previewTab.value = "parsed";
  updatePending();
}
async function reset() {
  if (problems.value.some((p) => !saved.value.has(p.key))) {
    try {
      await ElMessageBox.confirm(
        "尚有题目未保存到本场，清空此导入队列？",
        "重新导入",
        { confirmButtonText: "清空队列", cancelButtonText: "保留队列" },
      );
    } catch {
      return;
    }
  }
  await pdfSession.value?.task.destroy();
  pdfSession.value = undefined;
  setProblems([]);
  source.value = "";
  error.value = "";
}
function parseMarkdown() {
  try {
    error.value = "";
    setProblems([importEventMarkdown(source.value)]);
    fileName.value = "粘贴的 Markdown";
  } catch (e) {
    error.value = (e as Error).message;
  }
}
async function pickDocument(e: Event) {
  const input = e.target as HTMLInputElement,
    file = input.files?.[0];
  input.value = "";
  if (!file) return;
  busy.value = true;
  error.value = "";
  progress.value = 0;
  status.value = "正在读取题面";
  controller = new AbortController();
  const signal = controller.signal;
  try {
    if (/\.(md|markdown)$/i.test(file.name)) {
      if (file.size > 200000)
        throw new Error("Markdown 文档不能超过 200000 字节");
      const text = new TextDecoder("utf-8", { fatal: true }).decode(
        await file.arrayBuffer(),
      );
      if (!signal.aborted) setProblems([importEventMarkdown(text)]);
    } else if (/\.pdf$/i.test(file.name)) {
      const { readPDF } = await import("../../features/events/pdfImport");
      const result = await readPDF(file, signal, (n, total) => {
        progress.value = Math.round((n / total) * 100);
        status.value = `正在读取第 ${n} / ${total} 页`;
      });
      if (signal.aborted) {
        await result.session.task.destroy();
        return;
      }
      pdfSession.value = markRaw(result.session);
      setProblems(result.problems);
    } else throw new Error("请选择 .md、.markdown 或 .pdf 文件");
    fileName.value = file.name;
  } catch (e) {
    if (!signal.aborted) error.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}
async function imageAction(
  action: (p: ImportedProblem, signal: AbortSignal) => Promise<void>,
) {
  if (!selected.value || busy.value) return;
  const problem = selected.value;
  error.value = "";
  busy.value = true;
  progress.value = 0;
  status.value = "正在导入题面图片";
  controller = new AbortController();
  try {
    await action(problem, controller.signal);
  } catch (e) {
    if (!controller.signal.aborted) error.value = apiErrorMessage(e);
  } finally {
    busy.value = false;
    activeURL.value = "";
  }
}
function fetchImage(url: string) {
  activeURL.value = url;
  return imageAction(async (p, signal) => {
    const { data } = await generatedApi.importEventProblemImage<ImportAsset>({
      path: eventPath(props.kind, props.eventId),
      body: { url, revision: props.revision },
      signal,
      timeout: 20000,
    });
    if (!signal.aborted) attachImportedImage(p, url, data);
  });
}
function bindFile(e: Event, url: string) {
  const input = e.target as HTMLInputElement,
    file = input.files?.[0];
  input.value = "";
  if (file)
    void imageAction(async (p, signal) => {
      const asset = await imageFileAsset(file);
      if (!signal.aborted) attachImportedImage(p, url, asset);
    });
}
function matchFiles(e: Event) {
  const input = e.target as HTMLInputElement,
    files = Array.from(input.files || []);
  input.value = "";
  void imageAction(async (p, signal) => {
    let count = 0;
    for (const url of unresolved.value) {
      if (/^https?:\/\//.test(url)) continue;
      const file = matchImageFile(url, files);
      if (file) {
        const asset = await imageFileAsset(file);
        if (signal.aborted) return;
        attachImportedImage(p, url, asset);
        count++;
      }
    }
    if (!count) throw new Error("未匹配到图片，请检查文件名或逐项指定图片");
  });
}
function attachCrop(file: File) {
  void imageAction(async (p, signal) => {
    const asset = await imageFileAsset(file);
    if (signal.aborted) return;
    attachImportedImage(p, "", asset);
    p.sections.description += `\n\n![原文插图](${asset.path})`;
  });
}
function choose() {
  if (!selected.value || unresolved.value.length) return;
  emit("select", JSON.parse(JSON.stringify(selected.value)), pdfSession.value);
  visible.value = false;
}
defineExpose({
  open: async (type?: "markdown" | "pdf") => {
    if (type && problems.value.length) {
      await reset();
      if (problems.value.length) return;
    }
    if (type) importType.value = type;
    mode.value = "file";
    visible.value = true;
  },
  markSaved: (key: string) => {
    saved.value.add(key);
    updatePending();
    const next = problems.value.find((p) => !saved.value.has(p.key));
    if (next) {
      selectedKey.value = next.key;
      visible.value = true;
    }
  },
});
</script>
<style scoped>
.intro,
.muted,
.list-caption {
  color: var(--muted);
  line-height: 1.7;
}
.intro {
  margin: 0 0 18px;
}
.document-picker {
  position: relative;
  display: grid;
  gap: 10px;
  padding: 46px 20px;
  text-align: center;
  border: 1px dashed var(--border);
  border-radius: 8px;
  color: var(--accent);
  font-weight: 600;
  cursor: pointer;
}
.document-picker span {
  font-weight: 400;
  color: var(--muted);
  font-size: 13px;
}
.document-picker input,
.file-button input {
  position: absolute;
  inset: 0;
  opacity: 0;
  width: 100%;
  height: 100%;
  cursor: pointer;
}
.document-picker:focus-within,
.file-button:focus-within {
  outline: 2px solid var(--accent);
  outline-offset: 3px;
}
.parse-button {
  margin-top: 12px;
}
.import-layout {
  display: grid;
  grid-template-columns: 230px minmax(0, 1fr);
  gap: 24px;
}
.import-list {
  border-right: 1px solid var(--border);
  padding-right: 16px;
  max-height: 65vh;
  overflow: auto;
}
.list-caption {
  font-size: 12px;
  overflow-wrap: anywhere;
}
.import-list button {
  display: flex;
  gap: 12px;
  width: 100%;
  border: 0;
  background: transparent;
  text-align: left;
  padding: 13px 10px;
  cursor: pointer;
  border-radius: 6px;
  color: var(--text);
}
.import-list button > span {
  color: var(--muted);
  font-size: 12px;
  padding-top: 3px;
}
.import-list button strong {
  font-size: 14px;
  line-height: 1.5;
}
.import-list button small {
  display: block;
  color: var(--muted);
  margin-top: 6px;
}
.import-list button.selected {
  background: var(--surface-strong);
  box-shadow: inset 3px 0 var(--accent);
}
.import-detail {
  min-width: 0;
  max-height: 65vh;
  overflow: auto;
  padding-right: 4px;
}
.import-detail h2 {
  margin: 0 0 14px;
  font-size: 20px;
}
.warnings {
  padding: 12px 12px 12px 30px;
  background: var(--surface-strong);
  font-size: 13px;
  line-height: 1.8;
  color: var(--muted);
}
.resource-section {
  padding: 10px 0;
  border-block: 1px solid var(--border);
  margin: 18px 0;
}
.resource-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}
.resource-heading h3 {
  font-size: 14px;
}
.file-button {
  position: relative;
  display: inline-flex;
  border: 1px solid var(--border);
  border-radius: 5px;
  padding: 7px 10px;
  white-space: nowrap;
  cursor: pointer;
  font-size: 12px;
}
.resource-row {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
  margin: 10px 0;
}
.resource-status {
  flex-shrink: 0;
  color: var(--color-success);
  background: var(--color-success-soft);
  padding: 3px 7px;
  border-radius: 4px;
  font-size: 12px;
}
.resource-path {
  flex: 1 1 160px;
  min-width: 0;
  overflow-wrap: anywhere;
  font-size: 12px;
}
.resource-section .muted {
  font-size: 12px;
}
.parsed-preview {
  padding: 0 4px 16px;
  overflow-wrap: anywhere;
}
.import-action {
  position: sticky;
  bottom: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  padding: 14px 0;
  background: var(--surface-strong);
  border-top: 1px solid var(--border);
}
.import-action .muted {
  font-size: 12px;
}
.parse-progress {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
  margin: 18px 0;
}
.parse-progress .el-progress {
  flex: 1;
  min-width: 140px;
}
.parse-progress > span {
  font-size: 13px;
  color: var(--muted);
}
.import-error {
  margin: 16px 0;
}
.import-rules {
  border-top: 1px solid var(--border);
  padding-top: 14px;
  margin-top: 20px;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.8;
}
.import-rules summary {
  cursor: pointer;
  color: var(--text);
}
@media (max-width: 700px) {
  .import-layout {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  .import-list {
    max-height: 170px;
    border-right: 0;
    border-bottom: 1px solid var(--border);
    padding: 0 0 12px;
  }
  .import-detail {
    max-height: none;
  }
  .import-action {
    position: static;
  }
  .document-picker {
    padding: 32px 14px;
  }
}
</style>
