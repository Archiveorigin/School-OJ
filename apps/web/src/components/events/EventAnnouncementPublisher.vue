<template>
  <el-dialog
    append-to-body
    v-model="editorOpen"
    :close-on-click-modal="!saving"
    :close-on-press-escape="!saving"
    :show-close="!saving"
    title="发布赛时公告"
    width="min(640px, calc(100vw - 24px))"
    ><el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
    /><el-form v-if="canPublish" label-position="top" @submit.prevent="publish">
      <el-form-item label="关联题目 ID（选填）"
        ><el-input
          v-model="problem"
          inputmode="numeric"
          :disabled="saving || Boolean(publishedID)"
      /></el-form-item>
      <el-form-item label="公告正文（Markdown）"
        ><el-input
          v-model="body"
          type="textarea"
          :rows="6"
          placeholder="支持 Markdown、公式和 HTTPS 图片链接"
          :disabled="saving || Boolean(publishedID)"
          maxlength="4000"
          show-word-limit
      /></el-form-item>
      <el-button :disabled="!body.trim()" @click="preview = !preview">{{
        preview ? "收起预览" : "预览正文"
      }}</el-button>
      <div v-if="preview" class="announcement-preview">
        <MarkdownRenderer :source="body" />
      </div>
      <el-button :disabled="saving" @click="fileInput?.click()"
        >添加图片或文档</el-button
      ><span>{{ files.map((file) => file.name).join("、") }}</span
      ><input
        ref="fileInput"
        hidden
        type="file"
        multiple
        accept=".png,.jpg,.jpeg,.gif,.webp,.pdf,.doc,.docx,.txt,.xlsx,.csv"
        @change="selectFiles"
      />
      <p>
        附件每个最多 32
        MiB。赛后同时归入本场附件。发布后保留历史；更正请另发公告。
      </p>
      <el-button
        type="primary"
        native-type="submit"
        :loading="saving"
        :disabled="!body.trim()"
        >发布公告</el-button
      >
    </el-form></el-dialog
  >
</template>
<script setup lang="ts">
import { ref, watch } from "vue";
import MarkdownRenderer from "../MarkdownRenderer.vue";
import { generatedApi } from "../../api/client";
import { eventPath, type EventKind } from "../../api/modules/events";
import { apiErrorMessage } from "../../api/errors";
const fileInput = ref<HTMLInputElement>();
const props = defineProps<{
  kind: EventKind;
  eventId: number;
  canPublish?: boolean;
}>();
const body = ref(""),
  problem = ref(""),
  error = ref(""),
  saving = ref(false);
const files = ref<File[]>([]);
function selectFiles(e: Event) {
  files.value = Array.from((e.target as HTMLInputElement).files || []);
}
const publishedID = ref<number>();
const emit = defineEmits<{ published: [] }>();
const preview = ref(false);
async function publish() {
  if (problem.value && !/^[1-9]\d*$/.test(problem.value)) {
    error.value = "请输入有效题目 ID";
    return;
  }
  if (
    files.value.some((file) => file.size === 0 || file.size > 32 * 1024 * 1024)
  ) {
    error.value = "附件必须为非空文件，单个最多 32 MiB";
    return;
  }
  saving.value = true;
  try {
    if (!publishedID.value) {
      const { data } = await generatedApi.publishEventAnnouncement<{
        id: number;
      }>({
        path: eventPath(props.kind, props.eventId),
        body: {
          body: body.value,
          problem_id: problem.value ? Number(problem.value) : null,
        },
      });
      publishedID.value = data.id;
    }
    while (files.value.length) {
      const form = new FormData();
      form.append("file", files.value[0]);
      form.append("announcement_id", String(publishedID.value));
      await generatedApi.uploadEventAttachment({
        path: eventPath(props.kind, props.eventId),
        body: form,
        timeout: 120000,
      });
      files.value.shift();
    }
    publishedID.value = undefined;
    body.value = "";
    problem.value = "";
    editorOpen.value = false;
    error.value = "";
    preview.value = false;
    emit("published");
  } catch (e) {
    error.value =
      (publishedID.value
        ? "公告正文已发布，附件未全部上传；再次点击将重试剩余附件。"
        : "") + apiErrorMessage(e);
  } finally {
    saving.value = false;
  }
}
const editorOpen = ref(false);
defineExpose({
  openPublisher: () => {
    if (props.canPublish) editorOpen.value = true;
  },
});
watch(
  () => [props.kind, props.eventId],
  () => {
    editorOpen.value = false;
    body.value = "";
    problem.value = "";
    files.value = [];
    publishedID.value = undefined;
    error.value = "";
  },
);
</script>
<style scoped>
.selected-files {
  overflow-wrap: anywhere;
}
.announcement-preview {
  margin: 12px 0;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: 8px;
}
</style>
