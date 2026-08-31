<template>
  <div class="problem-step">
    <section class="problem-library">
      <div class="library-heading">
        <div>
          <h2>题库</h2>
          <span>{{ activeChoices.length.toLocaleString() }}</span>
        </div>
        <el-segmented v-model="source" :options="sourceOptions" size="small" />
      </div>

      <el-input
        v-model="query"
        clearable
        placeholder="搜索题目标题、编号或标签"
      >
        <template #prefix
          ><el-icon><Search /></el-icon
        ></template>
      </el-input>

      <div class="library-filters">
        <el-select v-model="difficulty" placeholder="难度" clearable>
          <el-option label="入门" value="easy" />
          <el-option label="中等" value="medium" />
          <el-option label="困难" value="hard" />
        </el-select>
        <el-button
          v-if="source === 'markdown'"
          type="primary"
          plain
          @click="emit('create')"
          >新建 Markdown 题目</el-button
        >
        <el-button v-if="source === 'markdown'" @click="emit('batch')"
          >批量导入</el-button
        >
      </div>

      <div v-if="source === 'markdown'" class="authoring-callout">
        <strong>创建考试专用题目</strong>
        <p>
          支持 Markdown、LaTeX、图片和隐藏测试点，考试结束后可同步到公共题库。
        </p>
      </div>

      <div v-else-if="pagedChoices.length" class="problem-list">
        <article
          v-for="choice in pagedChoices"
          :key="choice.value"
          class="problem-row"
        >
          <el-button
            circle
            plain
            :disabled="selectedIDs.has(choice.value)"
            :aria-label="`添加 ${choice.title}`"
            @click="emit('add', source, choice.value)"
          >
            <el-icon><Plus /></el-icon>
          </el-button>
          <div class="problem-copy">
            <div>
              <strong>{{ choice.title }}</strong
              ><span>#{{ choice.value }}</span>
            </div>
            <div class="problem-tags">
              <el-tag
                v-for="tag in choice.tags?.slice(0, 2)"
                :key="tag"
                size="small"
                effect="plain"
                >{{ tag }}</el-tag
              >
              <el-tag
                v-if="choice.difficulty"
                size="small"
                :type="difficultyType(choice.difficulty)"
                effect="plain"
              >
                {{ difficultyText(choice.difficulty) }}
              </el-tag>
            </div>
          </div>
        </article>
      </div>

      <el-empty v-else description="没有匹配的题目" :image-size="72" />

      <el-pagination
        v-if="filteredChoices.length > pageSize && source !== 'markdown'"
        v-model:current-page="page"
        small
        background
        layout="prev, pager, next"
        :page-size="pageSize"
        :total="filteredChoices.length"
      />
    </section>

    <main class="composition-canvas">
      <div class="composition-heading">
        <div>
          <h1>
            已选试题 <span>{{ problems.length }}</span>
          </h1>
          <p>{{ examTitle || "未命名考试" }}</p>
        </div>
        <div class="score-total">
          总分 <strong>{{ total }}</strong
          ><span>/100</span>
        </div>
      </div>

      <div class="selected-table-head" aria-hidden="true">
        <span>#</span><span>题目</span><span>难度</span><span>分值</span
        ><span>操作</span>
      </div>

      <div v-if="problems.length" class="selected-problems">
        <article
          v-for="(item, index) in problems"
          :key="item.problem_id"
          class="selected-row"
        >
          <span class="problem-order">{{ index + 1 }}</span>
          <div class="selected-title">
            <strong>{{ item.title }}</strong>
            <div>
              <el-input
                v-model="item.label"
                size="small"
                maxlength="16"
                aria-label="题号"
              />
              <el-tag size="small" effect="plain">{{ item.source }}</el-tag>
            </div>
          </div>
          <el-tag size="small" effect="plain">编程题</el-tag>
          <el-input-number
            v-model="item.score"
            :min="1"
            :max="1000"
            size="small"
            controls-position="right"
          />
          <div class="row-actions">
            <el-button
              text
              :disabled="index === 0"
              aria-label="上移"
              @click="move(index, -1)"
              ><el-icon><ArrowUp /></el-icon
            ></el-button>
            <el-button
              text
              :disabled="index === problems.length - 1"
              aria-label="下移"
              @click="move(index, 1)"
              ><el-icon><ArrowDown /></el-icon
            ></el-button>
            <el-button
              text
              type="danger"
              aria-label="移除"
              @click="remove(index)"
              ><el-icon><Delete /></el-icon
            ></el-button>
          </div>
        </article>
      </div>

      <el-empty v-else description="从左侧题库添加题目" />
    </main>

    <ExamScoreSummary
      :total="total"
      :count="problems.length"
      :scoring-rule="scoringRule"
    />
  </div>
