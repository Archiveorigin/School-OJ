<template>
  <section class="paper-workspace">
    <el-alert v-if="error" type="error" :closable="false" :title="error" />
    <template v-if="paper">
      <header class="paper-header">
        <div>
          <span class="eyebrow">PAPER WORKSPACE</span>
          <h1>{{ paper.event.title }}</h1>
          <p>组织题目与本场附件，预览完整内容后提交整场审核。</p>
        </div>
        <el-tag size="large">{{ lifecycleLabels[lifecycle] }}</el-tag>
      </header>
      <ol class="lifecycle-track" aria-label="场次流程">
        <li
          v-for="(label, state) in {
            pending: '待审核',
            upcoming: '未开始',
            running: '进行中',
            closed: '已截止',
            completed: '已完结',
          }"
          :key="state"
          :class="{ current: lifecycle === state }"
        >
          {{ label }}
        </li>
      </ol>
      <el-alert
        v-if="paper.event.review_note"
        :title="paper.event.review_note"
        type="info"
        :closable="false"
      />
      <el-alert
        v-if="lifecycle === 'closed'"
        type="warning"
        :closable="false"
        title="本场已截止，答题已关闭。管理员确认完结后，题目、附件和原权限允许的排名才进入前台比赛归档。"
      />
      <div class="paper-toolbar">
        <div>
          <strong>{{ paper.problems.length }}</strong> 道题
          <span v-if="kind === 'exam'"
            >·
            {{ paper.problems.reduce((sum, p) => sum + p.score, 0) }} 分</span
          >
        </div>
        <div class="actions">
          <el-button
            v-if="paper.can_edit"
            :disabled="busy"
            @click="openImport('pdf')"
            >导入 PDF 题册</el-button
          >
          <el-button :disabled="busy || editorOpen" @click="refresh"
            >刷新</el-button
          >
          <el-button
            v-if="canWithdraw"
            :disabled="busy || editorOpen"
            @click="review('withdraw')"
            >撤回并修改</el-button
          >
          <el-button
            v-if="paper.can_edit"
            type="primary"
            :disabled="busy || editorOpen || !paper.problems.length"
            @click="review('submit')"
            >提交整场审核</el-button
          >
          <template
            v-if="
              auth.role === 'admin' && paper.event.review_status === 'pending'
            "
            ><el-button
              :disabled="busy"
              type="danger"
              plain
              @click="review('reject')"
              >退回修改</el-button
            ><el-button
              :disabled="busy"
              type="primary"
              @click="review('approve')"
              >批准整场</el-button
            ></template
          >
          <el-button
            v-if="auth.role === 'admin' && lifecycle === 'closed'"
            type="primary"
            :disabled="
              busy ||
              !!paper.completion_checks?.judging ||
              !!paper.completion_checks?.ungraded
            "
            @click="review('complete')"
            >确认完结</el-button
          >
          <el-button
            v-if="auth.role === 'admin' && lifecycle === 'completed'"
            :disabled="busy"
            @click="review('reopen')"
            >撤销完结</el-button
          >
        </div>
      </div>
      <p
        v-if="
          lifecycle === 'closed' &&
          (paper.completion_checks?.judging ||
            paper.completion_checks?.ungraded)
        "
        class="muted"
      >
        待判题 {{ paper.completion_checks?.judging }} 份 · 待人工评分
        {{ paper.completion_checks?.ungraded }} 份；处理完成后刷新再确认完结。
      </p>
      <div class="paper-layout" v-loading="busy">
        <aside class="paper-rail">
          <div class="rail-title">
            <h2>本场题目</h2>
            <el-button
              v-if="paper.can_edit"
              type="primary"
              size="small"
              :disabled="busy"
              @click="editProblem()"
              >新增题目</el-button
            >
          </div>
          <el-button
            v-if="paper.can_edit"
            class="import-trigger"
            :disabled="busy"
            @click="openImport('markdown')"
            >导入 Markdown 单题</el-button
          >
          <el-button
            v-if="paper.can_edit && importPending"
            link
            class="queue-trigger"
            :disabled="busy"
            @click="openImport()"
            >继续处理导入队列</el-button
          >
          <div v-if="paper.can_edit" class="existing-picker">
            <el-select
              v-model="selectedID"
              filterable
              remote
              :remote-method="searchChoices"
              :loading="searching"
              placeholder="输入题号或标题"
              aria-label="输入题号或标题"
              @visible-change="
                (visible: boolean) => visible && searchChoices('')
              "
              ><el-option
                v-for="choice in choices"
                :key="choice.id"
                :value="choice.id"
                :label="`${choice.display_code || choice.id} · ${choice.title}`" /></el-select
            ><el-button
              :disabled="!selectedID || editorOpen || busy"
              @click="addSelected"
              >添加已有题目</el-button
            >
          </div>
          <nav class="problem-directory" aria-label="试卷题目目录">
            <button
              v-for="link in paper.problems"
              :key="link.problem_id"
              type="button"
              :class="{
                selected:
                  !editorOpen && selected?.problem_id === link.problem_id,
              }"
              @click="selectProblem(link)"
            >
              <span class="letter">{{ link.label }}</span
              ><span
                ><strong>{{ link.problem.title }}</strong
                ><small
                  >{{ link.case_count }} 组测试<span v-if="kind === 'exam'">
                    · {{ link.score }} 分</span
                  ></small
                ></span
              >
            </button>
          </nav>
          <p v-if="!paper.problems.length" class="muted empty-directory">
            从新增题目或输入已有题号开始。
          </p>
        </aside>
        <section class="paper-main panel">
          <EventProblemEditor
            v-if="editorOpen"
            :key="editing?.problem_id || importedDraft?.key || 'new'"
            :kind="kind"
            :event-id="eventId"
            :revision="paper.event.review_revision"
            :link="editing"
            :initial-draft="importedDraft"
            :source-p-d-f="sourcePDF"
            :next-label="nextLabel()"
            @dirty="dirty = $event"
            @saved="problemSaved"
            @cancel="closeEditor"
          />
          <template v-else>
            <el-tabs v-model="tab"
              ><el-tab-pane label="题目预览" name="problems" /><el-tab-pane
                label="本场附件"
                name="attachments" /><el-tab-pane
                label="说明与时间"
                name="settings" /><el-tab-pane label="审核记录" name="history"
            /></el-tabs>
            <template v-if="tab === 'problems'"
              ><article v-if="selected" class="paper-problem">
                <div class="preview-heading">
                  <div>
                    <span class="eyebrow">题目 {{ selected.label }}</span>
                    <h2>{{ selected.problem.title }}</h2>
                    <p class="muted">{{ selected.case_count }} 组测试</p>
                  </div>
                  <el-button @click="download(selected)"
                    >下载审核题包</el-button
                  >
                </div>
                <div v-if="paper.can_edit" class="actions question-actions">
                  <el-button
                    v-if="
                      selected.problem.event_kind === kind &&
                      selected.problem.event_id === eventId
                    "
                    @click="editProblem(selected)"
                    >编辑题目</el-button
                  ><el-button @click="adoptLatest(selected)"
                    >采用最新版本</el-button
                  ><el-button
                    :disabled="paper.problems.indexOf(selected) === 0"
                    @click="move(paper.problems.indexOf(selected), -1)"
                    >上移</el-button
                  ><el-button
                    :disabled="
                      paper.problems.indexOf(selected) ===
                      paper.problems.length - 1
                    "
                    @click="move(paper.problems.indexOf(selected), 1)"
                    >下移</el-button
                  ><el-button
                    v-if="kind === 'exam'"
                    @click="changeScore(selected)"
                    >分值</el-button
                  ><el-button type="danger" link @click="remove(selected)"
                    >移除</el-button
                  >
                </div>
                <ProblemStatementView
                  embedded
                  :problem="selected.problem"
                  :show-meta="false"
                  :asset-base="eventAssets(kind, eventId, selected.problem_id)"
                />
              </article>
              <el-empty v-else description="本场还没有题目" :image-size="100"
                ><el-button
                  v-if="paper.can_edit"
                  type="primary"
                  @click="editProblem()"
                  >新增题目</el-button
                ></el-empty
              ></template
            >
            <EventAttachments
              v-else-if="tab === 'attachments'"
              :kind="kind"
              :event-id="eventId"
              :revision="paper.event.review_revision"
              :editable="paper.can_edit && !busy"
              @changed="refresh"
            />
            <el-form
              v-else-if="tab === 'settings'"
              label-position="top"
              :disabled="!paper.can_edit"
              class="paper-settings"
              ><el-form-item label="场次名称"
                ><el-input
                  v-model="settings.title"
                  maxlength="200" /></el-form-item
              ><el-form-item label="说明"
                ><el-input
                  v-model="settings.description"
                  type="textarea"
                  :rows="6"
                  maxlength="20000"
              /></el-form-item>
              <div class="actions">
                <el-form-item label="开始时间"
                  ><el-date-picker
                    v-model="settings.starts_at"
                    type="datetime"
                    value-format="YYYY-MM-DDTHH:mm:ssZ" /></el-form-item
                ><el-form-item label="截止时间"
                  ><el-date-picker
                    v-model="settings.ends_at"
                    type="datetime"
                    value-format="YYYY-MM-DDTHH:mm:ssZ"
                /></el-form-item>
              </div>
              <el-button
                v-if="paper.can_edit"
                type="primary"
                @click="saveSettings"
                >保存说明与时间</el-button
              ></el-form
            >
            <section v-else-if="tab === 'history'">
              <el-empty
                v-if="!paper.history?.length"
                description="暂无审核记录"
              />
              <ol v-else class="review-history">
                <li v-for="entry in paper.history" :key="entry.id">
                  <strong>{{
                    actionLabels[entry.action] || entry.action
                  }}</strong
                  ><small>{{ formatDateTime(entry.created_at) }}</small>
                  <p v-if="entry.note">{{ entry.note }}</p>
                </li>
              </ol>
            </section>
          </template>
        </section>
      </div>
      <EventDocumentImport
        v-if="paper.can_edit"
        :key="`${kind}/${eventId}`"
        ref="importer"
        :kind="kind"
        :event-id="eventId"
        :revision="paper.event.review_revision"
        @select="importSelected"
        @pending="importPending = $event"
      /> </template
    ><el-skeleton v-else-if="!error" :rows="8" animated />
  </section>
