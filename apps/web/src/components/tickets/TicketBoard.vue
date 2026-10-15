<template>
  <section class="ticket-board" v-loading="loading">
    <header class="board-toolbar">
      <div>
        <h2>
          {{
            mode === "new"
              ? "发起工单"
              : mode === "edit"
                ? "编辑工单"
                : active
                  ? `工单 #${active.id}`
                  : title
          }}
        </h2>
        <p v-if="context.kind === 'exam' || context.kind === 'contest'">
          工单仅本人和所属管理者可见，反馈不会改变当前场次的题目版本。
        </p>
        <p v-else-if="context.kind === 'problem'">
          工单正文在题目的可见范围内公开；附件保持私有。
        </p>
      </div>
      <div class="board-actions">
        <el-button v-if="mode !== 'list'" @click="back">返回列表</el-button
        ><el-button v-if="mode === 'list' || mode === 'view'" @click="refresh"
          >刷新</el-button
        ><el-button
          v-if="mode === 'list' && canCreate"
          type="primary"
          @click="create"
          >{{ context.kind === "mine" ? "新增题目" : "提交反馈" }}</el-button
        >
      </div>
    </header>
    <el-alert
      v-if="loadError"
      :title="loadError"
      type="error"
      :closable="false"
      class="load-error"
    />
    <TicketEditor
      v-if="mode === 'new' || mode === 'edit'"
      :key="`${mode}-${active?.id || 0}`"
      :context="context"
      :ticket="mode === 'edit' ? active || undefined : undefined"
      @cancel="back"
      @saved="saved"
    />
    <template v-else-if="mode === 'view' && active">
      <div v-if="active.can_edit" class="own-actions">
        <el-button @click="mode = 'edit'">{{ active.status === 'awaiting_changes' ? '补充材料并重提' : '编辑工单' }}</el-button
        ><el-button type="danger" plain @click="remove">删除工单</el-button>
      </div>
      <TicketDetail
        :key="`${active.id}-${active.revision}`"
        :ticket="active"
        :context="context"
        @changed="refresh"
      />
    </template>
    <template v-else>
      <div class="ticket-filters">
        <el-select
          v-model="filters.action"
          aria-label="筛选工单类型"
          clearable
          placeholder="全部类型"
          @change="filterChanged"
          ><el-option
            v-for="(label, value) in ticketActionLabels"
            :key="value"
            :label="label"
            :value="value"
        /></el-select>
        <el-select
          v-model="filters.status"
          aria-label="筛选工单状态"
          clearable
          placeholder="全部状态"
          @change="filterChanged"
          ><el-option
            v-for="(label, value) in ticketStatusLabels"
            :key="value"
            :label="label"
            :value="value"
        /></el-select>
        <el-select
          v-if="context.kind === 'manage'"
          v-model="filters.context"
          aria-label="筛选所属对象"
          clearable
          placeholder="全部对象"
          @change="filterChanged"
          ><el-option label="普通题目" value="problem" /><el-option
            label="考试"
            value="exam" /><el-option label="比赛" value="contest" /><el-option
            label="新增题目"
            value="new"
        /></el-select>
      </div>
      <el-table
        :data="items"
        row-key="id"
        class="ticket-table"
        empty-text="暂无工单"
      >
        <el-table-column label="工单编号" width="100"
          ><template #default="{ row }"
            >#{{ row.id }}</template
          ></el-table-column
        >
        <el-table-column label="工单类型" width="100"
          ><template #default="{ row }">{{
            ticketActionLabels[row.action as ProblemChangeAction]
          }}</template></el-table-column
        >
        <el-table-column label="申请人" width="125"
          ><template #default="{ row }">{{
            row.requester.name
          }}</template></el-table-column
        >
        <el-table-column label="修改说明" min-width="230"
          ><template #default="{ row }"
            ><div class="target-label">{{ targetLabel(row) }}</div>
            <div class="description-excerpt">
              {{ row.description }}
            </div></template
          ></el-table-column
        >
        <el-table-column label="附件" width="110"
          ><template #default="{ row }">{{
            row.attachment_count ? `${row.attachment_count} 个 · 私有` : "无"
          }}</template></el-table-column
        >
        <el-table-column label="状态" width="100"
          ><template #default="{ row }"
            ><el-tag
              :type="
                row.status === 'completed'
                  ? 'success'
                  : row.status === 'rejected'
                    ? 'danger'
                    : 'info'
              "
              >{{
                ticketStatusLabels[row.status as ProblemChangeStatus]
              }}</el-tag
            ></template
          ></el-table-column
        >
        <el-table-column label="提交时间" width="170"
          ><template #default="{ row }">{{
            formatDateTime(row.created_at)
          }}</template></el-table-column
        >
        <el-table-column label="操作" width="80" fixed="right"
          ><template #default="{ row }"
            ><el-button link type="primary" @click="open(row)"
              >查看</el-button
            ></template
          ></el-table-column
        >
      </el-table>
      <el-pagination
        v-model:current-page="page"
        :page-size="20"
        :total="total"
        layout="prev, pager, next"
        :pager-count="5"
        @current-change="load"
      />
    </template>
  </section>