</template>

<script setup lang="ts">
import {
  ArrowDown,
  ArrowUp,
  Delete,
  Plus,
  Search,
} from "@element-plus/icons-vue";
import { computed, ref, watch } from "vue";
import type { ScoringRule } from "../../api/client";
import type {
  ProblemChoice,
  SelectedExamProblem,
} from "../../features/exams/types";
import ExamScoreSummary from "./ExamScoreSummary.vue";

const problems = defineModel<SelectedExamProblem[]>("problems", {
  required: true,
});
const props = defineProps<{
  classChoices: ProblemChoice[];
  preparedChoices: ProblemChoice[];
  examTitle: string;
  total: number;
  scoringRule: ScoringRule;
}>();

const emit = defineEmits<{
  add: [source: "class" | "prepared", problemID: number];
  create: [];
  batch: [];
}>();

const source = ref<"class" | "prepared" | "markdown">("class");
const query = ref("");
const difficulty = ref("");
const page = ref(1);
const pageSize = 10;
const sourceOptions = [
  { label: "公共题库", value: "class" },
  { label: "预备题库", value: "prepared" },
  { label: "新建题目", value: "markdown" },
];

const activeChoices = computed(() =>
  source.value === "prepared" ? props.preparedChoices : props.classChoices,
);
const selectedIDs = computed(
  () => new Set(problems.value.map((item) => item.problem_id)),
);
const filteredChoices = computed(() => {
  const term = query.value.trim().toLowerCase();
  return activeChoices.value.filter((choice) => {
    const matchesTerm =
      !term ||
      `${choice.value} ${choice.title} ${(choice.tags || []).join(" ")}`
        .toLowerCase()
        .includes(term);
    const matchesDifficulty =
      !difficulty.value || choice.difficulty === difficulty.value;
    return matchesTerm && matchesDifficulty;
  });
});
const pagedChoices = computed(() =>
  filteredChoices.value.slice(
    (page.value - 1) * pageSize,
    page.value * pageSize,
  ),
);

watch([source, query, difficulty], () => {
  page.value = 1;
});

function move(index: number, delta: number) {
  const target = index + delta;
  if (target < 0 || target >= problems.value.length) return;
  const next = [...problems.value];
  const [item] = next.splice(index, 1);
  next.splice(target, 0, item);
  problems.value = next;
}

function remove(index: number) {
  problems.value = problems.value.filter((_, itemIndex) => itemIndex !== index);
}

function difficultyText(value: string) {
  return (
    ({ easy: "入门", medium: "中等", hard: "困难" } as Record<string, string>)[
      value
    ] || value
  );
}

function difficultyType(
  value: string,
): "success" | "warning" | "danger" | "info" {
  if (value === "easy") return "success";
  if (value === "hard") return "danger";
  if (value === "medium") return "warning";
  return "info";
}
</script>

<style scoped>
.problem-step {
  display: grid;
  grid-template-columns: minmax(310px, 0.8fr) minmax(560px, 1.65fr) minmax(
      240px,
      0.58fr
    );
  min-height: calc(100vh - 82px);
  background: #f6f8fb;
}

.problem-library,
.composition-canvas {
  min-width: 0;
  padding: 24px 20px;
  border-right: 1px solid #e0e5ed;
  background: #fff;
}