</template>
<script setup lang="ts">
import {
  computed,
  reactive,
  ref,
  shallowRef,
  watch,
  onBeforeUnmount,
} from "vue";
import { onBeforeRouteLeave, onBeforeRouteUpdate } from "vue-router";
import { ElMessageBox } from "element-plus";
import { generatedApi, type Problem } from "../../api/client";
import { apiErrorMessage } from "../../api/errors";
import {
  downloadPaperProblem,
  eventAssets,
  eventPath,
  loadPaper,
  reviewLabels,
  eventLifecycle,
  lifecycleLabels,
  type EventKind,
  type Paper,
  type PaperProblem,
} from "../../api/modules/events";
import { useAuthStore } from "../../stores/auth";
import { formatDateTime } from "../../features/time";
import EventAttachments from "./EventAttachments.vue";
import EventProblemEditor from "./EventProblemEditor.vue";
import ProblemStatementView from "../ProblemStatementView.vue";
import EventDocumentImport from "./EventDocumentImport.vue";
import type { ImportedProblem } from "../../features/events/documentImport";
import type { PDFImportSession } from "../../features/events/pdfImport";
const importer = ref<InstanceType<typeof EventDocumentImport>>();
const importedDraft = ref<ImportedProblem>();
const sourcePDF = shallowRef<PDFImportSession>();
const importPending = ref(false);
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
const tab = ref("problems"),
  dirty = ref(false);
