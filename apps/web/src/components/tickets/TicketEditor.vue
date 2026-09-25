<template>
  <el-form label-position="top" class="ticket-editor" @submit.prevent="submit">
    <div class="editor-fields">
      <el-form-item label="工单类型" required
        ><el-select
          v-model="form.action"
          :disabled="Boolean(ticket)"
          aria-label="工单类型"
          ><el-option
            v-for="action in actions"
            :key="action"
            :value="action"
            :label="ticketActionLabels[action]" /></el-select
      ></el-form-item>
      <el-form-item v-if="form.action === 'create'" label="入库范围"
        ><el-select
          v-model="form.target_scope"
          :disabled="Boolean(ticket)"
          aria-label="入库范围"
          ><el-option value="public" label="公共题库" /><el-option
            value="prepared"
            label="预备题库" /><el-option
            value="team_problem_set"
            label="团队题单" /></el-select
      ></el-form-item>
      <el-form-item
        v-if="
          form.action === 'create' && form.target_scope === 'team_problem_set'
        "
        label="团队题单 ID"
        required
        ><el-input-number
          v-model="form.team_problem_set_id"
          :min="1"
          :disabled="Boolean(ticket)"
      /></el-form-item>
    </div>
    <el-form-item label="修改说明" required
      ><el-input
        v-model="form.description"
        type="textarea"
        :rows="4"
        maxlength="10000"
        show-word-limit
        placeholder="请具体说明问题、修改理由或建议（至少 4 个字）"
    /></el-form-item>
    <el-form-item label="附件（可选）">
      <div class="attachments-input">
        <p class="muted" role="status">
          已选择 {{ retained.length + files.length }} / {{ attachmentLimit }} 个附件（含保留附件）
        </p>
        <el-upload
          :auto-upload="false"
          multiple
          :limit="Math.max(0, attachmentLimit - retained.length)"
          :disabled="retained.length >= attachmentLimit"
          :file-list="uploads"
          :on-change="changeFiles"
          :on-remove="changeFiles"
          :on-exceed="tooManyFiles"
          ><el-button :disabled="retained.length >= attachmentLimit">选择附件</el-button
          ><template #tip
            ><div class="muted">
              支持测试点文件、ZIP 和图片；最多
              {{ attachmentLimit }} 个（含保留附件），单个 32 MiB，总计 96
              MiB。附件仅申请人和有权管理者可见。
            </div></template
          ></el-upload
        >
        <div v-for="file in retained" :key="file.id" class="retained-file">
          <span>{{ file.name }}</span
          ><el-button link type="danger" @click="removeRetained(file.id)"
            >移除</el-button
          >
        </div>
      </div>
    </el-form-item>
    <template v-if="form.action === 'create'">
      <el-form-item label="题目标题" required
        ><el-input
          v-model="form.draft_title"
          maxlength="200"
          placeholder="填写新题目标题"
      /></el-form-item>
      <div class="preview-heading">
        <h3>题面编辑与预览</h3>
        <el-radio-group v-model="mobileTab" class="mobile-tabs"
          ><el-radio-button value="edit">编辑</el-radio-button
          ><el-radio-button value="preview"
            >预览</el-radio-button
          ></el-radio-group
        >
      </div>
      <div v-if="imageFiles.length" class="image-insert">
        <span>插入附件图片：</span
        ><el-button
          v-for="file in imageFiles"
          :key="file.name"
          size="small"
          @click="insertImage(file.name)"
          >{{ file.name }}</el-button
        >
      </div>
      <div class="markdown-grid" :class="`mobile-${mobileTab}`">
        <section class="markdown-source">
          <ProblemStatementEditor
            ref="statementEditor"
            v-model="form.draft_statement"
          />
        </section>
        <section class="markdown-preview" aria-label="新题目实时预览">
          <h2>{{ form.draft_title || "题目预览" }}</h2>
          <MarkdownRenderer
            :source="form.draft_statement"
            :asset-urls="imageUrls"
          />
        </section>
      </div>
    </template>
    <div class="editor-actions">
      <el-button @click="$emit('cancel')">取消</el-button
      ><el-button type="primary" native-type="submit" :loading="saving">{{
        ticket ? "保存修改" : "提交工单"
      }}</el-button>
    </div>
  </el-form>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from "vue";
