<template>
  <el-form label-position="top" class="problem-editor" @submit.prevent="save">
    <el-alert
      type="info"
      :closable="false"
      title="题目仅属于本场，完整题面、测试点和附件将随整场送审。"
    />
    <div class="fields">
      <el-form-item label="题号"
        ><el-input v-model="form.label" maxlength="16" /></el-form-item
      ><el-form-item v-if="kind === 'exam'" label="分值"
        ><el-input-number v-model="form.score" :min="1" :max="1000"
      /></el-form-item>
    </div>
    <el-form-item label="题目标题" required
      ><el-input v-model="form.title" maxlength="200"
    /></el-form-item>
    <el-form-item label="题面（Markdown）" required
      ><el-input
        v-model="form.statement"
        type="textarea"
        :rows="14"
        maxlength="200000"
        placeholder="完整填写题目描述、输入输出、样例及说明"
    /></el-form-item>
    <label
      >添加题面图片<input
        type="file"
        accept="image/png,image/jpeg,image/gif,image/webp"
        multiple
        @change="addImages"
    /></label>
    <el-collapse
      ><el-collapse-item title="预览题面" name="preview"
        ><MarkdownRenderer
          :source="form.statement"
          :asset-urls="imageUrls"
          :asset-base="
            link ? eventAssets(kind, eventId, link.problem_id) : undefined
          "
          :problem-id="link?.problem_id" /></el-collapse-item
    ></el-collapse>
    <div class="fields">
      <el-form-item label="时间限制（ms）"
        ><el-input-number
          v-model="form.time_limit_ms"
          :min="1"
          :max="60000" /></el-form-item
      ><el-form-item label="内存（MiB）"
        ><el-input-number
          v-model="form.memory_limit_mb"
          :min="1"
          :max="4096" /></el-form-item
      ><el-form-item label="输出（KiB）"
        ><el-input-number v-model="form.output_limit_kb" :min="1" :max="65536"
      /></el-form-item>
    </div>
    <el-form-item label="标签"
      ><el-select
        v-model="form.tags"
        multiple
        filterable
        allow-create
        default-first-option
        placeholder="输入标签后回车"
    /></el-form-item>
    <div class="fields">
      <el-form-item label="比较方式"
        ><el-select v-model="form.checker.type"
          ><el-option value="exact" label="精确匹配" /><el-option
            value="tokens"
            label="忽略空白" /><el-option
            value="float"
            label="浮点误差" /></el-select></el-form-item
      ><template v-if="form.checker.type === 'float'"
        ><el-form-item label="绝对误差"
          ><el-input-number
            v-model="form.checker.absolute_tolerance"
            :min="0"
            :max="1"
            :step="0.000001" /></el-form-item
        ><el-form-item label="相对误差"
          ><el-input-number
            v-model="form.checker.relative_tolerance"
            :min="0"
            :max="1"
            :step="0.000001" /></el-form-item
      ></template>
    </div>
    <el-form-item label="测试点处理"
      ><el-select v-model="form.test_mode"
        ><el-option
          v-if="link"
          value="preserve"
          label="保留原有全部测试点" /><el-option
          v-if="link"
          value="merge"
          label="按名称合并，保留未上传的测试点及原权重" /><el-option
          value="replace"
          label="完整替换为本次上传的测试点" /></el-select
    ></el-form-item>
    <el-alert
      v-if="link && form.test_mode === 'replace'"
      type="warning"
      :closable="false"
      title="完整替换会移除本题新版本中所有未上传的测试点。旧版本仍会保留。"
    />
    <label v-if="form.test_mode !== 'preserve'"
      >上传 .in / .out 文件或 ZIP<input
        type="file"
        multiple
        accept=".in,.out,.zip"
        @change="pickTests"
      /><span>{{ tests.length }} 个文件</span></label
    >
    <p class="muted">
      同名输入输出文件须成对提供。合并按原测试点名称或包内路径匹配；可先下载题包核对名称。
    </p>
    <el-alert v-if="error" type="error" :title="error" :closable="false" />
    <div>
      <el-button type="primary" :loading="busy" @click="save"
        >保存题目</el-button
      ><el-button :disabled="busy" @click="$emit('cancel')">取消</el-button>
    </div>
  </el-form>
