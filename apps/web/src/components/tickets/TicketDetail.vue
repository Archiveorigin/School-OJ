<template>
  <div class="ticket-detail">
    <el-descriptions :column="1" border>
      <el-descriptions-item label="工单编号"
        >#{{ ticket.id }} ·
        {{ ticketActionLabels[ticket.action] }}</el-descriptions-item
      >
      <el-descriptions-item label="申请人">{{
        ticket.requester.name
      }}</el-descriptions-item>
      <el-descriptions-item label="所属对象">{{
        contextLabel
      }}</el-descriptions-item>
      <el-descriptions-item label="状态">{{
        ticketStatusLabels[ticket.status]
      }}</el-descriptions-item>
      <el-descriptions-item v-if="ticket.action === 'create'" label="入库范围">{{ { public: '公共题库', prepared: '预备题库', team_problem_set: '团队题单' }[ticket.target_scope] }}<span v-if="ticket.team_problem_set_id"> #{{ ticket.team_problem_set_id }}</span></el-descriptions-item>
      <el-descriptions-item label="提交时间">{{
        formatDateTime(ticket.created_at)
      }}</el-descriptions-item>
      <el-descriptions-item label="修改说明"
        ><div class="plain-text">
          {{ ticket.description }}
        </div></el-descriptions-item
      >
      <el-descriptions-item label="附件">
        <span v-if="!ticket.attachment_count">无附件</span
        ><span v-else-if="ticket.attachments_private"
          >附件仅申请人和有权管理者可见</span
        >
        <div
          v-for="file in ticket.attachments"
          :key="file.id"
          class="attachment-row"
        >
          <span>{{ file.name }}</span
          ><el-button link type="primary" @click="download(file)"
            >下载</el-button
          ><el-button
            v-if="file.content_type.startsWith('image/')"
            link
            @click="preview(file)"
            >预览图片</el-button
          >
        </div>
      </el-descriptions-item>
      <el-descriptions-item v-if="ticket.resolution_note" label="处理结果"
        ><div class="plain-text">
          {{ ticket.resolution_note }}
        </div></el-descriptions-item
      >
    </el-descriptions>
    <div v-if="previewUrl" class="image-preview">
      <el-button link @click="previewUrl = ''">关闭图片预览</el-button
      ><img :src="previewUrl" alt="工单附件预览" />
    </div>
    <ProblemDraftPreview v-if="ticket.action === 'create' && ticket.draft_statement && !(ticket.can_apply && ticket.status === 'pending')" class="draft-preview" :title="ticket.draft_title || ''" :source="ticket.draft_statement" :asset-urls="imageUrls" />
    <el-alert v-if="ticket.status === 'awaiting_changes'" type="warning" :closable="false" title="工单已退回补充，请根据处理说明修改材料后重新提交。" />
    <router-link v-if="ticket.status === 'completed' && ticket.applied_version_id && ticket.problem_id" :to="`/problems/${ticket.problem_id}`">查看已入库题目：{{ ticket.problem?.title || ticket.draft_title }}</router-link>
    <el-collapse v-if="ticket.history?.length"><el-collapse-item title="处理记录" name="history"><ol class="ticket-history"><li v-for="item in ticket.history" :key="item.revision"><strong>{{ historyLabels[item.action] || item.action }}</strong> · {{ item.actor }} · {{ formatDateTime(item.at) }}<p v-if="item.note" class="plain-text">{{ item.note }}</p></li></ol></el-collapse-item></el-collapse>
    <section
      v-if="ticket.can_apply && ticket.impact_summary"
      class="impact-panel"
    >
      <h3>执行影响</h3>
      <p>
        题库修改生成新版本。所有已组题场次和历史提交保留原版本；赛前须撤回整场审核、明确采用新版本后重新送审。
      </p>
      <p>
        未开始考试 {{ ticket.impact_summary.future_exams }} · 已固定考试
        {{ ticket.impact_summary.pinned_exams }} · 未开始比赛
        {{ ticket.impact_summary.future_contests }} · 已固定比赛
        {{ ticket.impact_summary.pinned_contests }}
      </p>
      <p v-if="ticket.current_version_id">
        当前题目版本：#{{ ticket.current_version_id
        }}<span v-if="ticket.problem_version_id"
          >；反馈时版本：#{{ ticket.problem_version_id }}</span
        >
      </p>
    </section>
    <el-form
      v-if="ticket.can_manage && ticket.status === 'pending'"
      label-position="top"
      class="decision-form"
      :disabled="working"
    >
      <template v-if="ticket.can_apply && ticket.action !== 'archive'">
        <el-form-item label="入库材料"><el-radio-group v-model="packageMode" :disabled="working"><el-radio-button value="materials">使用工单材料</el-radio-button><el-radio-button value="zip">使用完整 ZIP 题包</el-radio-button></el-radio-group></el-form-item>
        <TicketPackageReview v-if="packageMode === 'materials'" ref="packageReview" :ticket="ticket" :image-urls="imageUrls" :disabled="working" />
        <el-form-item v-else label="完整 ZIP 题包（可选方式）"
          ><el-upload
            :auto-upload="false"
            accept=".zip"
            :limit="1"
            :on-change="pickPackage"
            :on-remove="clearPackage"
            ><el-button>选择完整 ZIP 题包</el-button
            ><template #tip
              ><p>
                此方式使用完整题包中的题面、图片和测试点，覆盖本次入库内容。普通附件请使用“工单材料”。
              </p></template
            ></el-upload
          ></el-form-item
        >
        <el-form-item label="难度"
          ><el-select v-model="difficulty" aria-label="题目难度" clearable
            ><el-option
              v-for="value in problemDifficultyOptions"
              :key="value"
              :label="value"
              :value="value" /></el-select
        ></el-form-item>
        <el-form-item label="算法标签"><ProblemTagSelector v-model="tags" /></el-form-item>
      </template>
      <el-alert v-if="ticket.can_resolve && (ticket.exam_id || ticket.contest_id)" type="info" :closable="false" title="填写处置结果后可直接结单，不更改题库、不更新当前场次版本，也不触发重判。题库修正需单独执行。" />
      <el-form-item label="处理说明"
        ><el-input
          v-model="note"
          type="textarea"
          :rows="3"
          maxlength="10000"
          placeholder="填写处理结果或驳回原因"
      /></el-form-item>
      <div class="decision-actions">
        <el-button
          type="danger"
          plain
          :loading="working"
          @click="decide('reject')"
          >驳回</el-button
        ><el-button :disabled="working" @click="decide('return')">退回补充</el-button
        ><el-button
          v-if="ticket.can_resolve"
          type="primary"
          :loading="working"
          @click="decide('resolve')"
          >{{ ticket.exam_id || ticket.contest_id ? '记录处置并结单' : '处理完成' }}</el-button
        ><el-button
          v-if="ticket.can_apply"
          type="primary"
          :loading="working"
          @click="apply"
          >{{
            ticket.action === "archive" ? "归档题目并完成" : "执行并完成"
          }}</el-button
        ><span v-if="!ticket.can_apply && !ticket.can_resolve" class="muted"
          >题库变更由系统管理员执行</span
        >
      </div>
    </el-form>
  </div>
