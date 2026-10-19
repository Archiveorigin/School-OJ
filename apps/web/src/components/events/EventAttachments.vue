<template>
  <section class="event-attachments panel" aria-label="本场附件">
    <div class="attachment-heading">
      <h3>本场附件</h3>
      <label v-if="editable" class="upload-label"
        >上传附件<input type="file" :disabled="busy" @change="upload"
      /></label>
    </div>
    <p class="muted">
      {{
        editable
          ? "单个文件最多 32 MiB，每场最多 20 个、共 96 MiB。附件随整场内容一并审核。"
          : "附件仅供本场有权限的人员下载。"
      }}
    </p>
    <el-alert v-if="error" type="error" :title="error" :closable="false" />
    <ul v-if="files.length">
      <li v-for="file in files" :key="file.id">
        <span
          >{{ file.name }}
          <small>{{ (file.size / 1024).toFixed(1) }} KiB</small></span
        >
        <div>
          <el-button :disabled="busy" @click="download(file)">下载</el-button
          ><el-button
            v-if="editable"
            type="danger"
            link
            :disabled="busy"
            @click="remove(file)"
            >移除</el-button
          >
        </div>
      </li>
    </ul>
    <p v-else-if="!error" class="muted">暂无附件</p>
  </section>
</template>
<script setup lang="ts">
import { ref, watch } from "vue";
import { ElMessageBox } from "element-plus";
import { generatedApi } from "../../api/client";
import { apiErrorMessage } from "../../api/errors";
import {
  downloadEventFile,
  eventPath,
  type EventFile,
  type EventKind,
} from "../../api/modules/events";
const props = defineProps<{
  kind: EventKind;
  eventId: number;
  revision?: number;
  editable?: boolean;
}>();
const emit = defineEmits<{ changed: [] }>();
const files = ref<EventFile[]>([]),
  error = ref(""),
  busy = ref(false);
let generation = 0;
async function load() {
  const current = ++generation;
  error.value = "";
  try {
    const { data } = await generatedApi.listEventAttachments<EventFile[]>({
      path: eventPath(props.kind, props.eventId),
    });
    if (current === generation) files.value = data;
  } catch (e) {
    if (current === generation) {
      files.value = [];
      error.value = apiErrorMessage(e);
    }
  }
}
async function upload(event: Event) {
  const input = event.target as HTMLInputElement,
    file = input.files?.[0];
  if (!file) return;
  input.value = "";
  if (!file.size || file.size > 32 * 1024 * 1024) {
    error.value = "请选择非空且不超过 32 MiB 的附件";
    return;
  }
  busy.value = true;
  error.value = "";
  try {
    const body = new FormData();
    body.append("file", file);
    body.append("revision", String(props.revision));
    await generatedApi.uploadEventAttachment({
      path: eventPath(props.kind, props.eventId),
      body,
      timeout: 120000,
    });
    emit("changed");
    await load();
  } catch (e) {
    error.value = apiErrorMessage(e);
  } finally {
    busy.value = false;
  }
}
async function remove(file: EventFile) {
  try {
    await ElMessageBox.confirm(`移除附件“${file.name}”？`, "移除附件", {
      type: "warning",
    });
  } catch {
    return;
  }
  busy.value = true;
  error.value = "";
  try {
    await generatedApi.deleteEventAttachment({
      path: { ...eventPath(props.kind, props.eventId), attachment_id: file.id },
      body: { revision: props.revision },
    });
    emit("changed");
    await load();
  } catch (e) {
    error.value = apiErrorMessage(e);
  } finally {
    busy.value = false;
  }
}
async function download(file: EventFile) {
  busy.value = true;
  error.value = "";
  try {
    await downloadEventFile(props.kind, props.eventId, file);
  } catch (e) {
    error.value = apiErrorMessage(e, "下载失败，请确认本场访问权限");
  } finally {
    busy.value = false;
  }
}
watch(() => [props.kind, props.eventId, props.revision], load, {
  immediate: true,
});
</script>
<style scoped>
.event-attachments {
  padding: 20px;
  min-width: 0;
}
.attachment-heading,
li,
li > div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
h3 {
  margin: 0;
}
.muted,
small {
  color: var(--muted);
}
ul {
  padding: 0;
  list-style: none;
}
li {
  padding: 12px 0;
  border-bottom: 1px solid var(--border);
}
li > span {
  overflow-wrap: anywhere;
  min-width: 0;
}
.upload-label {
  color: var(--accent);
  cursor: pointer;
}
.upload-label input {
  display: block;
  max-width: 240px;
  margin-top: 8px;
}
</style>
