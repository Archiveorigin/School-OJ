<template>
  <section class="paper-workspace">
    <el-alert v-if="error" type="error" :closable="false" :title="error" />
    <section v-if="paper" class="panel paper-panel" v-loading="busy">
      <header>
        <div>
          <h2>{{ paper.event.title }} · 组题与整场审核</h2>
          <p class="muted">
            题目、时间和本场附件统一送审；批准后在开始前可撤回修改，修改后须重新审核。
          </p>
        </div>
        <el-tag>{{ reviewLabels[paper.event.review_status] }}</el-tag>
      </header>
      <el-alert
        v-if="paper.event.review_note"
        :title="paper.event.review_note"
        type="info"
        :closable="false"
      />
      <div class="actions">
        <el-button
          v-if="paper.can_edit"
          type="primary"
          @click="review('submit')"
          >提交整场审核</el-button
        >
        <el-button v-if="canWithdraw" @click="review('withdraw')"
          >撤回并修改</el-button
        >
        <template
          v-if="
            auth.role === 'admin' && paper.event.review_status === 'pending'
          "
          ><el-button type="success" @click="review('approve')"
            >批准整场</el-button
          ><el-button type="danger" @click="review('reject')"
            >退回修改</el-button
          ></template
        >
        <el-button :disabled="busy" @click="refresh">刷新</el-button>
      </div>
      <el-collapse>
        <el-collapse-item title="场次说明与时间" name="settings"
          ><el-form label-position="top" :disabled="!paper.can_edit">
            <el-form-item label="名称"
              ><el-input
                v-model="settings.title"
                maxlength="200" /></el-form-item
            ><el-form-item label="说明"
              ><el-input
                v-model="settings.description"
                type="textarea"
                :rows="4"
                maxlength="20000"
            /></el-form-item>
            <div class="actions">
              <el-form-item label="开始时间"
                ><el-date-picker
                  v-model="settings.starts_at"
                  type="datetime"
                  value-format="YYYY-MM-DDTHH:mm:ssZ" /></el-form-item
              ><el-form-item label="结束时间"
                ><el-date-picker
                  v-model="settings.ends_at"
                  type="datetime"
                  value-format="YYYY-MM-DDTHH:mm:ssZ"
              /></el-form-item>
            </div>
            <el-button v-if="paper.can_edit" @click="saveSettings"
              >保存说明与时间</el-button
            >
          </el-form></el-collapse-item
        >
      </el-collapse>
      <div v-if="paper.can_edit" class="actions">
        <el-button type="primary" @click="editProblem()">新增本场题目</el-button
        ><el-select
          v-model="selectedID"
          filterable
          remote
          :remote-method="searchChoices"
          :loading="searching"
          placeholder="搜索题库题号或标题"
          class="problem-select"
          @visible-change="(visible: boolean) => visible && searchChoices('')"
          ><el-option
            v-for="choice in choices"
            :key="choice.id"
            :value="choice.id"
            :label="`${choice.display_code || choice.id} · ${choice.title}`" /></el-select
        ><el-button :disabled="!selectedID" @click="addSelected"
          >添加已有题目</el-button
        >
      </div>
      <div class="paper-problems">
        <article
          v-for="(link, index) in paper.problems"
          :key="link.problem_id"
          class="paper-problem"
        >
          <div class="problem-heading">
            <strong>{{ link.label }}. {{ link.problem.title }}</strong
            ><span class="muted"
              >版本 {{ link.problem_version_id }} ·
              {{ link.case_count }} 组测试点<span v-if="kind === 'exam'">
                · {{ link.score }} 分</span
              ></span
            >
          </div>
          <div class="actions">
            <el-button
              @click="
                preview =
                  preview === link.problem_id ? undefined : link.problem_id
              "
              >查看题面</el-button
            ><el-button @click="download(link)">下载审核题包</el-button>
            <template v-if="paper.can_edit"
              ><el-button
                v-if="
                  link.problem.event_kind === kind &&
                  link.problem.event_id === eventId
                "
                @click="editProblem(link)"
                >编辑新题</el-button
              ><el-button @click="adoptLatest(link)">采用最新版本</el-button
              ><el-button :disabled="index === 0" @click="move(index, -1)"
                >上移</el-button
              ><el-button
                :disabled="index === paper.problems.length - 1"
                @click="move(index, 1)"
                >下移</el-button
              ><el-button v-if="kind === 'exam'" @click="changeScore(link)"
                >分值</el-button
              ><el-button type="danger" link @click="remove(link)"
                >移除</el-button
              ></template
            >
          </div>
          <ProblemStatementView
            v-if="preview === link.problem_id"
            embedded
            :problem="link.problem"
            :show-meta="false"
            :asset-base="eventAssets(kind, eventId, link.problem_id)"
          />
        </article>
      </div>
      <el-empty
        v-if="!paper.problems.length"
        description="先添加题目，再提交整场审核"
        :image-size="70"
      />
      <el-collapse v-if="paper.history?.length"
        ><el-collapse-item title="审核记录" name="history"
          ><p v-for="entry in paper.history" :key="entry.id">
            {{ formatDateTime(entry.created_at) }} · 修订 {{ entry.revision }} ·
            {{ actionLabels[entry.action] || entry.action }} {{ entry.note }}
          </p></el-collapse-item
        ></el-collapse
      >
    </section>
    <EventAttachments
      v-if="paper"
      :kind="kind"
      :event-id="eventId"
      :revision="paper.event.review_revision"
      :editable="paper.can_edit && !busy"
      @changed="refresh"
    />
    <el-dialog
      v-model="editorOpen"
      title="本场题目"
      width="min(960px, calc(100vw - 24px))"
      destroy-on-close
      :close-on-click-modal="false"
      ><EventProblemEditor
        v-if="paper && editorOpen"
        :kind="kind"
        :event-id="eventId"
        :revision="paper.event.review_revision"
        :link="editing"
        :next-label="labelAt(paper.problems.length)"
        @saved="problemSaved"
        @cancel="editorOpen = false"
    /></el-dialog>
  </section>