</template>
<script setup lang="ts">
import { apiErrorMessage } from '../../api/errors'
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { ElMessage, ElMessageBox, type UploadFile } from "element-plus";
import {
  generatedApi,
  type ProblemChangeTicket,
  type TicketAttachment,
} from "../../api/client";
import { ticketRequest, type TicketContext } from "../../api/modules/tickets";
import {
  ticketActionLabels,
  ticketStatusLabels,
  problemChangeOperationMode,
} from "../../features/problems/changeTicket";
import { problemDifficultyOptions } from "../../features/problems/problemMeta";
import { formatDateTime } from "../../features/time";
import { mapConcurrent } from "../../features/async";
import ProblemDraftPreview from "./ProblemDraftPreview.vue";
import TicketPackageReview from "./TicketPackageReview.vue";
import ProblemTagSelector from "../ProblemTagSelector.vue";
const props = defineProps<{
  ticket: ProblemChangeTicket;
  context: TicketContext;
}>();
const emit = defineEmits<{ changed: [] }>();
const packageMode = ref<'materials' | 'zip'>('materials');
const packageReview = ref<InstanceType<typeof TicketPackageReview>>();
const tags = ref<string[]>([...(props.ticket.review?.tags || [])]);
const historyLabels: Record<string, string> = { return: '退回补充', resubmit: '补充后重新提交', update: '修改材料', reject: '驳回', resolve: '记录处置并完成', apply: '执行题库变更', cancel: '撤回' };
const note = ref(""),
  difficulty = ref(props.ticket.problem?.difficulty || ""),
  packageFile = ref<File | null>(null),
  working = ref(false);
const previewUrl = ref(""),
  imageUrls = ref<Record<string, string>>({});
const urls = new Set<string>(),
  controller = new AbortController();
