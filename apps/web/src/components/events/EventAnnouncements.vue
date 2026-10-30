<template>
  <section class="panel announcements" aria-label="赛时公告">
    <div class="heading">
      <h3>赛时公告</h3>
      <el-button :loading="loading" @click="load">刷新</el-button>
    </div>
    <el-alert v-if="error" type="error" :title="error" :closable="false" />
    <ol v-if="items.length" aria-live="polite">
      <li v-for="item in items" :key="item.id">
        <small
          >{{ formatDateTime(item.created_at) }} ·
          {{ item.problem_id ? `题目 #${item.problem_id}` : "全场公告" }}</small
        >
        <p>{{ item.body }}</p>
        <el-button
          v-for="file in item.attachments"
          :key="file.id"
          link
          @click="downloadEventFile(kind, eventId, file)"
          >{{ file.name }}</el-button
        >
      </li>
    </ol>
    <p v-else-if="!loading && !error">暂无公告</p>
    <el-dialog
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
      /><el-form
        v-if="canPublish"
        label-position="top"
        @submit.prevent="publish"
      >
        <el-form-item label="关联题目 ID（选填）"
          ><el-input v-model="problem" inputmode="numeric"
        /></el-form-item>
        <el-form-item label="公告正文"
          ><el-input
            v-model="body"
            type="textarea"
            :rows="3"
            maxlength="4000"
            show-word-limit
        /></el-form-item>
        <el-button @click="fileInput?.click()">添加图片或文档</el-button
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
  </section>
</template>
<script setup lang="ts">
import { ref, watch, onUnmounted } from "vue";
import { generatedApi } from "../../api/client";
import {
  eventPath,
  downloadEventFile,
  type EventFile,
  type EventKind,
} from "../../api/modules/events";
import { apiErrorMessage } from "../../api/errors";
import { formatDateTime } from "../../features/time";
const fileInput = ref<HTMLInputElement>();
const props = defineProps<{
  kind: EventKind;
  eventId: number;
  canPublish?: boolean;
}>();
const items = ref<
  {
    id: number;
    problem_id?: number;
    body: string;
    created_at: string;
    attachments?: EventFile[];
  }[]
>([]);
const body = ref(""),
  problem = ref(""),
  error = ref(""),
  saving = ref(false),
  loading = ref(false);
const files = ref<File[]>([]);
function selectFiles(e: Event) {
  files.value = Array.from((e.target as HTMLInputElement).files || []);
}
let publishedID: number | undefined;
let generation = 0;
async function load() {
  const current = ++generation;
  loading.value = true;
  try {
    const { data } = await generatedApi.listEventAnnouncements<
      typeof items.value
    >({ path: eventPath(props.kind, props.eventId) });
    if (current === generation) {
      items.value = data;
      error.value = "";
    }
  } catch (e) {
    if (current === generation) error.value = apiErrorMessage(e);
  } finally {
    if (current === generation) loading.value = false;
  }
}
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
    if (!publishedID) {
      const { data } = await generatedApi.publishEventAnnouncement<{
        id: number;
      }>({
        path: eventPath(props.kind, props.eventId),
        body: {
          body: body.value,
          problem_id: problem.value ? Number(problem.value) : null,
        },
      });
      publishedID = data.id;
    }
    while (files.value.length) {
      const form = new FormData();
      form.append("file", files.value[0]);
      form.append("announcement_id", String(publishedID));
      await generatedApi.uploadEventAttachment({
        path: eventPath(props.kind, props.eventId),
        body: form,
        timeout: 120000,
      });
      files.value.shift();
    }
    publishedID = undefined;
    body.value = "";
    problem.value = "";
    editorOpen.value = false;
    await load();
  } catch (e) {
    error.value =
      (publishedID
        ? "公告正文已发布，附件未全部上传；再次点击将重试剩余附件。"
        : "") + apiErrorMessage(e);
  } finally {
    saving.value = false;
  }
}
onUnmounted(() => {
  generation++;
});
const editorOpen = ref(false);
defineExpose({
  openPublisher: () => {
    editorOpen.value = true;
  },
});
watch(
  () => [props.kind, props.eventId],
  () => {
    items.value = [];
    void load();
  },
  { immediate: true },
);
</script>
<style scoped>
.announcements {
  margin: 16px 0;
  border-left: 4px solid var(--el-color-warning);
}
.heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
ol {
  padding-left: 24px;
  max-height: 320px;
  overflow: auto;
}
li p {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  margin-top: 6px;
}
</style>
