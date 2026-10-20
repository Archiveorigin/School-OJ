<template>
  <section class="competition-list">
    <header class="catalog-header">
      <div>
        <span class="eyebrow">{{
          reviewMode ? "EVENT REVIEW" : "COMPETITION ARCHIVE"
        }}</span>
        <h1>{{ reviewMode ? "考试与比赛审核" : "比赛" }}</h1>
        <p>
          {{
            reviewMode
              ? "赛前核对整场试卷，截止后确认成绩并发布归档。"
              : "查看经管理员确认完结的考试与比赛，回顾本场题目、附件和成绩。"
          }}
        </p>
      </div>
      <el-tag v-if="!reviewMode" effect="plain">沿用原班级与团队权限</el-tag>
    </header>
    <section class="panel filter-card">
      <form class="filters" @submit.prevent="search">
        <el-input
          v-model="query.q"
          clearable
          placeholder="搜索场次名称"
          aria-label="搜索场次名称"
        /><el-select
          v-model="query.kind"
          aria-label="场次类型"
          placeholder="全部类型"
          @change="search"
          ><el-option label="全部类型" value="" /><el-option
            label="考试"
            value="exam" /><el-option
            label="团队比赛"
            value="contest" /></el-select
        ><el-select
          v-if="reviewMode"
          v-model="query.status"
          aria-label="场次状态"
          placeholder="全部状态"
          @change="search"
          ><el-option label="全部状态" value="" /><el-option
            v-for="(label, value) in lifecycleLabels"
            :key="value"
            :label="label"
            :value="value"
        /></el-select>
        <div class="filter-actions">
          <el-button type="primary" native-type="submit" :loading="loading"
            >查询</el-button
          ><el-button @click="reset">重置</el-button>
        </div>
      </form>
    </section>
    <el-alert v-if="error" type="error" :title="error" :closable="false" />
    <section class="panel results" v-loading="loading">
      <div class="result-heading">
        <strong>{{ reviewMode ? "场次列表" : "已完结场次" }}</strong
        ><span>{{ total }} 场</span
        ><el-button
          v-if="reviewMode"
          link
          :type="query.status === 'closed' ? 'primary' : 'default'"
          @click="
            query.status = 'closed';
            search();
          "
          >查看待完结场次</el-button
        >
      </div>
      <el-table
        :data="items"
        class="desktop-table"
        empty-text="暂无符合条件的场次"
        ><el-table-column label="类型" width="115"
          ><template #default="{ row }"
            ><el-tag size="small" effect="plain">{{
              row.kind === "exam" ? "考试" : "团队比赛"
            }}</el-tag></template
          ></el-table-column
        ><el-table-column label="场次名称" min-width="260"
          ><template #default="{ row }"
            ><router-link class="event-link" :to="target(row)">{{
              row.title
            }}</router-link
            ><small>{{ row.context_name }}</small></template
          ></el-table-column
        ><el-table-column label="题目" width="90"
          ><template #default="{ row }"
            >{{ row.problem_count }} 道</template
          ></el-table-column
        ><el-table-column label="赛制" width="85"
          ><template #default="{ row }">{{
            row.scoring_rule.toUpperCase()
          }}</template></el-table-column
        ><el-table-column label="状态" width="110"
          ><template #default="{ row }"
            ><el-tag
              :type="
                row.lifecycle === 'completed'
                  ? 'success'
                  : row.lifecycle === 'closed'
                    ? 'warning'
                    : 'info'
              "
              >{{
                lifecycleLabels[
                  row.lifecycle as keyof typeof lifecycleLabels
                ] || lifecycleLabels[eventLifecycle(row)]
              }}</el-tag
            ></template
          ></el-table-column
        ><el-table-column
          :label="reviewMode ? '开始时间' : '截止时间'"
          min-width="175"
          ><template #default="{ row }">{{
            formatDateTime(reviewMode ? row.starts_at : row.ends_at)
          }}</template></el-table-column
        ><el-table-column label="操作" width="115" align="right"
          ><template #default="{ row }"
            ><router-link class="event-link" :to="target(row)"
              >{{ reviewMode ? "查看审核" : "查看归档" }} →</router-link
            ></template
          ></el-table-column
        ></el-table
      >
      <div class="mobile-list">
        <router-link
          v-for="item in items"
          :key="`${item.kind}-${item.id}`"
          :to="target(item)"
          class="event-row"
          ><div>
            <el-tag size="small" effect="plain">{{
              item.kind === "exam" ? "考试" : "团队比赛"
            }}</el-tag
            ><span>{{
              lifecycleLabels[item.lifecycle || eventLifecycle(item)]
            }}</span>
          </div>
          <h2>{{ item.title }}</h2>
          <p>{{ item.context_name }}</p>
          <footer>
            <span
              >{{ item.problem_count }} 道题 ·
              {{ item.scoring_rule.toUpperCase() }}</span
            ><span>{{
              formatDateTime(reviewMode ? item.starts_at : item.ends_at)
            }}</span>
          </footer></router-link
        ><el-empty
          v-if="!loading && !items.length"
          description="暂无符合条件的场次"
        />
      </div>
      <div v-if="total" class="pagination">
        <el-pagination
          v-model:current-page="query.page"
          :total="total"
          :page-size="20"
          layout="prev, pager, next"
          :pager-count="5"
          @current-change="load"
        />
      </div>
    </section>
  </section>