import { ElMessage, type UploadFile } from "element-plus";
import ProblemStatementEditor from "./ProblemStatementEditor.vue";
import MarkdownRenderer from "../MarkdownRenderer.vue";
import type {
  ProblemChangeAction,
  ProblemChangeTicket,
} from "../../api/client";
import { ticketRequest, type TicketContext } from "../../api/modules/tickets";
import {
  ticketActionLabels,
  ticketAttachmentLimit,
  validateProblemChangeTicketDraft,
} from "../../features/problems/changeTicket";
import { useAuthStore } from "../../stores/auth";
const props = defineProps<{
  context: TicketContext;
  ticket?: ProblemChangeTicket;
}>();
const emit = defineEmits<{ saved: [ProblemChangeTicket]; cancel: [] }>();
const auth = useAuthStore(),
  saving = ref(false),
  uploads = ref<UploadFile[]>([]),
  mobileTab = ref("edit");
const retained = ref([...(props.ticket?.attachments || [])]);
const form = reactive({
  action: (props.ticket?.action ||
    (props.context.kind === "mine"
      ? "create"
      : "replace")) as ProblemChangeAction,
  description: props.ticket?.description || "",
  target_scope:
    props.ticket?.target_scope || props.context.targetScope || "public",
  team_problem_set_id:
    props.ticket?.team_problem_set_id ||
    props.context.teamProblemSetID ||
    undefined,
  draft_title: props.ticket?.draft_title || "",
  draft_statement: props.ticket?.draft_statement || "",
});
const statementEditor = ref<InstanceType<typeof ProblemStatementEditor>>();
const attachmentLimit = computed(() => ticketAttachmentLimit(form.action));
const actions = computed<ProblemChangeAction[]>(() => {
  if (props.ticket) return [props.ticket.action];
  if (props.context.kind === "mine") return ["create"];
  return [
    "replace",
    "suggest",
    ...(auth.user?.can_author || auth.role === "admin"
      ? props.context.kind === "problem" || props.context.problemID
        ? ["archive" as const]
        : []
      : []),
  ];
});
const files = computed(() =>
  uploads.value.flatMap((file) => (file.raw ? [file.raw] : [])),
);
const imageFiles = computed(() => [
  ...retained.value.filter((file) => file.content_type.startsWith("image/")),
  ...files.value.filter((file) => /^image\/(png|jpeg|gif)$/.test(file.type)),
]);
const imageUrls = ref<Record<string, string>>({});
const imageKey = (name: string) => `attachments/${encodeURIComponent(name)}`;
let imageGeneration = 0;
watch(
  [files, retained],
  async (_value, _previous, onCleanup) => {
    const generation = ++imageGeneration,
      urls: Record<string, string> = {},
      controller = new AbortController();
    onCleanup(() => {
      controller.abort();
      Object.values(urls).forEach(URL.revokeObjectURL);
    });
    imageUrls.value = {};
    for (const file of files.value)
      if (/^image\/(png|jpeg|gif)$/.test(file.type))
        urls[imageKey(file.name)] = URL.createObjectURL(file);
    imageUrls.value = { ...urls };
    if (props.ticket)
      for (const file of retained.value.filter((file) =>
        file.content_type.startsWith("image/"),
      )) {
        try {
          const { data } = await ticketRequest<Blob>(
            props.context,
            "attachment",
            props.ticket.id,
            {
              path: { attachment_id: file.id },
              query: { preview: "1" },
              responseType: "blob",
              signal: controller.signal,
            },
          );
          if (!controller.signal.aborted)
            urls[imageKey(file.name)] = URL.createObjectURL(data);
        } catch {
          /* Missing private images must not fall back to a public URL. */
        }
      }
    if (generation === imageGeneration && !controller.signal.aborted)
      imageUrls.value = { ...urls };
  },
  { immediate: true },
);
onBeforeUnmount(() => {
  imageGeneration++;
});
function changeFiles(_file: UploadFile, next: UploadFile[]) {
  uploads.value = next;
}
function tooManyFiles() {
  ElMessage.warning(`最多上传 ${attachmentLimit.value} 个附件（含保留附件）`);
}
function removeRetained(id: number) {
  retained.value = retained.value.filter((file) => file.id !== id);
}
function insertImage(name: string) {
  statementEditor.value?.insertImage(`\n\n![附件图片](${imageKey(name)})\n`);
}
async function submit() {
  if (saving.value) return;
  if (form.action === "create") {
    const statementError = statementEditor.value?.validate();
    if (statementError) return void ElMessage.warning(statementError);
  }
  const problemID =
    props.ticket?.problem_id ||
    (props.context.kind === "problem"
      ? props.context.id
      : props.context.problemID);
  const error = validateProblemChangeTicketDraft({
    action: form.action,
    problemID,
    eventID: ["exam", "contest"].includes(props.context.kind)
      ? props.context.id
      : props.ticket?.exam_id || props.ticket?.contest_id || undefined,
    targetScope: form.target_scope,
    teamProblemSetID: form.team_problem_set_id,
    description: form.description,
    attachments: files.value,
    retainedCount: retained.value.length,
    retainedSize: retained.value.reduce((sum, file) => sum + file.size, 0),
    draftTitle: form.draft_title,
    draftStatement: form.draft_statement,
  });
  if (error) return void ElMessage.warning(error);
  if (
    new Set([...retained.value, ...files.value].map((file) => file.name))
      .size !==
    retained.value.length + files.value.length
  )
    return void ElMessage.warning("请为附件使用不同文件名");
  saving.value = true;
  try {
    const data = new FormData();
    data.append(
      "draft",
      JSON.stringify({
        ...form,
        problem_id: problemID,
        revision: props.ticket?.revision,
        retain_attachment_ids: retained.value.map((file) => file.id),
      }),
    );
    for (const file of files.value) data.append("attachments", file);
    const response = await ticketRequest(
      props.context,
      props.ticket ? "update" : "create",
      props.ticket?.id,
      { body: data },
    );
    ElMessage.success(props.ticket ? "工单已更新" : "工单已提交");
    emit("saved", response.data);
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || "工单保存失败");
  } finally {
    saving.value = false;
  }
}
</script>
<style scoped>
.ticket-editor {
  min-width: 0;
}
.editor-fields {
  display: flex;
  gap: 18px;
  flex-wrap: wrap;
}
.editor-fields .el-form-item {
  min-width: 180px;
}
.attachments-input {
  width: 100%;
}
.muted {
  color: var(--muted);
  margin-top: 8px;
  line-height: 1.6;
}
.retained-file,
.preview-heading {
  display: flex;
  justify-content: space-between;
  gap: 14px;
  align-items: center;
}
.retained-file span {
  overflow-wrap: anywhere;
}
.preview-heading h3 {
  margin: 0 0 12px;
}
.markdown-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 16px;
}
.markdown-source,
.markdown-preview {
  min-width: 0;
}
.markdown-preview {
  padding: 18px;
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow-wrap: anywhere;
  max-height: 600px;
  overflow-y: auto;
}
.markdown-preview h2 {
  font-size: 22px;
  margin-top: 0;
}
.markdown-source :deep(textarea) {
  font-family: monospace;
  line-height: 1.6;
}
.image-insert {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}
.editor-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 22px;
}
.mobile-tabs {
  display: none;
}
@media (max-width: 720px) {
  .markdown-grid {
    grid-template-columns: minmax(0, 1fr);
  }
  .mobile-tabs {
    display: flex;
  }
  .mobile-edit .markdown-preview,
  .mobile-preview .markdown-source {
    display: none;
  }
  .editor-fields {
    display: block;
  }
}
</style>
