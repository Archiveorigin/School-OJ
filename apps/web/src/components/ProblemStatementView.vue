<template>
  <div
    class="problem-view-grid"
    :class="{
      'problem-view-grid--single': !showMeta,
      'problem-view-grid--embedded': embedded
    }"
  >
    <section class="panel statement-box">
      <div v-if="showTitle" class="statement-head">
        <h1>{{ problem.title }}</h1>
      </div>
      <MarkdownRenderer :source="problem.statement" :problem-id="problem.id" />
    </section>

    <aside v-if="showMeta" class="panel meta-box">
      <div class="meta-title">
        <span class="eyebrow">题目编号</span>
        <strong>{{ displayNumber }}</strong>
      </div>
      <div class="meta-grid">
        <span>提交状态</span>
        <div v-if="statusImage" class="status-image-wrap">
          <SubmissionStatusMark :status="statusImage" class="status-icon" />
        </div>
        <el-tag v-else :type="statusType || 'info'" effect="light">{{ statusText || '未提交' }}</el-tag>
        <span>分值</span>
        <strong>{{ score ?? '-' }}</strong>
        <template v-if="showDifficulty">
          <span>难度</span>
          <el-tag :type="difficultyTagType(difficulty)" effect="light">{{ difficulty || '未设置' }}</el-tag>
        </template>
        <span>限制</span>
        <strong class="limit-lines">
          <span v-for="line in problemLimitLines(problem)" :key="line">{{ line }}</span>
        </strong>
      </div>
      <div v-if="showTags" class="tag-section">
        <span class="muted">标签</span>
        <div v-if="tags.length" class="tag-row">
          <el-tag v-for="tag in tags" :key="tag" size="small">{{ tag }}</el-tag>
        </div>
        <span v-else class="muted">暂无标签</span>
      </div>
      <slot name="sidebar-footer" />
    </aside>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Problem } from '../api/client'
import { difficultyTagType, problemDisplayCode, problemDifficulty, problemLimitLines, tagList } from '../features/problems/problemMeta'
import MarkdownRenderer from './MarkdownRenderer.vue'
import SubmissionStatusMark from './SubmissionStatusMark.vue'

const props = withDefaults(
  defineProps<{
    problem: Problem
    problemNumber?: number | string
    score?: number | string
    statusText?: string
    statusType?: 'success' | 'warning' | 'info' | 'danger'
    statusImage?: 'ac' | 'uac' | ''
    showDifficulty?: boolean
    showTitle?: boolean
    showTags?: boolean
    showMeta?: boolean
    embedded?: boolean
  }>(),
  {
    showDifficulty: true,
    showTitle: true,
    showTags: true,
    showMeta: true,
    embedded: false
  }
)

const tags = computed(() => tagList(props.problem.tags))
const difficulty = computed(() => problemDifficulty(props.problem))
const displayNumber = computed(() => props.problemNumber || problemDisplayCode(props.problem))
const statusImage = computed(() => props.statusImage || '')
</script>

<style scoped>
.problem-view-grid {
  display: grid;
  grid-template-columns: minmax(360px, 1fr) minmax(362px, 420px);
  gap: 14px;
  align-items: start;
}

.problem-view-grid--single {
  grid-template-columns: minmax(0, 980px);
  justify-content: center;
}

.problem-view-grid--embedded {
  width: 100%;
  grid-template-columns: minmax(0, 1fr) minmax(300px, 360px);
  gap: 24px;
}

.problem-view-grid--embedded.problem-view-grid--single {
  grid-template-columns: minmax(0, 1fr);
  justify-content: stretch;
}

.problem-view-grid--embedded .statement-box,
.problem-view-grid--embedded .meta-box {
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  backdrop-filter: none;
}

.problem-view-grid--embedded .statement-box:hover,
.problem-view-grid--embedded .meta-box:hover {
  border-color: transparent;
  box-shadow: none;
}

.problem-view-grid--embedded .statement-head {
  margin-bottom: 22px;
  padding: 4px 0 16px;
}

.statement-box,
.meta-box {
  min-width: 0;
}

.statement-head {
  margin-bottom: 18px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
}

.statement-head h1 {
  margin: 0;
  color: var(--text);
  font-size: 24px;
  line-height: 1.3;
}

.eyebrow {
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
}

.meta-title {
  display: grid;
  gap: 2px;
  margin-bottom: 14px;
}

.meta-title strong {
  margin: 4px 0 0;
  color: var(--text);
  font-size: 24px;
}

.meta-grid {
  display: grid;
  grid-template-columns: 76px minmax(0, 1fr);
  gap: 10px 12px;
  align-items: center;
}

.status-image-wrap {
  display: flex;
  align-items: center;
  min-height: 48px;
}

.status-icon {
  width: min(156px, 100%);
  height: auto;
  flex: 0 0 auto;
  transition: transform 0.2s ease-in-out;
}

.status-icon:hover {
  transform: scale(1.1);
}

.meta-grid span,
.tag-section > span {
  color: var(--muted);
}

.limit-lines {
  display: grid;
  gap: 2px;
}

.tag-section {
  display: grid;
  gap: 8px;
  margin-top: 18px;
}

.tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

@media (max-width: 980px) {
  .problem-view-grid {
    grid-template-columns: 1fr;
  }
}
</style>