</template>
<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import {
  listEvents,
  lifecycleLabels,
  eventLifecycle,
  type EventListItem,
} from "../../api/modules/events";
import { apiErrorMessage } from "../../api/errors";
import { formatDateTime } from "../../features/time";
const props = defineProps<{ reviewMode?: boolean }>();
const query = reactive({
    page: 1,
    q: "",
    kind: "",
    status: "",
  }),
  items = ref<EventListItem[]>([]),
  total = ref(0),
  error = ref(""),
  loading = ref(false);
let sequence = 0;
async function load() {
  const current = ++sequence;
  loading.value = true;
  error.value = "";
  try {
    const result = await listEvents(!props.reviewMode, { ...query });
    if (current === sequence) {
      items.value = result.items;
      total.value = result.total;
    }
  } catch (e) {
    if (current === sequence) {
      items.value = [];
      error.value = apiErrorMessage(e);
    }
  } finally {
    if (current === sequence) loading.value = false;
  }
}
function search() {
  query.page = 1;
  void load();
}
function target(item: EventListItem) {
  return props.reviewMode
    ? `/admin/competitions/${item.kind}/${item.id}`
    : `/competitions/${item.kind}/${item.id}`;
}
function reset() {
  query.q = "";
  query.kind = "";
  query.status = "";
  search();
}
onMounted(load);
</script>
<style scoped>
.competition-list {
  padding: 30px;
  display: grid;
  gap: 22px;
  max-width: 1440px;
  margin: auto;
  min-width: 0;
}
.catalog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 20px;
}
.eyebrow {
  font-size: 11px;
  letter-spacing: 0.14em;
  font-weight: 700;
  color: var(--muted);
}
h1 {
  margin: 8px 0;
  font-size: 30px;
}
.catalog-header p {
  color: var(--muted);
  margin: 0;
  line-height: 1.7;
  font-size: 14px;
}
.filter-card {
  padding: 18px;
}
.filters {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
.filters .el-input {
  flex: 1;
  min-width: 210px;
}
.filters .el-select {
  width: 160px;
}
.filter-actions {
  display: flex;
  gap: 8px;
}
.filter-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}
.results {
  padding: 0 22px 18px;
  min-width: 0;
}
.result-heading {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 22px 0 14px;
}
.result-heading > span {
  font-size: 13px;
  color: var(--muted);
}
.event-link {
  font-weight: 600;
  color: var(--text);
  line-height: 1.6;
  overflow-wrap: anywhere;
}
.event-link:hover {
  color: var(--accent);
}
small {
  display: block;
  color: var(--muted);
  margin-top: 4px;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
}
.mobile-list {
  display: none;
}
.event-row {
  display: block;
  color: var(--text);
  padding: 18px 0;
  border-bottom: 1px solid var(--border);
}
.event-row > div,
.event-row footer {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--muted);
}
.event-row h2 {
  font-size: 17px;
  line-height: 1.5;
  overflow-wrap: anywhere;
  margin: 12px 0 6px;
}
.event-row p {
  color: var(--muted);
  font-size: 13px;
  margin: 0 0 14px;
}
@media (max-width: 700px) {
  .competition-list {
    padding: 18px 14px;
    gap: 16px;
  }
  .catalog-header {
    align-items: flex-start;
    flex-direction: column;
    gap: 12px;
  }
  .filters .el-input {
    width: 100%;
    flex: auto;
  }
  .filters .el-select {
    flex: 1;
    min-width: 120px;
  }
  .filter-actions {
    width: 100%;
    justify-content: flex-end;
  }
  .desktop-table {
    display: none;
  }
  .mobile-list {
    display: block;
  }
  .results {
    padding: 0 16px 16px;
  }
  .result-heading {
    flex-wrap: wrap;
  }
  .pagination {
    justify-content: center;
  }
}
</style>
