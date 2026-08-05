<template>
  <div class="problem-view-grid" :class="{ single: !showMeta }">
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
        <div v-if="statusImageAlt" class="status-image-wrap" role="img" :aria-label="statusImageAlt">
          <svg
            v-if="statusImage === 'ac'"
            class="status-icon"
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 156 48"
            width="156"
            height="48"
            fill="none"
            aria-hidden="true"
          >
            <rect width="156" height="48" rx="14" fill="#22c55e" />
            <text x="78" y="30" fill="white" text-anchor="middle" font-size="17" font-weight="800" font-family="Inter, Arial, sans-serif">
              Accepted
            </text>
          </svg>
          <svg
            v-else
            class="status-icon"
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 156 48"
            width="156"
            height="48"
            fill="none"
            aria-hidden="true"
          >
            <rect width="156" height="48" rx="14" fill="#ef4444" />
            <text x="78" y="30" fill="white" text-anchor="middle" font-size="17" font-weight="800" font-family="Inter, Arial, sans-serif">
              Unaccepted
            </text>
          </svg>
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
  }>(),
  {
    showDifficulty: true,
    showTitle: true,
    showTags: true,
    showMeta: true
  }
)

const tags = computed(() => tagList(props.problem.tags))
const difficulty = computed(() => problemDifficulty(props.problem))
const displayNumber = computed(() => props.problemNumber || problemDisplayCode(props.problem))
const statusImage = computed(() => props.statusImage || '')
const statusImageAlt = computed(() => {
  if (statusImage.value === 'ac') return '通过'
  if (statusImage.value === 'uac') return '未通过'
  return ''
})
</script>

<style scoped>
.problem-view-grid {
  display: grid;
  grid-template-columns: minmax(360px, 1fr) minmax(362px, 420px);
  gap: 14px;
  align-items: start;
}

.problem-view-grid.single {
  grid-template-columns: minmax(0, 980px);
  justify-content: center;
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
