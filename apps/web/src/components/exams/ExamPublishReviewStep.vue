<template>
  <div class="publish-step">
    <main class="review-document">
      <header class="review-hero">
        <div>
          <span>发布前最终确认</span>
          <h1>发布考试</h1>
          <p>请检查考试内容、时间、范围与发布设置。</p>
        </div>
        <img src="/bg-hero.webp" alt="青岛黄海学院校园" />
      </header>

      <section class="review-section exam-identity">
        <h2>{{ draft.title }}</h2>
        <div class="identity-grid">
          <span>考试时间</span><strong>{{ timeSummary }}</strong>
          <span>考试形式</span
          ><strong>{{
            draft.manual_review ? "人工确认分数" : "自动判题"
          }}</strong>
          <span>考试对象</span><strong>{{ classLabel }}</strong>
          <span>所属课程</span><strong>{{ courseLabel }}</strong>
        </div>
      </section>

      <section v-if="draft.description" class="review-section">
        <h3>考试说明</h3>
        <p>{{ draft.description }}</p>
      </section>

      <section class="review-section">
        <div class="section-heading">
          <h3>试题与分值</h3>
          <strong>共 {{ problems.length }} 题 · {{ total }} 分</strong>
        </div>
        <div class="review-table">
          <div class="review-table-head">
            <span>题号</span><span>题目名称</span><span>来源</span
            ><span>分值</span>
          </div>
          <div
            v-for="item in problems"
            :key="item.problem_id"
            class="review-row"
          >
            <strong>{{ item.label }}</strong
            ><span>{{ item.title }}</span
            ><span>{{ item.source }}</span
            ><strong>{{ item.score }}</strong>
          </div>
        </div>
      </section>

      <section class="review-section">
        <h3>发布设置</h3>
        <div class="publish-settings">
          <div>
            <span>榜单可见</span
            ><strong>{{
              draft.ranking_visible ? "考生可查看实时排名" : "仅教师可见"
            }}</strong>
          </div>
          <div>
            <span>计分规则</span
            ><strong>{{
              draft.scoring_rule.toUpperCase()
            }}</strong>
          </div>
          <div>
            <span>封榜</span><strong>{{ draft.freeze_enabled ? `结束前 ${draft.freeze_duration_minutes} 分钟` : "未启用" }}</strong>
          </div>
          <div>
            <span>结束方式</span
            ><strong>{{
              draft.ends_at ? "到期自动结束" : "教师手动结束"
            }}</strong>
          </div>
          <div>
            <span>题目发布</span
            ><strong>{{
              hasReleaseProblems ? "预备题将在考试结束后发布" : "不发布新题"
            }}</strong>
          </div>
        </div>
      </section>
    </main>

    <aside class="publish-summary">
      <h2>发布摘要</h2>
      <dl>
        <div>
          <dt>考试名称</dt>
          <dd>{{ draft.title }}</dd>
        </div>
        <div>
          <dt>考试对象</dt>
          <dd>{{ classLabel }}</dd>
        </div>
        <div>
          <dt>试题数量</dt>
          <dd>{{ problems.length }} 题</dd>
        </div>
        <div>
          <dt>总分</dt>
          <dd>{{ total }} 分</dd>
        </div>
      </dl>
      <div class="publish-checks">
        <h3>发布检查</h3>
        <p v-for="check in checks" :key="check">
          <el-icon><CircleCheck /></el-icon>{{ check }}
        </p>
      </div>
      <el-button
        class="publish-button"
        type="danger"
        size="large"
        :loading="saving"
        @click="emit('publish')"
      >
        确认发布
      </el-button>
      <el-button class="back-button" size="large" @click="emit('back')"
        >返回修改</el-button
      >
    </aside>
  </div>
</template>

<script setup lang="ts">
import { CircleCheck } from "@element-plus/icons-vue";
import { computed } from "vue";
import { formatExamDate } from "../../features/exams/builder";
import type {
  ExamDraft,
  SelectedExamProblem,
} from "../../features/exams/types";

const props = defineProps<{
  draft: ExamDraft;
  problems: SelectedExamProblem[];
  courseLabel: string;
  classLabel: string;
  total: number;
  saving: boolean;
}>();

const emit = defineEmits<{ publish: []; back: [] }>();

