<template>
  <el-dialog
    :model-value="modelValue"
    title="添加现有题目"
    width="min(960px, calc(100vw - 24px))"
    append-to-body
    destroy-on-close
    :close-on-click-modal="!saving"
    :close-on-press-escape="!saving"
    :show-close="!saving"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="event-picker">
      <div class="picker-filters">
        <el-select v-model="source" aria-label="题目来源" @change="search">
          <el-option label="公共题库" value="public" />
          <el-option
            v-if="kind === 'contest'"
            label="团队私有题目／题单"
            value="team"
          />
          <el-option v-else label="全部可用题目" value="all" />
        </el-select>
        <el-select
          v-if="source === 'team'"
          v-model="setID"
          clearable
          placeholder="本团队全部题目"
          aria-label="团队题单"
          @change="search"
          ><el-option
            v-for="set in sets"
            :key="set.id"
            :value="set.id"
            :label="set.title"
        /></el-select>
        <el-input
          v-model="query"
          clearable
          placeholder="搜索题号或标题"
          aria-label="搜索现有题目"
          @keyup.enter="search"
        />
        <el-button @click="search">搜索</el-button>
      </div>
      <p class="picker-note">
        跨页、切换来源均保留已选题目。本场最多 100 道题，已有题目不会重复添加。
      </p>
      <el-alert v-if="error" :title="error" type="error" :closable="false" />
      <div class="picker-results" v-loading="loading">
        <div v-for="problem in items" :key="problem.id" class="picker-row">
          <el-checkbox
            :model-value="
              selected.has(problem.id) || existingIDs.has(problem.id)
            "
            :disabled="saving || existingIDs.has(problem.id)"
            :aria-label="`选择 ${problem.display_code || problem.id} ${problem.title}`"
            @change="toggle(problem)"
          >
            <span class="problem-code">{{
              problem.display_code || problem.id
            }}</span
            ><span>{{ problem.title }}</span>
          </el-checkbox>
          <el-tag v-if="existingIDs.has(problem.id)" type="info"
            >已在本场</el-tag
          >
        </div>
        <el-empty
          v-if="!loading && !items.length"
          description="暂无符合条件的题目"
          :image-size="80"
        />
      </div>
      <el-pagination
        v-model:current-page="page"
        :total="total"
        :page-size="20"
        :pager-count="5"
        layout="total, prev, pager, next"
        @current-change="load"
      />
      <section class="selection-tray" aria-label="已选题目">
        <header>
          <strong>已选 {{ selected.size }} 道</strong
          ><el-button link :disabled="saving" @click="selected.clear()"
            >清空</el-button
          >
        </header>
        <div class="selected-list">
          <el-tag
            v-for="problem in selected.values()"
            :key="problem.id"
            :closable="!saving"
            @close="toggle(problem)"
            >{{ problem.display_code || problem.id }} ·
            {{ problem.title }}</el-tag
          ><span v-if="!selected.size" class="picker-note"
            >选择的题目会显示在这里</span
          >
        </div>
      </section>
    </div>
    <template #footer
      ><el-button :disabled="saving" @click="$emit('update:modelValue', false)"
        >取消</el-button
      ><el-button
        type="primary"
        :loading="saving"
        :disabled="!selected.size"
        @click="add"
        >添加已选 {{ selected.size }} 道题目</el-button
      ></template
    >
  </el-dialog>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { generatedApi, type Problem } from "../../api/client";
import { apiErrorMessage } from "../../api/errors";
import {
  eventPath,
  type EventKind,
  type Paper,
} from "../../api/modules/events";
const props = defineProps<{
  modelValue: boolean;
  kind: EventKind;
  eventId: number;
  paper: Paper;
}>();
const emit = defineEmits<{ "update:modelValue": [boolean]; added: [] }>();
const source = ref("public"),
  query = ref(""),
  setID = ref<number>(),
  sets = ref<Array<{ id: number; title: string }>>([]);
const items = ref<Problem[]>([]),
  selected = ref(new Map<number, Problem>()),
  page = ref(1),
  total = ref(0);
const loading = ref(false),
  saving = ref(false),
  error = ref("");
const existing = ref<Paper["problems"]>([]),
  revision = ref(0);
const existingIDs = computed(
  () => new Set(existing.value.map((p) => p.problem_id)),
);
let controller: AbortController | undefined,
  sequence = 0;