const contextLabel = computed(() =>
  [
    props.ticket.exam_id
      ? `考试 #${props.ticket.exam_id}`
      : props.ticket.contest_id
        ? `比赛 #${props.ticket.contest_id}`
        : "",
    props.ticket.problem?.title ||
      (props.ticket.action === "create" ? "新增题目" : "场次整体"),
  ]
    .filter(Boolean)
    .join(" · "),
);
function pickPackage(file: UploadFile) {
  packageFile.value = file.raw || null;
}
function clearPackage() {
  packageFile.value = null;
}
async function attachment(file: TicketAttachment, inline = false) {
  const { data } = await ticketRequest<Blob>(
    props.context,
    "attachment",
    props.ticket.id,
    {
      path: { attachment_id: file.id },
      query: inline ? { preview: "1" } : undefined,
      responseType: "blob",
      signal: controller.signal,
    },
  );
  if (controller.signal.aborted) throw new Error("cancelled");
  const url = URL.createObjectURL(data);
  urls.add(url);
  return url;
}
async function download(file: TicketAttachment) {
  try {
    const url = await attachment(file);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = file.name;
    anchor.click();
  } catch {
    if (!controller.signal.aborted) ElMessage.error("附件下载失败或权限已变化");
  }
}
async function preview(file: TicketAttachment) {
  try {
    previewUrl.value = await attachment(file, true);
  } catch {
    if (!controller.signal.aborted) ElMessage.error("图片预览不可用");
  }
}
watch(
  () => props.ticket.id,
  async () => {
    if (!props.ticket.draft_statement) return;
    const loaded: Record<string, string> = {};
    await mapConcurrent((props.ticket.attachments || []).filter(file => file.content_type.startsWith('image/')), 3, async file => {
      if (!controller.signal.aborted) {
        try {
          const url = await attachment(file, true);
          loaded[`attachments/${encodeURIComponent(file.name)}`] = url;
        } catch {
          /* Private assets have no public fallback. */
        }
      }
    });
    if (!controller.signal.aborted) imageUrls.value = loaded;
  },
  { immediate: true },
);
onBeforeUnmount(() => {
  controller.abort();
  urls.forEach(URL.revokeObjectURL);
});
async function decide(action: "reject" | "resolve" | "return") {
  if (working.value) return;
  if (!note.value.trim())
    return void ElMessage.warning("请填写处理结果或驳回原因");
  working.value = true;
  try {
    await ticketRequest(props.context, action, props.ticket.id, {
      body: { revision: props.ticket.revision, note: note.value.trim() },
    });
    ElMessage.success(action === "return" ? "已退回作者补充" : action === "reject" ? "工单已驳回" : "工单已完成");
    emit("changed");
  } catch (error: any) {
    ElMessage.error(apiErrorMessage(error, "处理失败"));
  } finally {
    working.value = false;
  }
}
async function apply() {
  if (working.value) return;
  const data = new FormData();
  if (props.ticket.action !== 'archive') {
    if (packageMode.value === 'materials') { if (!packageReview.value?.appendTo(data)) return; }
    else if (!packageFile.value) return void ElMessage.warning('请选择完整 ZIP 题包');
  }
  try {
    await ElMessageBox.confirm(
      props.ticket.action === "archive"
        ? "确认归档题目？历史场次和提交将保留。"
        : "确认执行题库变更？已开始场次和历史提交保持原版本。",
      "确认执行",
      {
        type: "warning",
        confirmButtonText: "确认执行",
        cancelButtonText: "取消",
      },
    );
  } catch {
    return;
  }
  working.value = true;
  try {
    data.append("revision", String(props.ticket.revision));
    if (props.ticket.current_version_id)
      data.append(
        "expected_version_id",
        String(props.ticket.current_version_id),
      );
    data.append(
      "operation_mode",
      problemChangeOperationMode(props.ticket.action),
    );
    data.append("resolution_note", note.value.trim());
    data.append("difficulty", difficulty.value);
    data.append("tags", tags.value.join(','));
    if (packageMode.value === 'zip' && packageFile.value) data.append("package", packageFile.value);
    await generatedApi.applyProblemChangeTicket({
      path: { id: props.ticket.id },
      body: data,
      timeout: 600000,
    });
    ElMessage.success("工单已执行");
    emit("changed");
  } catch (error: any) {
    ElMessage.error(apiErrorMessage(error, "执行失败"));
  } finally {
    working.value = false;
  }
}
</script>
<style scoped>
.ticket-detail {
  display: grid;
  gap: 20px;
  min-width: 0;
}
.plain-text {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.ticket-history { display: grid; gap: 12px; padding-left: 24px; }
.ticket-history p { margin: 6px 0; }
.decision-form > .el-alert { margin: 16px 0; }
.decision-form .package-review { margin-bottom: 20px; }
.attachment-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.attachment-row span {
  overflow-wrap: anywhere;
  min-width: 0;
}
.image-preview img {
  display: block;
  max-width: 100%;
  max-height: 500px;
  margin-top: 10px;
}
.draft-preview,
.impact-panel {
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: 8px;
}
.impact-panel p,
.muted {
  color: var(--muted);
  line-height: 1.6;
}
.decision-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.decision-form .el-select {
  width: 200px;
}
.ticket-detail :deep(.el-descriptions__label) {
  width: 100px;
}
</style>