const timeSummary = computed(
  () =>
    `${formatExamDate(props.draft.starts_at) || "立即开始"} — ${formatExamDate(props.draft.ends_at) || "手动结束"}`,
);
const hasReleaseProblems = computed(() =>
  props.problems.some((item) => item.release_after_exam),
);
const checks = computed(() => [
  "考试信息完整",
  `已选择 ${props.problems.length} 道题并设置分值`,
  props.total === 100
    ? "试卷总分为 100 分"
    : `当前试卷总分为 ${props.total} 分`,
  "发布范围与计分方式已确认",
]);
</script>

<style scoped>
.publish-step {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 360px;
  min-height: calc(100vh - 82px);
  background: #f8f8f6;
}

.review-document {
  min-width: 0;
  padding: 28px 42px 60px;
  background: #fff;
}

.review-hero {
  display: grid;
  grid-template-columns: 1fr minmax(260px, 42%);
  min-height: 180px;
  align-items: center;
  gap: 30px;
  overflow: hidden;
  border-bottom: 2px solid #164b9d;
}

.review-hero span {
  color: #667085;
  font-size: 13px;
}

.review-hero h1 {
  margin: 5px 0 8px;
  color: #123d7a;
  font-family: "STSong", "Songti SC", serif;
  font-size: 54px;
  letter-spacing: 0.08em;
}

.review-hero p {
  margin: 0;
  color: #7a8492;
}

.review-hero img {
  width: 100%;
  height: 180px;
  object-fit: cover;
  opacity: 0.38;
}

.review-section {
  max-width: 980px;
  padding: 24px 8px;
  border-bottom: 1px solid #dfe3e8;
}

.review-section h2,
.review-section h3,
.review-section p {
  margin-top: 0;
}

.review-section h2 {
  color: #123d7a;
  font-family: "STSong", "Songti SC", serif;
  font-size: 28px;
}

.review-section h3 {
  padding-left: 12px;
  border-left: 4px solid #164b9d;
  color: #173d73;
  font-size: 18px;
}

.identity-grid {
  display: grid;
  grid-template-columns: 100px minmax(0, 1fr) 100px minmax(0, 1fr);
  gap: 14px;
  color: #667085;
}

.identity-grid strong {
  color: #344054;
}

.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-heading strong {
  color: #164b9d;
}

.review-table-head,
.review-row {
  display: grid;
  grid-template-columns: 70px minmax(220px, 1fr) 120px 70px;
  gap: 14px;
  align-items: center;
  min-height: 42px;
  padding: 0 10px;
}

.review-table-head {
  color: #7a8492;
  font-size: 12px;
  font-weight: 700;
}

.review-row {
  border-top: 1px solid #edf0f3;
  color: #354154;
}

.review-row strong {
  color: #164b9d;
}

.publish-settings {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px 32px;
}

.publish-settings div {
  display: grid;
  gap: 5px;
}

.publish-settings span {
  color: #7a8492;
  font-size: 13px;
}

.publish-settings strong {
  color: #354154;
}

.publish-summary {
  align-self: start;
  margin: 28px 28px 28px 0;
  padding: 28px;
  border: 1px solid #d8dde5;
  background: #fff;
}

.publish-summary h2 {
  margin: 0 0 24px;
  padding-bottom: 12px;
  border-bottom: 3px solid #c9302c;
  color: #173d73;
  font-family: "STSong", "Songti SC", serif;
}

.publish-summary dl {
  display: grid;
  gap: 18px;
  margin: 0;
}

.publish-summary dl div {
  display: flex;
  justify-content: space-between;
  gap: 18px;
}

.publish-summary dt {
  color: #7a8492;
}

.publish-summary dd {
  margin: 0;
  color: #303c4e;
  font-weight: 650;
  text-align: right;
}

.publish-checks {
  margin-top: 26px;
  padding-top: 22px;
  border-top: 1px solid #e0e4e9;
}

.publish-checks h3 {
  color: #173d73;
}

.publish-checks p {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #237a45;
  font-size: 13px;
}

.publish-button,
.back-button {
  width: 100%;
  margin: 16px 0 0;
}

@media (max-width: 1100px) {
  .publish-step {
    grid-template-columns: 1fr;
  }
  .publish-summary {
    margin: 0 28px 28px;
  }
}

@media (max-width: 700px) {
  .review-document {
    padding: 20px 16px 40px;
  }
  .review-hero {
    grid-template-columns: 1fr;
  }
  .review-hero img {
    display: none;
  }
  .identity-grid,
  .publish-settings {
    grid-template-columns: 1fr;
  }
  .review-table {
    overflow-x: auto;
  }
  .review-table-head,
  .review-row {
    min-width: 620px;
  }
  .publish-summary {
    margin: 0 16px 24px;
  }
}
</style>
