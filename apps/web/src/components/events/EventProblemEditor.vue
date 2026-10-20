<template>
  <el-form
    label-position="top"
    class="problem-editor"
    :disabled="busy"
    @submit.prevent="save"
  >
    <el-alert
      type="info"
      :closable="false"
      title="题目仅属于本场，完整题面、测试点和附件将随整场送审。"
    />
    <el-alert
      v-if="initialDraft"
      type="warning"
      :closable="false"
      title="题面已导入。请校对样例、数据范围与图片，再添加测试点并保存。"
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
    <div class="statement-workspace">
      <section class="statement-fields">
        <div class="section-title">
          <h3>规范题面</h3>
          <el-button
            v-if="legacyMode"
            link
            @click="
              legacyMode = false;
              sections.description = form.statement;
            "
            >按新规范整理</el-button
          >
        </div>
        <el-form-item v-if="legacyMode" label="原题面（保留原格式）" required
          ><el-input
            v-model="form.statement"
            type="textarea"
            :rows="15"
            maxlength="200000"
        /></el-form-item>
        <template v-else>
          <el-form-item label="题目描述" required
            ><el-input
              v-model="sections.description"
              type="textarea"
              :rows="6"
              placeholder="描述任务、背景与约束条件，支持 Markdown 和公式"
              maxlength="100000"
          /></el-form-item>
          <el-form-item label="输入格式" required
            ><el-input
              v-model="sections.input_format"
              type="textarea"
              :rows="3"
              placeholder="说明每行输入及数据范围"
              maxlength="30000"
          /></el-form-item>
          <el-form-item label="输出格式" required
            ><el-input
              v-model="sections.output_format"
              type="textarea"
              :rows="3"
              placeholder="说明需要输出的内容和格式"
              maxlength="30000"
          /></el-form-item>
          <div class="section-title">
            <h3>样例</h3>
            <el-button
              link
              :disabled="sections.samples.length >= 20"
              @click="sections.samples.push({ input: '', output: '' })"
              >添加样例</el-button
            >
          </div>
          <div
            v-for="(sample, index) in sections.samples"
            :key="index"
            class="sample-pair"
          >
            <div class="section-title">
              <strong>样例 {{ index + 1 }}</strong
              ><el-button
                v-if="sections.samples.length > 1"
                link
                type="danger"
                @click="sections.samples.splice(index, 1)"
                >移除</el-button
              >
            </div>
            <div class="fields">
              <el-form-item :label="`样例 ${index + 1} 输入`"
                ><el-input
                  v-model="sample.input"
                  type="textarea"
                  :rows="3"
                  maxlength="65536" /></el-form-item
              ><el-form-item :label="`样例 ${index + 1} 输出`" required
                ><el-input
                  v-model="sample.output"
                  type="textarea"
                  :rows="3"
                  maxlength="65536"
              /></el-form-item>
            </div>
          </div>
          <el-form-item label="说明（可选）"
            ><el-input
              v-model="sections.notes"
              type="textarea"
              :rows="3"
              placeholder="样例解释或补充提示；留空时不展示本节"
              maxlength="30000"
          /></el-form-item>
        </template>
        <p class="muted">
          图片请在题面中填写 ![说明](https://图片地址)，右侧可直接预览。
        </p>
      </section>
      <aside class="statement-preview">
        <div class="section-title">
          <h3>题面预览</h3>
          <span class="muted">与发布题面一致</span>
        </div>
        <h2>{{ form.title || "未命名题目" }}</h2>
        <MarkdownRenderer
          :source="statement"
          :asset-urls="imageUrls"
          :asset-base="
            link ? eventAssets(kind, eventId, link.problem_id) : undefined
          "
          :problem-id="link?.problem_id"
        />
      </aside>
    </div>
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
    <EventTestFiles
      v-if="form.test_mode !== 'preserve'"
      v-model="tests"
      :disabled="busy"
    />
    <el-progress
      v-if="busy"
      :percentage="progress"
      :format="
        (value: number) => (value === 100 ? '正在保存题目' : `上传 ${value}%`)
      "
    />
    <el-alert v-if="error" type="error" :title="error" :closable="false" />
    <el-checkbox v-if="initialDraft" v-model="importConfirmed"
      >已对照原文核对题面、公式、图片和样例</el-checkbox
    >
    <div>
      <el-button type="primary" :loading="busy" @click="save"
        >保存题目</el-button
      ><el-button :disabled="busy" @click="$emit('cancel')">取消</el-button>
    </div>
  </el-form>
</template>
<script setup lang="ts">
import {
  computed,
  reactive,
  ref,
  onBeforeUnmount,
  onMounted,
  watch,
} from "vue";
import {
  emptyStatement,
  renderStatement,
  validateStatement,
  type StatementSections,
} from "../../features/events/statement";
import {
  uploadEventTests,
  EVENT_CHUNK_SIZE,
  testFileError,
} from "../../features/events/testUploads";
import EventTestFiles from "./EventTestFiles.vue";
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
import type { ImportedProblem } from "../../features/events/documentImport";
const props = defineProps<{
  kind: EventKind;
  eventId: number;
  revision: number;
  link?: PaperProblem;
  initialDraft?: ImportedProblem;
  nextLabel: string;
}>();
const emit = defineEmits<{ saved: []; cancel: []; dirty: [boolean] }>();
const problem = props.link?.problem;
const persisted =
  props.link?.statement_sections || props.initialDraft?.sections;
const importConfirmed = ref(false);
onMounted(() => {
  if (props.initialDraft) emit("dirty", true);
});
const sections = reactive<StatementSections>(
  persisted
    ? { ...emptyStatement(), ...JSON.parse(JSON.stringify(persisted)) }
    : emptyStatement(),
);
const legacyMode = ref(Boolean(props.link && !persisted));
const statement = computed(() =>
  legacyMode.value ? form.statement : renderStatement(sections),
);
const progress = ref(0);
let controller: AbortController | undefined;
onBeforeUnmount(() => controller?.abort());
const form = reactive({
  label: props.link?.label || props.nextLabel,
  score: props.link?.score || 100,
  title: problem?.title || props.initialDraft?.title || "",
  statement: problem?.statement || "",
  time_limit_ms:
    problem?.time_limit_ms || props.initialDraft?.time_limit_ms || 1000,
  memory_limit_mb:
    problem?.memory_limit_mb || props.initialDraft?.memory_limit_mb || 256,
  output_limit_kb: problem?.output_limit_kb || 1024,
  tags: problem ? tagList(problem.tags) : [],
  checker: {
    type: props.link?.checker.type || "exact",
    absolute_tolerance: props.link?.checker.absolute_tolerance ?? 0.000001,
    relative_tolerance: props.link?.checker.relative_tolerance ?? 0.000001,
  },
  test_mode: props.link ? "preserve" : "replace",
  assets: (props.initialDraft
    ? JSON.parse(JSON.stringify(props.initialDraft.assets))
    : []) as Array<{
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
async function save() {
  if (busy.value) return;
  error.value = "";
  if (props.initialDraft && !importConfirmed.value) {
    error.value = "请先确认已对照原文校对导入题面";
    return;
  }
  if (
    !form.title.trim() ||
    (legacyMode.value ? !form.statement.trim() : validateStatement(sections))
  ) {
    error.value = !form.title.trim()
      ? "请填写题目标题"
      : legacyMode.value
        ? "请填写题面"
        : validateStatement(sections);
    return;
  }
  if (form.test_mode !== "preserve" && !tests.value.length) {
    error.value = "请选择测试点文件";
    return;
  }
  if (form.test_mode !== "preserve" && testFileError(tests.value)) {
    error.value = testFileError(tests.value);
    return;
  }
  busy.value = true;
  progress.value = 0;
  controller = new AbortController();
  try {
    const chunked =
      form.test_mode !== "preserve" &&
      tests.value.some((file) => file.size > EVENT_CHUNK_SIZE);
    const test_uploads = chunked
      ? await uploadEventTests(
          tests.value,
          props.kind,
          props.eventId,
          props.revision,
          (value) => (progress.value = value),
          controller.signal,
        )
      : [];
    const body = new FormData();
    body.append(
      "draft",
      JSON.stringify({
        ...form,
        statement: statement.value,
        statement_sections: legacyMode.value ? undefined : sections,
        test_uploads,
        revision: props.revision,
        expected_version_id: props.link?.problem_version_id,
      }),
    );
    if (form.test_mode !== "preserve" && !chunked)
      tests.value.forEach((file) => body.append("tests", file));
    const options = {
      path: eventPath(props.kind, props.eventId),
      body,
      timeout: 180000,
      signal: controller.signal,
      onUploadProgress(event: any) {
        if (!chunked)
          progress.value = Math.min(
            100,
            Math.round((event.loaded / (event.total || event.loaded)) * 100),
          );
      },
    };
    if (props.link)
      await generatedApi.updateEventProblem({
        ...options,
        path: { ...options.path, problem_id: props.link.problem_id },
      });
    else await generatedApi.createEventProblem(options);
    emit("dirty", false);
    emit("saved");
  } catch (e) {
    controller?.abort();
    error.value = apiErrorMessage(e);
  } finally {
    busy.value = false;
  }
}
watch([form, sections, tests], () => emit("dirty", true), { deep: true });
</script>
<style scoped>
.statement-workspace {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 28px;
  align-items: start;
}
.statement-fields,
.statement-preview {
  min-width: 0;
}
.statement-preview {
  position: sticky;
  top: 20px;
  max-height: 80vh;
  overflow: auto;
  padding: 0 20px 20px;
  border-left: 1px solid var(--border);
}
.section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.section-title h3 {
  margin: 8px 0 14px;
  font-size: 15px;
}
.sample-pair {
  padding: 10px 0;
  border-top: 1px solid var(--border);
}
.sample-pair .fields > * {
  flex: 1;
  min-width: 0;
}
@media (max-width: 1000px) {
  .statement-workspace {
    grid-template-columns: 1fr;
  }
  .statement-preview {
    position: static;
    max-height: 460px;
    border: 1px solid var(--border);
    padding: 15px;
  }
}

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