const lifecycle = computed(
  () =>
    paper.value?.lifecycle ||
    (paper.value ? eventLifecycle(paper.value.event) : "draft"),
);
const selected = computed(
  () =>
    paper.value?.problems.find((p) => p.problem_id === preview.value) ||
    paper.value?.problems[0],
);
async function discard() {
  if (!dirty.value) return true;
  try {
    await ElMessageBox.confirm(
      "本题有尚未保存的修改，离开将丢弃这些修改。",
      "离开题目编辑",
      { confirmButtonText: "丢弃并离开", cancelButtonText: "继续编辑" },
    );
    dirty.value = false;
    return true;
  } catch {
    return false;
  }
}
async function selectProblem(link: PaperProblem) {
  if (!(await discard())) return;
  editorOpen.value = false;
  preview.value = link.problem_id;
  tab.value = "problems";
}
async function closeEditor() {
  if (await discard()) editorOpen.value = false;
}
function beforeUnload(event: BeforeUnloadEvent) {
  if (dirty.value || importPending.value) {
    event.preventDefault();
    event.returnValue = "";
  }
}
window.addEventListener("beforeunload", beforeUnload);
onBeforeUnmount(() => window.removeEventListener("beforeunload", beforeUnload));
async function leaveWorkspace() {
  if (!(await discard())) return false;
  if (!importPending.value) return true;
  try {
    await ElMessageBox.confirm(
      "导入队列中还有未保存的题目，离开将清空队列。",
      "离开组题",
      { confirmButtonText: "离开", cancelButtonText: "继续组题" },
    );
    return true;
  } catch {
    return false;
  }
}
onBeforeRouteLeave(leaveWorkspace);
onBeforeRouteUpdate(leaveWorkspace);
const actionLabels: Record<string, string> = {
  submit: "提交审核",
  approve: "批准",
  reject: "退回",
  withdraw: "撤回",
  delete: "删除",
  complete: "确认完结",
  reopen: "撤销完结",
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
    if (!result.can_edit) importPending.value = false;
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
async function review(
  action: "submit" | "withdraw" | "approve" | "reject" | "complete" | "reopen",
) {
  if (!paper.value) return;
  let note = "";
  try {
    if (action === "reject" || action === "reopen") {
      const answer = await ElMessageBox.prompt(
        action === "reopen"
          ? "撤销后归档将隐藏，并恢复为已截止；不会重新开放答题。请填写原因。"
          : "请填写整场退回原因",
        actionLabels[action],
        {
          confirmButtonText: actionLabels[action],
          cancelButtonText: "取消",
          inputValidator: (value) => !!value?.trim() || "退回原因不能为空",
        },
      );
      note = answer.value;
    } else {
      await ElMessageBox.confirm(
        action === "complete"
          ? "已核对题目、附件和最终成绩？确认后本场以已完结状态进入前台比赛归档，访问权限保持原范围。"
          : action === "approve"
            ? "请确认本场题面、测试点、时间和附件均已核对。批准后参与者将按开始时间获得访问权限。"
            : action === "withdraw"
              ? "撤回后参与者无法进入，修改完成后须重新审核。"
              : importPending.value
                ? "导入队列还有未保存的题目，它们不会包含在本次审核中，提交后队列将清空。确认仅提交已保存的题目、设置和附件？"
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
function nextLabel() {
  const used = new Set(paper.value?.problems.map((p) => p.label.toUpperCase()));
  let i = 0;
  while (used.has(labelAt(i))) i++;
  return labelAt(i);
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
      label: nextLabel(),
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
async function openImport(type?: "markdown" | "pdf") {
  if (!(await discard())) return;
  editorOpen.value = false;
  importer.value?.open(type);
}
function importSelected(draft: ImportedProblem, source?: PDFImportSession) {
  editing.value = undefined;
  importedDraft.value = draft;
  sourcePDF.value = source;
  dirty.value = true;
  editorOpen.value = true;
}
async function editProblem(link?: PaperProblem) {
  if (!(await discard())) return;
  tab.value = "problems";
  dirty.value = false;
  importedDraft.value = undefined;
  sourcePDF.value = undefined;
  editing.value = link;
  editorOpen.value = true;
}
async function problemSaved() {
  if (importedDraft.value) importer.value?.markSaved(importedDraft.value.key);
  importedDraft.value = undefined;
  sourcePDF.value = undefined;
  dirty.value = false;
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
.import-trigger {
  width: 100%;
  margin: 4px 0 16px;
}
.queue-trigger {
  width: 100%;
  margin: -4px 0 16px !important;
}
.paper-workspace {
  display: grid;
  gap: 18px;
  min-width: 0;
}
.paper-header,
.paper-toolbar,
.preview-heading,
.rail-title,
.actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  flex-wrap: wrap;
}
.paper-header h1 {
  margin: 7px 0;
  font-size: clamp(22px, 3vw, 30px);
  overflow-wrap: anywhere;
}
.paper-header p,
.muted {
  color: var(--muted);
  line-height: 1.65;
  margin: 0;
}
.eyebrow {
  font-size: 11px;
  letter-spacing: 0.12em;
  color: var(--muted);
  font-weight: 700;
}
.paper-toolbar {
  padding: 4px 0;
}
.paper-toolbar strong {
  font-size: 22px;
}
.actions {
  justify-content: flex-start;
  gap: 8px;
}
.actions :deep(.el-button + .el-button) {
  margin-left: 0;
}
.lifecycle-track {
  display: flex;
  list-style: none;
  padding: 0;
  margin: 0;
  border-bottom: 1px solid var(--border);
  overflow-x: auto;
}
.lifecycle-track li {
  padding: 11px 20px;
  white-space: nowrap;
  color: var(--muted);
  font-size: 13px;
  border-bottom: 2px solid transparent;
}
.lifecycle-track .current {
  color: var(--accent);
  border-color: var(--accent);
  font-weight: 700;
}
.paper-layout {
  display: grid;
  grid-template-columns: 250px minmax(0, 1fr);
  gap: 22px;
  align-items: start;
}
.paper-rail {
  position: sticky;
  top: 18px;
  min-width: 0;
}
.rail-title h2 {
  font-size: 14px;
  margin: 0;
}
.existing-picker {
  display: grid;
  gap: 8px;
  padding: 16px 0;
  border-bottom: 1px solid var(--border);
}
.problem-directory {
  display: grid;
  gap: 4px;
  margin-top: 12px;
  max-height: 65vh;
  overflow: auto;
}
.problem-directory button {
  border: 0;
  background: transparent;
  color: var(--text);
  text-align: left;
  display: flex;
  gap: 10px;
  padding: 12px 8px;
  border-radius: 8px;
  cursor: pointer;
  width: 100%;
  align-items: flex-start;
}
.problem-directory button:hover,
.problem-directory button.selected {
  background: var(--surface-strong);
}
.problem-directory button.selected {
  box-shadow: inset 3px 0 var(--accent);
}
.problem-directory button:focus-visible {
  outline: 2px solid var(--accent);
}
.letter {
  font-size: 13px;
  font-weight: 700;
  color: var(--accent);
  width: 24px;
  flex-shrink: 0;
}
.problem-directory strong {
  font-size: 13px;
  overflow-wrap: anywhere;
  display: block;
  line-height: 1.5;
}
.problem-directory small {
  color: var(--muted);
  display: block;
  margin-top: 5px;
}
.empty-directory {
  font-size: 13px;
  padding: 22px 4px;
}
.paper-main {
  padding: 24px;
  min-width: 0;
  min-height: 520px;
}
.paper-problem {
  display: grid;
  gap: 16px;
}
.preview-heading h2 {
  font-size: 23px;
  overflow-wrap: anywhere;
  margin: 6px 0;
}
.question-actions {
  border-bottom: 1px solid var(--border);
  padding-bottom: 15px;
}
.paper-settings {
  max-width: 740px;
  padding: 12px 0;
}
.review-history {
  padding: 8px 0;
  list-style: none;
}
.review-history li {
  padding: 14px 0;
  border-bottom: 1px solid var(--border);
}
.review-history small {
  display: block;
  color: var(--muted);
  margin-top: 6px;
}
.review-history p {
  overflow-wrap: anywhere;
}
.paper-main :deep(.event-attachments) {
  border: 0;
  box-shadow: none;
  padding: 8px 0;
}
@media (max-width: 1100px) {
  .paper-layout {
    grid-template-columns: 220px minmax(0, 1fr);
    gap: 14px;
  }
  .paper-main {
    padding: 18px;
  }
}
@media (max-width: 760px) {
  .paper-layout {
    grid-template-columns: 1fr;
  }
  .paper-rail {
    position: static;
  }
  .problem-directory {
    max-height: 220px;
  }
  .paper-main {
    padding: 14px;
    min-height: 360px;
  }
  .lifecycle-track li {
    padding: 10px 14px;
  }
  .paper-toolbar {
    align-items: flex-start;
    gap: 12px;
  }
  .paper-toolbar .actions {
    width: 100%;
  }
  .paper-settings :deep(.el-date-editor) {
    max-width: 100%;
  }
}
</style>