.library-heading,
.composition-heading,
.library-heading > div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.library-heading {
  align-items: flex-start;
  margin-bottom: 18px;
}

.library-heading h2,
.composition-heading h1 {
  margin: 0;
  color: #172033;
}

.library-heading h2 {
  font-size: 20px;
}

.library-heading span,
.composition-heading h1 span {
  display: inline-grid;
  min-width: 24px;
  height: 24px;
  place-items: center;
  border-radius: 12px;
  color: #0a5bd7;
  background: #eaf2ff;
  font-size: 12px;
}

.library-filters {
  display: flex;
  gap: 8px;
  margin: 12px 0;
}

.library-filters :deep(.el-select) {
  flex: 1;
}

.problem-list {
  margin-top: 8px;
  border-top: 1px solid #e5e9ef;
}

.problem-row {
  display: grid;
  grid-template-columns: 38px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  min-height: 66px;
  border-bottom: 1px solid #edf0f4;
}

.problem-copy {
  min-width: 0;
}

.problem-copy > div:first-child {
  display: flex;
  justify-content: space-between;
  gap: 8px;
}

.problem-copy strong {
  overflow: hidden;
  color: #263246;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.problem-copy > div:first-child span {
  color: #8690a0;
  font-size: 12px;
}

.problem-tags {
  display: flex;
  gap: 4px;
  margin-top: 6px;
}

.authoring-callout {
  margin-top: 22px;
  padding: 22px;
  border: 1px solid #cddaf0;
  border-radius: 8px;
  color: #29405e;
  background: #f4f8ff;
}

.authoring-callout p {
  margin: 8px 0 0;
  color: #637188;
  line-height: 1.65;
}

.composition-canvas {
  background: #fff;
}

.composition-heading {
  min-height: 72px;
  padding: 0 8px 18px;
  border-bottom: 1px solid #e5e9ef;
}

.composition-heading h1 {
  font-size: 20px;
}

.composition-heading p {
  margin: 8px 0 0;
  color: #0b58c4;
  font-weight: 650;
}

.score-total {
  color: #586579;
  font-weight: 650;
}

.score-total strong {
  margin-left: 8px;
  color: #0a5bd7;
  font-size: 22px;
}

.score-total span {
  color: #8a94a3;
}

.selected-table-head,
.selected-row {
  display: grid;
  grid-template-columns: 42px minmax(220px, 1fr) 90px 118px 116px;
  align-items: center;
  gap: 10px;
}

.selected-table-head {
  min-height: 54px;
  padding: 0 8px;
  color: #7a8494;
  font-size: 12px;
  font-weight: 650;
}

.selected-row {
  min-height: 78px;
  padding: 10px 8px;
  border-top: 1px solid #edf0f4;
}

.problem-order {
  color: #273449;
  font-weight: 700;
}

.selected-title {
  min-width: 0;
}

.selected-title strong {
  display: block;
  overflow: hidden;
  color: #243146;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.selected-title div {
  display: flex;
  gap: 6px;
  margin-top: 7px;
}

.selected-title :deep(.el-input) {
  width: 58px;
}

.selected-row :deep(.el-input-number) {
  width: 104px;
}

.row-actions {
  display: flex;
}

@media (max-width: 1100px) {
  .problem-step {
    grid-template-columns: 330px minmax(560px, 1fr);
  }
  .problem-step :deep(.score-summary) {
    grid-column: 1 / -1;
    border-top: 1px solid #e0e5ed;
    border-left: 0;
  }
}

@media (max-width: 900px) {
  .problem-step {
    grid-template-columns: 1fr;
  }
  .problem-library,
  .composition-canvas {
    border-right: 0;
    border-bottom: 1px solid #e0e5ed;
  }
  .selected-table-head {
    display: none;
  }
  .selected-row {
    grid-template-columns: 32px minmax(180px, 1fr) 90px;
  }
  .selected-row > :nth-child(3) {
    display: none;
  }
  .row-actions {
    grid-column: 2 / -1;
  }
}
</style>
