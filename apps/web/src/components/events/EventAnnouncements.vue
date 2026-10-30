<template>
  <section class="panel announcements" aria-label="赛时公告">
    <header class="announcement-heading">
      <h3>答疑 / 公告</h3>
      <div class="announcement-actions">
        <el-button :loading="loading" @click="load">刷新</el-button>
        <el-button v-if="canAsk" type="primary" @click="$emit('ask')"
          >提问</el-button
        >
      </div>
    </header>
    <el-alert v-if="error" type="error" :title="error" :closable="false" />
    <ol v-if="items.length" class="announcement-list" aria-live="polite">
      <li v-for="item in items" :key="item.id" class="announcement-item">
        <div class="announcement-meta">
          <span>公告 · {{ problemLabel(item.problem_id) }}</span>
          <time :datetime="item.created_at">{{
            formatDateTime(item.created_at)
          }}</time>
        </div>
        <MarkdownRenderer :source="item.body" />
        <div v-if="item.attachments?.length" class="announcement-files">
          <el-button
            v-for="file in item.attachments"
            :key="file.id"
            link
            @click="downloadEventFile(kind, eventId, file)"
            >{{ file.name }}</el-button
          >
        </div>
      </li>
    </ol>
    <p v-else-if="!loading && !error" class="announcement-empty">暂无公告</p>
    <p
      v-else-if="loading && !items.length"
      class="announcement-empty"
      role="status"
    >
      正在加载公告…
    </p>
  </section>
</template>
<script setup lang="ts">
import { ref, watch, onUnmounted } from "vue";
import MarkdownRenderer from "../MarkdownRenderer.vue";
import { generatedApi } from "../../api/client";
import {
  eventPath,
  downloadEventFile,
  type EventFile,
  type EventKind,
} from "../../api/modules/events";
import { apiErrorMessage } from "../../api/errors";
import { formatDateTime } from "../../features/time";
const props = defineProps<{
  kind: EventKind;
  eventId: number;
  canAsk?: boolean;
  problems?: { problem_id: number; label: string }[];
}>();
defineEmits<{ ask: [] }>();
const items = ref<
  {
    id: number;
    problem_id?: number;
    body: string;
    created_at: string;
    attachments?: EventFile[];
  }[]
>([]);
const error = ref(""),
  loading = ref(false);
function problemLabel(id?: number) {
  if (!id) return "全场公告";
  return (
    props.problems?.find((p) => p.problem_id === id)?.label || `题目 #${id}`
  );
}
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
onUnmounted(() => generation++);
watch(
  () => [props.kind, props.eventId],
  () => {
    items.value = [];
    void load();
  },
  { immediate: true },
);
defineExpose({ reload: load });
</script>
<style scoped>
.announcements {
  min-width: 0;
}
.announcement-heading,
.announcement-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}
.announcement-heading {
  justify-content: space-between;
  flex-wrap: wrap;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
}
.announcement-heading h3 {
  margin: 0;
}
.announcement-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}
.announcement-list {
  display: grid;
  gap: 16px;
  list-style: none;
  margin: 20px 0 0;
  padding: 0;
}
.announcement-item {
  min-width: 0;
  padding: 16px 20px;
  border: 1px solid var(--border);
  border-left: 4px solid var(--el-color-primary);
  border-radius: 8px;
}
.announcement-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 16px;
  margin-bottom: 12px;
  color: var(--muted);
  font-size: 13px;
}
.announcement-item :deep(.markdown-body > :last-child) {
  margin-bottom: 0;
}
.announcement-files {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 16px;
  margin-top: 12px;
}
.announcement-files :deep(.el-button) {
  max-width: 100%;
  margin-left: 0;
  white-space: normal;
  height: auto;
  text-align: left;
  overflow-wrap: anywhere;
}
.announcement-empty {
  color: var(--muted);
  margin-bottom: 0;
}
@media (max-width: 600px) {
  .announcement-item {
    padding: 12px;
  }
}
</style>
