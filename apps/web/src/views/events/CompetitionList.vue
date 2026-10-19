<template>
  <section class="competition-list">
    <header>
      <h1>{{ reviewMode ? "考试与比赛审核" : "比赛" }}</h1>
      <p>
        {{
          reviewMode
            ? "按整场核对题目、时间和附件；只有批准的场次可以开始。"
            : "查看已完结考试与比赛的题目、附件和排名，沿用原班级与团队访问权限。"
        }}
      </p>
    </header>
    <form class="filters" @submit.prevent="search">
      <el-input
        v-model="query.q"
        placeholder="搜索场次名称"
        clearable
      /><el-select v-model="query.kind" @change="search"
        ><el-option label="全部类型" value="" /><el-option
          label="考试"
          value="exam" /><el-option
          label="团队比赛"
          value="contest" /></el-select
      ><el-select v-if="reviewMode" v-model="query.status" @change="search"
        ><el-option label="全部审核状态" value="" /><el-option
          v-for="(label, value) in reviewLabels"
          :key="value"
          :value="value"
          :label="label" /></el-select
      ><el-button native-type="submit" :loading="loading">查询</el-button>
    </form>
    <el-alert v-if="error" type="error" :title="error" :closable="false" />
    <div v-loading="loading" class="event-cards">
      <router-link
        v-for="item in items"
        :key="`${item.kind}-${item.id}`"
        :to="
          reviewMode
            ? `/admin/competitions/${item.kind}/${item.id}`
            : `/competitions/${item.kind}/${item.id}`
        "
        class="panel event-card"
        ><div>
          <el-tag>{{ item.kind === "exam" ? "考试" : "团队比赛" }}</el-tag
          ><el-tag v-if="reviewMode" type="info">{{
            reviewLabels[item.review_status]
          }}</el-tag>
        </div>
        <h2>{{ item.title }}</h2>
        <p>
          {{ item.context_name }} · {{ item.problem_count }} 道题 ·
          {{ item.scoring_rule.toUpperCase() }}
        </p>
        <p>
          {{ formatDateTime(reviewMode ? item.starts_at : item.ends_at) }}
          {{ reviewMode ? "开始" : "结束" }}
        </p>
        <span>{{
          reviewMode ? "查看整场审核 →" : "查看归档 →"
        }}</span></router-link
      >
    </div>
    <el-empty
      v-if="!loading && !error && !items.length"
      :description="
        reviewMode ? '暂无符合条件的场次' : '暂无有权查看的已完结场次'
      "
    />
    <el-pagination
      v-if="total"
      v-model:current-page="query.page"
      :total="total"
      :page-size="20"
      layout="prev, pager, next"
      :pager-count="5"
      @current-change="load"
    />
  </section>
</template>
<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import {
  listEvents,
  reviewLabels,
  type EventListItem,
} from "../../api/modules/events";
import { apiErrorMessage } from "../../api/errors";
import { formatDateTime } from "../../features/time";
const props = defineProps<{ reviewMode?: boolean }>();
const query = reactive({
    page: 1,
    q: "",
    kind: "",
    status: props.reviewMode ? "pending" : "",
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
onMounted(load);
</script>
<style scoped>
.competition-list {
  padding: 28px;
  display: grid;
  gap: 22px;
  max-width: 1440px;
  margin: auto;
}
header h1 {
  margin: 0;
}
header p,
.event-card p {
  color: var(--muted);
  line-height: 1.7;
}
.filters {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}
.filters .el-input {
  width: min(330px, 100%);
}
.filters .el-select {
  width: 180px;
}
.event-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(320px, 100%), 1fr));
  gap: 16px;
}
.event-card {
  padding: 22px;
  color: var(--text);
  text-decoration: none;
  overflow-wrap: anywhere;
}
.event-card > div {
  display: flex;
  gap: 10px;
}
.event-card h2 {
  font-size: 20px;
}
.event-card > span {
  color: var(--accent);
}
@media (max-width: 600px) {
  .competition-list {
    padding: 16px;
  }
}
</style>
