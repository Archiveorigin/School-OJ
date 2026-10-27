<template>
  <div class="ranking-roster" v-loading="loading">
    <div class="roster-heading">
      <span
        >排名与评奖名单 <strong>{{ count }} 人</strong></span
      >
      <el-radio-group
        :model-value="modelValue === null ? 'all' : 'selected'"
        size="small"
        :disabled="loading || !!error"
        aria-label="排名人员范围"
        @update:model-value="changeMode"
      >
        <el-radio-button value="all">全体成员</el-radio-button>
        <el-radio-button value="selected">指定人员</el-radio-button>
      </el-radio-group>
    </div>
    <div class="roster-actions">
      <el-button
        size="small"
        :disabled="loading || !!error"
        @click="visible = true"
        >选择排名人员</el-button
      >
      <el-button
        size="small"
        :disabled="loading || !!error"
        @click="importVisible = true"
        >导入用户名名单</el-button
      >
      <el-button v-if="error" size="small" @click="load">重新加载</el-button>
    </div>
    <p class="roster-note">
      名单用于排名和奖项人数换算，不限制团队成员参加或提交。审核通过后锁定；开赛前需撤回审核才能修改。
    </p>
    <p v-if="error || staleIDs.length" role="alert" class="roster-error">
      {{ error || "名单中有已退出或注销的成员，请重新选择。" }}
    </p>
    <el-dialog
      v-model="visible"
      title="选择排名与评奖人员"
      width="min(620px, calc(100vw - 24px))"
      append-to-body
    >
      <el-input
        v-model="search"
        placeholder="搜索用户名或邮箱"
        aria-label="搜索排名人员"
        clearable
      />
      <div class="roster-actions">
        <el-button
          size="small"
          @click="setSelection(members.map((m) => m.user_id))"
          >选择全部</el-button
        ><el-button size="small" @click="setSelection([])"
          >全部不计排名</el-button
        ><span>已选 {{ count }} / {{ members.length }} 人</span>
      </div>
      <div class="roster-list">
        <el-checkbox-group
          :model-value="selectedIDs"
          @update:model-value="setSelection($event as number[])"
        >
          <div
            v-for="member in filtered"
            :key="member.user_id"
            class="roster-member"
          >
            <el-checkbox :value="member.user_id"
              ><span>{{ member.name }}</span
              ><small>{{ member.email }}</small></el-checkbox
            >
          </div>
        </el-checkbox-group>
        <el-empty
          v-if="!filtered.length"
          description="没有匹配的团队成员"
          :image-size="64"
        />
      </div>
      <template #footer
        ><el-button type="primary" @click="visible = false"
          >完成选择</el-button
        ></template
      >
    </el-dialog>
    <el-dialog
      v-model="importVisible"
      title="导入排名人员名单"
      width="min(620px, calc(100vw - 24px))"
      append-to-body
      :close-on-click-modal="false"
    >
      <p class="roster-note">
        每行一个用户名，也支持逗号分隔。按团队内用户名精确匹配；导入会替换当前排名名单，重复用户名只计一次。
      </p>
      <label class="roster-upload"
        >选择 TXT / CSV 文件<input
          type="file"
          accept=".txt,.csv,text/plain,text/csv"
          aria-label="上传用户名名单"
          @change="readFile"
      /></label>
      <el-input
        v-model="importText"
        type="textarea"
        :autosize="{ minRows: 6, maxRows: 12 }"
        aria-label="用户名名单"
        placeholder="每行一个用户名"
      />
      <p aria-live="polite">
        匹配 {{ preview.matched.length }} 人 · 重复
        {{ preview.duplicates.length }} 项
      </p>
      <p v-if="preview.missing.length" class="roster-error">
        未匹配：{{ preview.missing.join("、") }}
      </p>
      <p v-if="preview.ambiguous.length" class="roster-error">
        存在重名，请通过“选择排名人员”按邮箱确认：{{
          preview.ambiguous.join("、")
        }}
      </p>
      <p v-if="fileError" role="alert" class="roster-error">{{ fileError }}</p>
      <template #footer
        ><el-button @click="importVisible = false">取消</el-button
        ><el-button
          type="primary"
          :disabled="
            !preview.matched.length ||
            !!preview.missing.length ||
            !!preview.ambiguous.length ||
            !!fileError
          "
          @click="applyImport"
          >应用名单</el-button
        ></template
      >
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { generatedApi } from "../api/client";
import { apiErrorMessage } from "../api/errors";
import {
  matchRankingRoster,
  type RankingMember,
} from "../features/teams/rankingRoster";
const props = defineProps<{ teamId: number; modelValue: number[] | null }>();
const emit = defineEmits<{
  "update:modelValue": [value: number[] | null];
  "count-change": [count: number];
  "validity-change": [valid: boolean];
}>();
const members = ref<RankingMember[]>([]),
  loading = ref(true),
  error = ref("");
