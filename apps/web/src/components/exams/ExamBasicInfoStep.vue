<template>
  <div class="basic-step">
    <main class="basic-canvas">
      <div class="step-intro">
        <span>新建考试</span>
        <h1>基本信息</h1>
        <p>设置考试名称、开放范围、时间与评分规则。</p>
      </div>

      <section class="form-section">
        <h2>考试信息</h2>
        <el-form :model="model" label-position="top" class="exam-form">
          <el-form-item label="考试名称" required>
            <el-input
              v-model="model.title"
              maxlength="120"
              show-word-limit
              placeholder="算法设计期中考试"
            />
          </el-form-item>
          <el-form-item label="考试描述">
            <el-input
              v-model="model.description"
              type="textarea"
              :rows="3"
              maxlength="400"
              show-word-limit
              placeholder="向考生说明考试范围和注意事项"
            />
          </el-form-item>

          <div class="form-grid">
            <el-form-item label="可见范围" required>
              <el-select
                v-model="model.class_id"
                @change="emit('class-change')"
              >
                <el-option label="全课程（不限班级）" :value="-1" />
                <el-option
                  v-for="item in classes"
                  :key="item.class_id"
                  :label="`${item.course_code} / ${item.class_name}`"
                  :value="item.class_id"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="所属课程" required>
              <el-select
                v-if="model.class_id === -1"
                v-model="model.course_id"
                @change="emit('course-change')"
              >
                <el-option
                  v-for="course in courses"
                  :key="course.id"
                  :label="`${course.code} ${course.name}`"
                  :value="course.id"
                />
              </el-select>
              <el-input v-else :model-value="courseLabel" disabled />
            </el-form-item>
            <el-form-item label="开始时间">
              <el-date-picker
                v-model="model.starts_at"
                type="datetime"
                format="YYYY-MM-DD HH:mm"
                placeholder="留空则立即开始"
              />
            </el-form-item>
            <el-form-item label="结束时间">
              <el-date-picker
                v-model="model.ends_at"
                type="datetime"
                format="YYYY-MM-DD HH:mm"
                placeholder="留空则手动结束"
              />
            </el-form-item>
          </div>

          <div class="rule-block">
            <h2>评分与考试规则</h2>
            <div class="rule-row">
              <div>
                <strong>计分方式</strong><span>决定榜单排序与同分规则</span>
              </div>
              <el-radio-group v-model="model.scoring_rule">
                <el-radio-button value="penalty">通过数 + 罚时</el-radio-button>
                <el-radio-button value="score">总分数</el-radio-button>
              </el-radio-group>
            </div>
            <div class="rule-row">
              <div>
                <strong>人工确认分数</strong
                ><span>判题完成后由教师确认最终分数</span>
              </div>
              <el-switch v-model="model.manual_review" />
            </div>
            <div class="rule-row">
              <div>
                <strong>实时榜单</strong
                ><span>允许考生在考试过程中查看排名</span>
              </div>
              <el-switch v-model="model.ranking_visible" />
            </div>
          </div>
        </el-form>
      </section>
    </main>

    <aside class="exam-overview">
      <h2>考试概览</h2>
      <dl>
        <div>
          <dt>考试名称</dt>
          <dd>{{ model.title || "尚未填写" }}</dd>
        </div>
        <div>
          <dt>题目数量</dt>
          <dd>{{ problemCount ? `${problemCount} 题` : "未选择" }}</dd>
        </div>
        <div>
          <dt>总分</dt>
          <dd>{{ totalScore }} 分</dd>
        </div>
        <div>
          <dt>可见范围</dt>
          <dd>{{ classLabel || "尚未选择" }}</dd>
        </div>
        <div>
          <dt>考试时间</dt>
          <dd>{{ timeSummary }}</dd>
        </div>
        <div>
          <dt>计分方式</dt>
          <dd>
            {{ model.scoring_rule === "score" ? "总分数" : "通过数 + 罚时" }}
          </dd>
        </div>
      </dl>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { ClassContext } from "../../api/client";
import { formatExamDate } from "../../features/exams/builder";
import type { CourseSummary, ExamDraft } from "../../features/exams/types";

const model = defineModel<ExamDraft>({ required: true });
const props = defineProps<{
  courses: CourseSummary[];
  classes: ClassContext[];
  courseLabel: string;
  classLabel: string;
  problemCount: number;
  totalScore: number;
}>();

const emit = defineEmits<{ "class-change": []; "course-change": [] }>();

const timeSummary = computed(() => {
  if (!model.value.starts_at && !model.value.ends_at) return "创建后立即开始";
  const start = formatExamDate(model.value.starts_at) || "立即开始";
  const end = formatExamDate(model.value.ends_at) || "手动结束";
  return `${start} — ${end}`;
});
</script>

<style scoped>
.basic-step {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 360px;
  min-height: calc(100vh - 82px);
  background: #f8fafc;
}

.basic-canvas {
  padding: 34px 42px 54px;
}

.step-intro span {
  color: #0a5bd7;
  font-size: 13px;
  font-weight: 700;
}

.step-intro h1 {
  margin: 8px 0 6px;
  color: #101828;
  font-size: 34px;
}

.step-intro p {
  margin: 0;
  color: #667085;
}

.form-section {
  max-width: 900px;
  margin-top: 28px;
  padding: 26px 28px;
  border: 1px solid #dce3ed;
  border-radius: 10px;
  background: #fff;
}

.form-section h2 {
  margin: 0 0 20px;
  color: #182230;
  font-size: 18px;
}

.exam-form :deep(.el-form-item__label) {
  color: #344054;
  font-weight: 650;
}

.exam-form :deep(.el-select),
.exam-form :deep(.el-date-editor) {
  width: 100%;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 18px;
}

.rule-block {
  margin-top: 8px;
  padding-top: 22px;
  border-top: 1px solid #e4e8ef;
}

.rule-row {
  display: flex;
  min-height: 66px;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  border-bottom: 1px solid #edf0f4;
}

.rule-row div {
  display: grid;
  gap: 4px;
}

.rule-row strong {
  color: #263246;
}

.rule-row span {
  color: #7a8494;
  font-size: 13px;
}

.exam-overview {
  padding: 38px 30px;
  border-left: 1px solid #dce3ed;
  background: #fff;
}

.exam-overview h2 {
  margin: 0 0 28px;
  color: #172033;
}

.exam-overview dl {
  display: grid;
  gap: 24px;
  margin: 0;
}

.exam-overview dl div {
  display: grid;
  gap: 6px;
}

.exam-overview dt {
  color: #7c8797;
  font-size: 13px;
}

.exam-overview dd {
  margin: 0;
  color: #263246;
  font-weight: 650;
  line-height: 1.55;
}

@media (max-width: 1100px) {
  .basic-step {
    grid-template-columns: 1fr;
  }
  .exam-overview {
    border-top: 1px solid #dce3ed;
    border-left: 0;
  }
}

@media (max-width: 700px) {
  .basic-canvas,
  .exam-overview {
    padding: 22px 16px;
  }
  .form-section {
    padding: 20px 16px;
  }
  .form-grid {
    grid-template-columns: 1fr;
  }
  .rule-row {
    align-items: flex-start;
    flex-direction: column;
    padding: 14px 0;
  }
}
</style>
