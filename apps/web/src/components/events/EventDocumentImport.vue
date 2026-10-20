<template>
  <el-dialog
    v-model="visible"
    title="导入 Markdown 单题"
    width="min(1040px, calc(100vw - 24px))"
    destroy-on-close
  >
    <p class="intro">
      从一份题面开始，预览后继续编辑。每次导入一道题，测试点在编辑页添加。
    </p>
    <el-alert v-if="error" type="error" :title="error" :closable="false" />
    <div v-if="!draft" class="import-layout">
      <section>
        <el-segmented
          v-model="mode"
          :options="[
            { label: '选择文档', value: 'file' },
            { label: '粘贴文本', value: 'paste' },
          ]"
          aria-label="导入方式"
        />
        <label v-if="mode === 'file'" class="document-picker">
          <strong>{{ busy ? "正在读取题面…" : "选择 Markdown 文档" }}</strong>
          <span>.md / .markdown · UTF-8 · 最大 200 KB</span>
          <input
            type="file"
            accept=".md,.markdown,text/markdown"
            :disabled="busy"
            aria-label="选择 Markdown 文档"
            @change="pickDocument"
          />
        </label>
        <template v-else>
          <el-input
            v-model="source"
            type="textarea"
            :rows="14"
            placeholder="# 题目标题"
            aria-label="Markdown 题面"
          />
          <el-button
            type="primary"
            :disabled="!source.trim()"
            @click="parse(source)"
            >解析并预览</el-button
          >
        </template>
      </section>
      <aside class="rules">
        <h3>题面格式</h3>
        <ol>
          <li>一个一级标题：题目名称</li>
          <li>二级标题：题目描述、输入格式、输出格式</li>
          <li>样例输入、样例输出使用代码块</li>
          <li>说明 / 提示可选，公式原样保留</li>
        </ol>
        <h3>远程图片</h3>
        <p>
          使用
          <code>![说明](https://…)</code
          >。图片链接与独立图片网址也可直接预览，无需上传本地图片。
        </p>
        <p class="muted">
          图片由浏览器从原网址加载；请使用稳定且允许外链的 HTTPS 地址。
        </p>
      </aside>
    </div>
    <template v-else>
      <div class="preview-heading">
        <div>
          <h3>{{ draft.title }}</h3>
          <span class="muted"
            >{{ fileName }} · {{ draft.sections.samples.length }} 组样例</span
          >
        </div>
        <el-button @click="reset">重新导入</el-button>
      </div>
      <el-alert
        v-for="warning in draft.warnings"
        :key="warning"
        :title="warning"
        type="warning"
        :closable="false"
      />
      <div class="import-preview">
        <MarkdownRenderer :source="renderStatement(draft.sections)" />
      </div>
    </template>
    <template #footer
      ><el-button @click="visible = false">关闭</el-button
      ><el-button v-if="draft" type="primary" @click="choose"
        >使用此题面</el-button
      ></template
    >
  </el-dialog>
</template>
<script setup lang="ts">
import { ref } from "vue";
import { ElMessageBox } from "element-plus";
import MarkdownRenderer from "../MarkdownRenderer.vue";
import {
  importEventMarkdown,
  type ImportedProblem,
} from "../../features/events/documentImport";
import { renderStatement } from "../../features/events/statement";

const emit = defineEmits<{ select: [ImportedProblem]; pending: [boolean] }>();
const visible = ref(false),
  mode = ref("file"),
  source = ref(""),
  error = ref(""),
  busy = ref(false);
const draft = ref<ImportedProblem>(),
  fileName = ref("粘贴的 Markdown");
function parse(text: string) {
  error.value = "";
  try {
    draft.value = importEventMarkdown(text);
    emit("pending", true);
  } catch (e) {
    error.value = (e as Error).message;
  }
}
async function pickDocument(event: Event) {
  const input = event.target as HTMLInputElement,
    file = input.files?.[0];
  input.value = "";
  if (!file) return;
  busy.value = true;
  error.value = "";
  try {
    if (!/\.(md|markdown)$/i.test(file.name))
      throw new Error("请选择单题 .md 或 .markdown 文档");
    if (file.size > 200000)
      throw new Error("Markdown 文档不能超过 200000 字节");
    const text = new TextDecoder("utf-8", { fatal: true }).decode(
      await file.arrayBuffer(),
    );
    fileName.value = file.name;
    parse(text);
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}
async function reset() {
  try {
    await ElMessageBox.confirm(
      "尚未保存的导入题面将被清空，继续？",
      "重新导入",
      { confirmButtonText: "重新导入", cancelButtonText: "保留题面" },
    );
  } catch {
    return;
  }
  draft.value = undefined;
  source.value = "";
  error.value = "";
  fileName.value = "粘贴的 Markdown";
  emit("pending", false);
}
function choose() {
  if (!draft.value) return;
  emit("select", JSON.parse(JSON.stringify(draft.value)));
  visible.value = false;
}
defineExpose({
  open: () => {
    visible.value = true;
  },
  markSaved: (key: string) => {
    if (draft.value?.key === key) {
      draft.value = undefined;
      source.value = "";
      emit("pending", false);
    }
  },
});
</script>
<style scoped>
.intro,
.muted {
  color: var(--muted);
  line-height: 1.7;
}
.intro {
  margin: 0 0 20px;
}
.import-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(260px, 1fr);
  gap: 28px;
}
.import-layout section {
  display: grid;
  align-content: start;
  gap: 16px;
  min-width: 0;
}
.document-picker {
  position: relative;
  display: grid;
  gap: 12px;
  padding: 64px 20px;
  text-align: center;
  border: 1px dashed var(--border);
  border-radius: 8px;
  background: var(--surface);
  cursor: pointer;
}
.document-picker:hover,
.document-picker:focus-within {
  border-color: var(--accent);
}
.document-picker strong {
  color: var(--accent);
}
.document-picker span {
  color: var(--muted);
}
.document-picker input {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  opacity: 0;
  cursor: pointer;
}
.rules {
  border-left: 1px solid var(--border);
  padding-left: 24px;
  line-height: 1.8;
}
.rules h3 {
  margin: 0 0 8px;
  font-size: 15px;
}
.rules ol {
  padding-left: 20px;
  margin: 0 0 20px;
}
.rules code {
  overflow-wrap: anywhere;
}
.preview-heading {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
  margin: 12px 0 20px;
}
.preview-heading h3 {
  margin: 0 0 6px;
}
.import-preview {
  max-height: 58vh;
  overflow: auto;
  padding: 20px;
  border: 1px solid var(--border);
  border-radius: 8px;
  margin-top: 16px;
}
@media (max-width: 700px) {
  .import-layout {
    grid-template-columns: 1fr;
    gap: 20px;
  }
  .rules {
    border-left: 0;
    border-top: 1px solid var(--border);
    padding: 18px 0 0;
  }
  .document-picker {
    padding: 40px 12px;
  }
}
</style>