const visible = ref(false),
  importVisible = ref(false),
  search = ref(""),
  importText = ref(""),
  fileError = ref("");
const selectedIDs = computed(
  () => props.modelValue ?? members.value.map((m) => m.user_id),
);
const staleIDs = computed(() =>
  selectedIDs.value.filter(
    (id) => !members.value.some((m) => m.user_id === id),
  ),
);
const count = computed(
  () =>
    new Set(
      selectedIDs.value.filter((id) =>
        members.value.some((m) => m.user_id === id),
      ),
    ).size,
);
const filtered = computed(() =>
  members.value.filter((m) =>
    `${m.name} ${m.email}`
      .toLowerCase()
      .includes(search.value.trim().toLowerCase()),
  ),
);
const preview = computed(() =>
  matchRankingRoster(importText.value, members.value),
);
watch(importText, () => {
  fileError.value = "";
});
watch(count, (value) => emit("count-change", value), { immediate: true });
watch(
  () => loading.value || !!error.value || !!staleIDs.value.length,
  (invalid) => emit("validity-change", !invalid),
  { immediate: true },
);
let generation = 0;
async function load() {
  const current = ++generation;
  loading.value = true;
  error.value = "";
  try {
    const { data } = await generatedApi.listTeamMembers<RankingMember[]>({
      path: { id: props.teamId },
    });
    if (current === generation) members.value = data;
  } catch (err) {
    if (current === generation) error.value = apiErrorMessage(err);
  } finally {
    if (current === generation) loading.value = false;
  }
}
watch(() => props.teamId, load, { immediate: true });
function setSelection(ids: number[]) {
  emit(
    "update:modelValue",
    [...new Set(ids)].sort((a, b) => a - b),
  );
}
function changeMode(value: unknown) {
  // Without a loaded member list a "selected" click would silently persist an
  // empty roster; keep the switch disabled until members are known.
  if (value === "selected" && !members.value.length) return;
  emit("update:modelValue", value === "all" ? null : [...selectedIDs.value]);
}
async function readFile(event: Event) {
  const input = event.target as HTMLInputElement,
    file = input.files?.[0];
  fileError.value = "";
  if (!file) return;
  try {
    if (file.size > 1024 * 1024) throw new Error("名单文件不能超过 1 MiB");
    const text = await file.text();
    if (text.includes("\uFFFD"))
      throw new Error("请将名单保存为 UTF-8 编码后重试");
    importText.value = text;
  } catch (err) {
    fileError.value = err instanceof Error ? err.message : "读取文件失败";
  } finally {
    input.value = "";
  }
}
function applyImport() {
  if (
    preview.value.missing.length ||
    preview.value.ambiguous.length ||
    !preview.value.matched.length ||
    fileError.value
  )
    return;
  setSelection(preview.value.matched);
  importText.value = "";
  importVisible.value = false;
}
</script>
<style scoped>
.ranking-roster {
  display: grid;
  gap: 10px;
  width: 100%;
  min-width: 0;
}
.roster-heading,
.roster-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
}
.roster-heading {
  color: var(--text);
  font-size: 13px;
}
.roster-heading strong {
  color: var(--accent);
  margin-left: 6px;
}
.roster-actions {
  justify-content: flex-start;
  margin: 6px 0;
}
.roster-note {
  margin: 0;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.6;
}
.roster-error {
  margin: 0;
  color: var(--el-color-danger);
  font-size: 13px;
  overflow-wrap: anywhere;
}
.roster-list {
  max-height: 360px;
  overflow-y: auto;
  border-block: 1px solid var(--border);
}
.roster-member {
  display: block;
  padding: 7px 4px;
  border-bottom: 1px solid var(--border);
}
.roster-member :deep(.el-checkbox) {
  height: auto;
  width: 100%;
}
.roster-member :deep(.el-checkbox__label) {
  white-space: normal;
  overflow-wrap: anywhere;
  min-width: 0;
}
.roster-member small {
  display: block;
  color: var(--muted);
}
.roster-upload {
  display: grid;
  gap: 8px;
  margin: 14px 0;
  color: var(--text);
}
.roster-upload input {
  max-width: 100%;
}
</style>
