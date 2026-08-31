<template>
  <aside class="score-summary">
    <h3>试卷摘要</h3>
    <div class="score-ring">
      <el-progress
        type="circle"
        :percentage="progress"
        :width="148"
        :stroke-width="15"
        color="#1464d8"
      >
        <template #default>
          <div class="score-value">
            <span>总分</span><strong>{{ total }}</strong
            ><small>已分配</small>
          </div>
        </template>
      </el-progress>
    </div>
    <dl>
      <div>
        <dt>题目数量</dt>
        <dd>{{ count }}</dd>
      </div>
      <div>
        <dt>试卷总分</dt>
        <dd>{{ total }} 分</dd>
      </div>
      <div>
        <dt>计分方式</dt>
        <dd>{{ scoringRule.toUpperCase() }}</dd>
      </div>
    </dl>
    <div class="score-validation" :class="{ valid: total === 100 }">
      <strong>{{
        total === 100
          ? "分值校验通过"
          : `还需分配 ${Math.max(0, 100 - total)} 分`
      }}</strong>
      <span>{{
        total === 100
          ? "总分已分配 100 分，符合要求。"
          : "建议将试卷总分调整为 100 分。"
      }}</span>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { ScoringRule } from "../../api/client";

const props = defineProps<{
  total: number;
  count: number;
  scoringRule: ScoringRule;
}>();
const progress = computed(() => Math.min(100, Math.max(0, props.total)));
</script>

<style scoped>
.score-summary {
  min-width: 0;
  padding: 22px 20px;
  border-left: 1px solid #e0e5ed;
  background: #fff;
}

.score-summary h3 {
  margin: 0 0 24px;
  color: #172033;
}

.score-ring {
  width: 148px;
  margin: 0 auto 26px;
}

.score-value {
  text-align: center;
}

.score-ring span,
.score-ring strong,
.score-ring small {
  display: block;
}

.score-ring strong {
  color: #111827;
  font-size: 34px;
  line-height: 1.1;
}

.score-ring span,
.score-ring small {
  color: #697386;
  font-size: 12px;
}

.score-summary dl {
  display: grid;
  gap: 18px;
  margin: 0;
  padding: 22px 0;
  border-top: 1px solid #e5e9f0;
}

.score-summary dl div {
  display: flex;
  justify-content: space-between;
  gap: 14px;
}

.score-summary dt {
  color: #737d8e;
}

.score-summary dd {
  margin: 0;
  color: #111827;
  font-weight: 650;
}

.score-validation {
  display: grid;
  gap: 5px;
  margin-top: 22px;
  padding: 14px;
  border: 1px solid #f3c98b;
  border-radius: 8px;
  color: #9a5700;
  background: #fff9ed;
}

.score-validation.valid {
  border-color: #b7dfc4;
  color: #16733a;
  background: #f0faf3;
}

.score-validation span {
  font-size: 12px;
  line-height: 1.6;
}
</style>
