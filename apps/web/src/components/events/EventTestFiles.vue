<template>
  <section class="test-files" aria-label="测试点文件">
    <label class="drop-zone" @dragover.prevent @drop.prevent="drop">
      <el-icon :size="24"><UploadFilled /></el-icon
      ><strong>拖入测试点，或点击选择文件</strong>
      <span>.in 与 .out 配对，或上传 ZIP · 总计不超过 128 MiB</span>
      <input
        type="file"
        multiple
        accept=".in,.out,.zip"
        :disabled="disabled"
        aria-label="选择测试点文件"
        @change="pick"
      />
    </label>
    <div class="file-summary">
      <span
        >{{ modelValue.length }} 个文件 ·
        {{ formatFileSize(totalTestFileSize(modelValue)) }}</span
      ><el-button
        v-if="modelValue.length"
        link
        :disabled="disabled"
        @click="clear"
        >清空</el-button
      >
    </div>
    <el-alert
      v-if="validation"
      type="warning"
      :title="validation"
      :closable="false"
    />
    <ul v-if="modelValue.length" class="files">
      <li v-for="file in visible" :key="file.name">
        <div>
          <strong>{{ file.name }}</strong
          ><small>{{ formatFileSize(file.size) }}</small>
        </div>
        <div>
          <el-button
            link
            :disabled="/\.zip$/i.test(file.name)"
            @click="preview(file)"
            >预览</el-button
          ><el-button
            link
            type="danger"
            :disabled="disabled"
            @click="remove(file)"
            >移除</el-button
          >
        </div>
      </li>
    </ul>
    <el-pagination
      v-if="modelValue.length > 8"
      v-model:current-page="page"
      :page-size="8"
      :total="modelValue.length"
      layout="prev, pager, next"
      :pager-count="5"
    />
    <p class="hint">
      预览仅读取前 64 KiB；超过 1 MiB 的文件分片上传，并显示真实进度。ZIP
      在服务端检查配对和解压大小。
    </p>
    <el-dialog
      v-model="previewOpen"
      :title="previewName"
      width="min(760px, calc(100vw - 24px))"
      append-to-body
      ><p v-if="truncated" class="hint">
        仅显示前 64 KiB，完整文件将原样上传。
      </p>
      <pre class="file-preview" v-text="previewText" />
    </el-dialog>
  </section>
</template>
<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { UploadFilled } from "@element-plus/icons-vue";
import {
  formatFileSize,
  totalTestFileSize,
} from "../../features/problems/testFileUpload";
import {
  testFileError,
  previewTestFile,
} from "../../features/events/testUploads";
const props = defineProps<{ modelValue: File[]; disabled?: boolean }>(),
  emit = defineEmits<{ "update:modelValue": [File[]] }>();
const page = ref(1),
  previewOpen = ref(false),
  previewName = ref(""),
  previewText = ref(""),
  truncated = ref(false),
  readError = ref("");
const visible = computed(() =>
    props.modelValue.slice((page.value - 1) * 8, page.value * 8),
  ),
  validation = computed(
    () => readError.value || testFileError(props.modelValue),
  );
function add(files: File[]) {
  if (props.disabled) return;
  readError.value = "";
  const incoming = [...props.modelValue, ...files];
  if (
    incoming.length > 400 ||
    totalTestFileSize(incoming) > 128 * 1024 * 1024
  ) {
    readError.value = "最多 400 个文件，总计 128 MiB";
    return;
  }
  emit("update:modelValue", incoming);
}
function pick(e: Event) {
  const input = e.target as HTMLInputElement;
  add(Array.from(input.files || []));
  input.value = "";
}
function drop(e: DragEvent) {
  add(Array.from(e.dataTransfer?.files || []));
}
function clear() {
  emit("update:modelValue", []);
  page.value = 1;
  readError.value = "";
}
function remove(file: File) {
  emit(
    "update:modelValue",
    props.modelValue.filter((x) => x !== file),
  );
}
let sequence = 0;
async function preview(file: File) {
  const current = ++sequence;
  try {
    const result = await previewTestFile(file);
    if (current !== sequence) return;
    previewName.value = file.name;
    previewText.value = result.text;
    truncated.value = result.truncated;
    previewOpen.value = true;
  } catch {
    readError.value = "文件无法读取，请重新选择";
  }
}
watch(
  () => props.modelValue.length,
  (n) => {
    page.value = Math.min(page.value, Math.max(1, Math.ceil(n / 8)));
  },
);
</script>
<style scoped>
.test-files {
  display: grid;
  gap: 12px;
  min-width: 0;
}
.drop-zone {
  position: relative;
  display: grid;
  justify-items: center;
  gap: 7px;
  text-align: center;
  padding: 24px 12px;
  border: 1px dashed var(--border);
  border-radius: 10px;
  background: var(--surface-strong);
  cursor: pointer;
}
.drop-zone:hover,
.drop-zone:focus-within {
  border-color: var(--accent);
}
.drop-zone input {
  position: absolute;
  inset: 0;
  opacity: 0;
  width: 100%;
  height: 100%;
  cursor: pointer;
}
.drop-zone span,
.hint,
small {
  color: var(--muted);
  font-size: 12px;
  line-height: 1.6;
}
.file-summary,
.files li,
.files li > div {
  display: flex;
  align-items: center;
  gap: 12px;
  justify-content: space-between;
}
.files {
  list-style: none;
  padding: 0;
  margin: 0;
}
.files li {
  padding: 9px 0;
  border-bottom: 1px solid var(--border);
}
.files li > div:first-child {
  min-width: 0;
  align-items: flex-start;
  flex-direction: column;
  gap: 3px;
}
.files strong {
  overflow-wrap: anywhere;
  font-size: 13px;
}
.hint {
  margin: 0;
}
.file-preview {
  max-height: 55vh;
  overflow: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-size: 12px;
}
</style>