</template>
<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { generatedApi } from "../../api/client";
import { apiErrorMessage } from "../../api/errors";
import {
  eventAssets,
  eventPath,
  type EventKind,
  type PaperProblem,
} from "../../api/modules/events";
import { tagList } from "../../features/problems/problemMeta";
import MarkdownRenderer from "../MarkdownRenderer.vue";
const props = defineProps<{
  kind: EventKind;
  eventId: number;
  revision: number;
  link?: PaperProblem;
  nextLabel: string;
}>();
const emit = defineEmits<{ saved: []; cancel: [] }>();
const problem = props.link?.problem;
const form = reactive({
  label: props.link?.label || props.nextLabel,
  score: props.link?.score || 100,
  title: problem?.title || "",
  statement: problem?.statement || "",
  time_limit_ms: problem?.time_limit_ms || 1000,
  memory_limit_mb: problem?.memory_limit_mb || 256,
  output_limit_kb: problem?.output_limit_kb || 1024,
  tags: problem ? tagList(problem.tags) : [],
  checker: {
    type: props.link?.checker.type || "exact",
    absolute_tolerance: props.link?.checker.absolute_tolerance ?? 0.000001,
    relative_tolerance: props.link?.checker.relative_tolerance ?? 0.000001,
  },
  test_mode: props.link ? "preserve" : "replace",
  assets: [] as Array<{
    name: string;
    path: string;
    content_type: string;
    data: string;
  }>,
});
const tests = ref<File[]>([]),
  error = ref(""),
  busy = ref(false);
const imageUrls = computed(() =>
  Object.fromEntries(form.assets.map((a) => [a.path, a.data])),
);
function pickTests(e: Event) {
  tests.value = Array.from((e.target as HTMLInputElement).files || []);
}
async function addImages(e: Event) {
  const input = e.target as HTMLInputElement,
    files = Array.from(input.files || []);
  input.value = "";
  for (const file of files) {
    if (file.size > 5 * 1024 * 1024) {
      error.value = "单张题面图片不能超过 5 MiB";
      return;
    }
    const data = await new Promise<string>((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(String(reader.result));
      reader.onerror = reject;
      reader.readAsDataURL(file);
    });
    const path = `assets/${crypto.randomUUID()}.${file.name.split(".").pop()?.toLowerCase()}`;
    form.assets.push({ name: file.name, path, content_type: file.type, data });
    form.statement += `\n\n![题面图片](${path})\n`;
  }
}
async function save() {
  if (busy.value) return;
  error.value = "";
  if (!form.title.trim() || !form.statement.trim()) {
    error.value = "请填写标题和完整题面";
    return;
  }
  if (form.test_mode !== "preserve" && !tests.value.length) {
    error.value = "请选择测试点文件";
    return;
  }
  busy.value = true;
  try {
    const body = new FormData();
    body.append(
      "draft",
      JSON.stringify({
        ...form,
        revision: props.revision,
        expected_version_id: props.link?.problem_version_id,
      }),
    );
    if (form.test_mode !== "preserve")
      tests.value.forEach((file) => body.append("tests", file));
    const options = {
      path: eventPath(props.kind, props.eventId),
      body,
      timeout: 180000,
    };
    if (props.link)
      await generatedApi.updateEventProblem({
        ...options,
        path: { ...options.path, problem_id: props.link.problem_id },
      });
    else await generatedApi.createEventProblem(options);
    emit("saved");
  } catch (e) {
    error.value = apiErrorMessage(e);
  } finally {
    busy.value = false;
  }
}
</script>
<style scoped>
.problem-editor {
  display: grid;
  gap: 14px;
}
.fields {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}
.fields .el-select {
  min-width: 190px;
}
input[type="file"] {
  display: block;
  margin: 8px 0;
  max-width: 100%;
}
.muted {
  color: var(--muted);
  line-height: 1.7;
  margin: 0;
}
</style>