</template>
<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { ElMessageBox } from "element-plus";
import { generatedApi, type Problem } from "../../api/client";
import { apiErrorMessage } from "../../api/errors";
import {
  downloadPaperProblem,
  eventAssets,
  eventPath,
  loadPaper,
  reviewLabels,
  type EventKind,
  type Paper,
  type PaperProblem,
} from "../../api/modules/events";
import { useAuthStore } from "../../stores/auth";
import { formatDateTime } from "../../features/time";
import EventAttachments from "./EventAttachments.vue";
import EventProblemEditor from "./EventProblemEditor.vue";
import ProblemStatementView from "../ProblemStatementView.vue";
const props = defineProps<{ kind: EventKind; eventId: number }>();
const emit = defineEmits<{ changed: [] }>();
const auth = useAuthStore(),
  paper = ref<Paper>(),
  error = ref(""),
  busy = ref(false),
  searching = ref(false),
  choices = ref<Problem[]>([]),
  selectedID = ref<number>(),
  preview = ref<number>(),
  editorOpen = ref(false),
  editing = ref<PaperProblem>();
const settings = reactive({
  title: "",
  description: "",
  starts_at: "",
  ends_at: "",
});
const actionLabels: Record<string, string> = {
  submit: "提交审核",
  approve: "批准",
  reject: "退回",
  withdraw: "撤回",
  delete: "删除",
};
const canWithdraw = computed(
  () =>
    paper.value?.event.review_status === "pending" ||
    (paper.value?.event.review_status === "approved" &&
      !!paper.value.event.starts_at &&
      Date.now() < Date.parse(paper.value.event.starts_at)),
);
let loadSequence = 0,
  searchSequence = 0;