</template>
<script setup lang="ts">
import { apiErrorMessage } from '../../api/errors'
import { computed, onBeforeUnmount, reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { useRoute, useRouter } from "vue-router";
import type {
  ProblemChangeTicket,
  ProblemChangeAction,
  ProblemChangeStatus,
} from "../../api/client";
import {
  listTickets,
  ticketRequest,
  type TicketContext,
} from "../../api/modules/tickets";
import {
  ticketActionLabels,
  ticketStatusLabels,
} from "../../features/problems/changeTicket";
import { formatDateTime } from "../../features/time";
import { useAuthStore } from "../../stores/auth";
import TicketEditor from "./TicketEditor.vue";
import TicketDetail from "./TicketDetail.vue";
const props = withDefaults(
  defineProps<{
    context: TicketContext;
    initialNew?: boolean;
    newBackPath?: string;
    createdPath?: string;
    title?: string;
  }>(),
  { title: "工单管理" },
);
const auth = useAuthStore(),
  route = useRoute(),
  router = useRouter();
const mode = ref<"list" | "new" | "view" | "edit">("list"),
  active = ref<ProblemChangeTicket | null>(null),
  items = ref<ProblemChangeTicket[]>([]);
const page = ref(1),
  total = ref(0),
  loading = ref(false),
  loadError = ref(""),
  filters = reactive({ action: "", status: "", context: "" });
const canCreate = computed(
  () =>
    props.context.kind !== "manage" &&
    (props.context.kind !== "mine" ||
      auth.user?.can_author ||
      auth.role === "admin"),
);
let sequence = 0,
  pending: AbortController | undefined;
function nextRequest() {
  pending?.abort();
  pending = new AbortController();
  return { generation: ++sequence, signal: pending.signal };
}
async function load() {
  const { generation, signal } = nextRequest();
  loading.value = true;
  loadError.value = "";
  try {
    const { data } = await listTickets(
      props.context,
      { ...filters, page: page.value, page_size: 20 },
      signal,
    );
    if (generation === sequence && !signal.aborted) {
      items.value = data.items;
      total.value = data.total;
    }
  } catch (error: any) {
    if (generation === sequence)
      loadError.value =
        apiErrorMessage(error, "工单加载失败，请刷新重试");
  } finally {
    if (generation === sequence) loading.value = false;
  }
}
async function open(ticket: ProblemChangeTicket) {
  const { generation, signal } = nextRequest();
  loading.value = true;
  loadError.value = "";
  try {
    const { data } = await ticketRequest(props.context, "get", ticket.id, {
      signal,
    });
    if (generation === sequence) {
      active.value = data;
      mode.value = "view";
    }
  } catch (error: any) {
    if (generation === sequence)
      loadError.value = apiErrorMessage(error, "工单详情加载失败");
  } finally {
    if (generation === sequence) loading.value = false;
  }
}
function refresh() {
  if (mode.value === "view" && active.value) void open(active.value);
  else void load();
}
function create() {
  if (!auth.isAuthed) {
    void router.push({ path: "/login", query: { redirect: route.fullPath } });
    return;
  }
  if (!canCreate.value) return;
  pending?.abort();
  sequence++;
  loading.value = false;
  active.value = null;
  mode.value = "new";
}
function back() {
  if (mode.value === "new" && props.initialNew && props.newBackPath) {
    void router.replace(props.newBackPath);
    return;
  }
  mode.value = "list";
  active.value = null;
  void load();
}
function saved(ticket: ProblemChangeTicket) {
  if (mode.value === "new" && props.createdPath) {
    if (route.path !== props.createdPath) void router.replace(props.createdPath);
    else back();
    return;
  }
  active.value = ticket;
  mode.value = "view";
}
function filterChanged() {
  page.value = 1;
  void load();
}
function targetLabel(ticket: ProblemChangeTicket) {
  return [
    ticket.exam_id
      ? `考试 #${ticket.exam_id}`
      : ticket.contest_id
        ? `比赛 #${ticket.contest_id}`
        : "",
    ticket.problem?.title ||
      (ticket.action === "create" ? "新增题目" : "场次整体"),
  ]
    .filter(Boolean)
    .join(" · ");
}
async function remove() {
  if (!active.value) return;
  try {
    await ElMessageBox.confirm(
      "删除这张待处理工单？题目本身不会被删除。",
      "删除工单",
      {
        type: "warning",
        confirmButtonText: "删除工单",
        cancelButtonText: "取消",
      },
    );
  } catch {
    return;
  }
  try {
    await ticketRequest(props.context, "delete", active.value.id, {
      body: { revision: active.value.revision },
    });
    ElMessage.success("工单已删除");
    back();
  } catch (error: any) {
    ElMessage.error(apiErrorMessage(error, "删除失败"));
  }
}
watch(
  () => JSON.stringify([props.context, props.initialNew]),
  () => {
    page.value = 1;
    active.value = null;
    mode.value = "list";
    if (props.initialNew && canCreate.value) create();
    else void load();
  },
  { immediate: true },
);
onBeforeUnmount(() => {
  sequence++;
  pending?.abort();
});
</script>
<style scoped>
.ticket-board {
  min-width: 0;
  width: 100%;
}
.board-toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}
.board-toolbar h2 {
  margin: 0;
  font-size: 22px;
}
.board-toolbar p {
  margin: 8px 0 0;
  line-height: 1.6;
  color: var(--muted);
  font-size: 13px;
}
.board-actions,
.ticket-filters,
.own-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.board-actions {
  flex-shrink: 0;
}
.ticket-filters,
.own-actions,
.load-error {
  margin-bottom: 18px;
}
.ticket-filters .el-select {
  width: 155px;
}
.target-label {
  font-size: 12px;
  color: var(--muted);
  margin-bottom: 4px;
}
.description-excerpt {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.el-pagination {
  justify-content: flex-end;
  margin-top: 18px;
}
@media (max-width: 720px) {
  .board-toolbar {
    flex-direction: column;
  }
  .board-actions {
    width: 100%;
  }
  .ticket-filters {
    display: grid;
    grid-template-columns: 1fr 1fr;
  }
  .ticket-filters .el-select {
    width: 100%;
  }
}
</style>