let filters = {
  q: "",
  source: "public",
  problem_set_id: undefined as number | undefined,
};
async function load() {
  controller?.abort();
  controller = new AbortController();
  const signal = controller.signal,
    current = ++sequence;
  loading.value = true;
  error.value = "";
  try {
    const { data } = await generatedApi.listEventProblemChoices<{
      items: Problem[];
      total: number;
    }>({
      path: eventPath(props.kind, props.eventId),
      query: { ...filters, page: page.value },
      signal,
    });
    if (current === sequence) {
      items.value = data.items;
      total.value = data.total;
    }
  } catch (err) {
    if (!signal.aborted) error.value = apiErrorMessage(err);
  } finally {
    if (current === sequence) loading.value = false;
  }
}
function search() {
  filters = {
    q: query.value,
    source: source.value,
    problem_set_id: source.value === "team" ? setID.value : undefined,
  };
  page.value = 1;
  void load();
}
function toggle(problem: Problem) {
  if (saving.value || existingIDs.value.has(problem.id)) return;
  if (selected.value.has(problem.id)) selected.value.delete(problem.id);
  else if (existing.value.length + selected.value.size >= 100)
    ElMessage.warning("每场最多 100 道题目");
  else selected.value.set(problem.id, problem);
}
function labelAt(index: number) {
  let text = "";
  for (index++; index > 0; index = Math.floor((index - 1) / 26))
    text = String.fromCharCode(65 + ((index - 1) % 26)) + text;
  return text;
}
async function add() {
  if (saving.value || !selected.value.size) return;
  saving.value = true;
  error.value = "";
  const problems = existing.value.map(
    ({ problem_id, problem_version_id, label, score }) => ({
      problem_id,
      problem_version_id,
      label,
      score,
    }),
  );
  const labels = new Set(problems.map((p) => p.label.toUpperCase()));
  let index = 0;
  for (const id of selected.value.keys()) {
    while (labels.has(labelAt(index))) index++;
    const label = labelAt(index++);
    labels.add(label);
    problems.push({ problem_id: id, problem_version_id: 0, label, score: 100 });
  }
  try {
    await generatedApi.updateEventProblemSelection({
      path: eventPath(props.kind, props.eventId),
      body: { revision: revision.value, problems },
    });
    ElMessage.success(`已添加 ${selected.value.size} 道题目`);
    selected.value.clear();
    emit("added");
    emit("update:modelValue", false);
  } catch (err) {
    error.value = apiErrorMessage(err, "添加失败，已保留选择，请核对后重试");
  } finally {
    saving.value = false;
  }
}
watch(
  () => props.modelValue,
  async (open) => {
    if (!open) {
      controller?.abort();
      sequence++;
      loading.value = false;
      return;
    }
    existing.value = props.paper.problems.map((p) => ({ ...p }));
    revision.value = props.paper.event.review_revision;
    for (const id of existingIDs.value) selected.value.delete(id);
    void load();
    if (props.kind === "contest" && props.paper.event.team_id) {
      try {
        sets.value = (
          await generatedApi.listTeamProblemSets<
            Array<{ id: number; title: string }>
          >({ path: { id: props.paper.event.team_id } })
        ).data;
      } catch (err) {
        error.value = apiErrorMessage(err, "团队题单加载失败");
      }
    }
  },
);
onBeforeUnmount(() => controller?.abort());
</script>
<style scoped>
.event-picker {
  display: grid;
  gap: 14px;
  min-width: 0;
  color: var(--text);
}
.picker-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.picker-filters .el-select {
  width: 180px;
}
.picker-filters .el-input {
  flex: 1 1 200px;
}
.picker-note {
  margin: 0;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.6;
}
.picker-results {
  min-height: 120px;
  max-height: 42dvh;
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-strong);
}
.picker-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-bottom: 1px solid var(--border);
}
.picker-row .el-checkbox {
  min-width: 0;
  height: auto;
}
.picker-row :deep(.el-checkbox__label) {
  white-space: normal;
  overflow-wrap: anywhere;
}
.problem-code {
  margin-right: 8px;
  font-weight: 700;
  color: var(--accent);
}
.el-pagination {
  justify-content: flex-end;
}
.selection-tray {
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--app-bg);
}
.selection-tray header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.selected-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  max-height: 120px;
  overflow-y: auto;
}
.selected-list .el-tag {
  height: auto;
  max-width: 100%;
  white-space: normal;
  overflow-wrap: anywhere;
  padding: 6px;
}
@media (max-width: 600px) {
  .picker-filters .el-select {
    flex: 1 1 100%;
  }
  .el-pagination {
    justify-content: center;
  }
}
</style>