async function load() {
  const sequence = ++loadSequence;
  error.value = "";
  try {
    const result = await loadPaper(props.kind, props.eventId);
    if (sequence !== loadSequence) return;
    paper.value = result;
    const e = result.event;
    Object.assign(settings, {
      title: e.title,
      description: e.description,
      starts_at: e.starts_at || "",
      ends_at:
        e.ends_at ||
        (e.starts_at && e.duration_minutes
          ? new Date(
              Date.parse(e.starts_at) + e.duration_minutes * 60000,
            ).toISOString()
          : ""),
    });
  } catch (e) {
    if (sequence === loadSequence) {
      paper.value = undefined;
      error.value = apiErrorMessage(e);
    }
  }
}
async function refresh() {
  await load();
  emit("changed");
}
async function mutate(action: () => Promise<unknown>) {
  if (busy.value) return;
  busy.value = true;
  error.value = "";
  try {
    await action();
    await refresh();
  } catch (e) {
    error.value = apiErrorMessage(e);
  } finally {
    busy.value = false;
  }
}
async function review(action: "submit" | "withdraw" | "approve" | "reject") {
  if (!paper.value) return;
  let note = "";
  try {
    if (action === "reject") {
      const answer = await ElMessageBox.prompt(
        "请填写整场退回原因",
        "退回审核",
        {
          confirmButtonText: "退回",
          cancelButtonText: "取消",
          inputValidator: (value) => !!value?.trim() || "退回原因不能为空",
        },
      );
      note = answer.value;
    } else {
      await ElMessageBox.confirm(
        action === "approve"
          ? "请确认本场题面、测试点、时间和附件均已核对。批准后参与者将按开始时间获得访问权限。"
          : action === "withdraw"
            ? "撤回后参与者无法进入，修改完成后须重新审核。"
            : "提交后题目、设置和附件将锁定，等待管理员整场审核。",
        actionLabels[action],
        { confirmButtonText: "确认", cancelButtonText: "取消" },
      );
    }
  } catch {
    return;
  }
  await mutate(() =>
    generatedApi.reviewEventPaper({
      path: eventPath(props.kind, props.eventId),
      body: { revision: paper.value!.event.review_revision, action, note },
    }),
  );
}
function saveSettings() {
  if (!paper.value) return;
  void mutate(() =>
    generatedApi.updateEventSettings({
      path: eventPath(props.kind, props.eventId),
      body: { ...settings, revision: paper.value!.event.review_revision },
    }),
  );
}
async function searchChoices(q: string) {
  const sequence = ++searchSequence;
  searching.value = true;
  try {
    const { data } = await generatedApi.listEventProblemChoices<Problem[]>({
      path: eventPath(props.kind, props.eventId),
      query: { q },
    });
    if (sequence === searchSequence) choices.value = data;
  } catch (e) {
    if (sequence === searchSequence) error.value = apiErrorMessage(e);
  } finally {
    if (sequence === searchSequence) searching.value = false;
  }
}
function labelAt(index: number) {
  let value = "";
  for (index++; index > 0; index = Math.floor((index - 1) / 26))
    value = String.fromCharCode(65 + ((index - 1) % 26)) + value;
  return value;
}
async function saveSelection(
  problems: Array<{
    problem_id: number;
    problem_version_id?: number;
    label: string;
    score: number;
    use_latest?: boolean;
  }>,
) {
  await mutate(() =>
    generatedApi.updateEventProblemSelection({
      path: eventPath(props.kind, props.eventId),
      body: { revision: paper.value!.event.review_revision, problems },
    }),
  );
}
function selection() {
  return paper.value!.problems.map(
    ({ problem_id, problem_version_id, label, score }) => ({
      problem_id,
      problem_version_id,
      label,
      score,
    }),
  );
}
async function addSelected() {
  if (!selectedID.value || !paper.value) return;
  if (paper.value.problems.some((p) => p.problem_id === selectedID.value)) {
    error.value = "此题已在本场";
    return;
  }
  await saveSelection([
    ...selection(),
    {
      problem_id: selectedID.value,
      label: labelAt(paper.value.problems.length),
      score: 100,
    },
  ]);
  selectedID.value = undefined;
}
async function move(index: number, offset: number) {
  const list = selection();
  [list[index], list[index + offset]] = [list[index + offset], list[index]];
  list.forEach((p, i) => {
    p.label = labelAt(i);
  });
  await saveSelection(list);
}
async function remove(link: PaperProblem) {
  try {
    await ElMessageBox.confirm(
      `从本场移除 ${link.label}.${link.problem.title}？`,
      "移除题目",
    );
  } catch {
    return;
  }
  await saveSelection(
    selection()
      .filter((p) => p.problem_id !== link.problem_id)
      .map((p, i) => ({ ...p, label: labelAt(i) })),
  );
}
async function adoptLatest(link: PaperProblem) {
  try {
    await ElMessageBox.confirm(
      "将采用题库当前最新版本。请重新核对题面和测试点，再提交整场审核。",
      "采用最新版本",
    );
  } catch {
    return;
  }
  await saveSelection(
    selection().map((p) => ({
      ...p,
      use_latest: p.problem_id === link.problem_id,
    })),
  );
}
async function changeScore(link: PaperProblem) {
  let value: string;
  try {
    const answer = await ElMessageBox.prompt(
      "输入 1 至 1000 的整数分值",
      "题目分值",
      {
        inputValue: String(link.score),
        inputValidator: (v) =>
          (/^\d+$/.test(v || "") && Number(v) >= 1 && Number(v) <= 1000) ||
          "分值无效",
      },
    );
    value = answer.value;
  } catch {
    return;
  }
  await saveSelection(
    selection().map((p) => ({
      ...p,
      score: p.problem_id === link.problem_id ? Number(value) : p.score,
    })),
  );
}
function editProblem(link?: PaperProblem) {
  editing.value = link;
  editorOpen.value = true;
}
async function problemSaved() {
  editorOpen.value = false;
  await refresh();
}
async function download(link: PaperProblem) {
  try {
    await downloadPaperProblem(props.kind, props.eventId, link);
  } catch (e) {
    error.value = apiErrorMessage(e);
  }
}
watch(() => [props.kind, props.eventId], load, { immediate: true });
</script>
<style scoped>
.paper-workspace {
  display: grid;
  gap: 18px;
  min-width: 0;
  margin-top: 18px;
}
.paper-panel {
  padding: 20px;
  display: grid;
  gap: 18px;
  min-width: 0;
}
header,
.actions,
.problem-heading {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
header {
  justify-content: space-between;
}
h2 {
  margin: 0;
  font-size: 20px;
}
.muted {
  color: var(--muted);
  line-height: 1.7;
}
.problem-select {
  width: min(360px, 100%);
}
.paper-problem {
  padding: 16px 0;
  border-top: 1px solid var(--border);
  display: grid;
  gap: 12px;
}
.problem-heading strong {
  overflow-wrap: anywhere;
}
.actions :deep(.el-button + .el-button) {
  margin-left: 0;
}
@media (max-width: 600px) {
  .paper-panel {
    padding: 14px;
  }
}
</style>
