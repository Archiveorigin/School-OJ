<template>
  <div class="problem-view-grid">
    <section class="panel statement-box">
      <div class="statement-head">
        <div>
          <span class="eyebrow">题目信息</span>
          <h3>{{ problem.title }}</h3>
        </div>
        <span class="muted">{{ problemLimitText(problem) }}</span>
      </div>
      <MarkdownRenderer :source="problem.statement" :problem-id="problem.id" />

      <div v-if="samples.length" class="sample-section">
        <div class="section-title">
          <h3>输入输出样例</h3>
        </div>
        <div v-for="sample in samples" :key="sample.index" class="sample-pair">
          <div class="sample-block">
            <div class="sample-head">
              <strong>输入样例 {{ sample.index }}</strong>
              <el-button size="small" @click="copyText(sample.input)">复制</el-button>
            </div>
            <pre>{{ sample.input }}</pre>
          </div>
          <div class="sample-block">
            <div class="sample-head">
              <strong>输出样例 {{ sample.index }}</strong>
              <el-button size="small" @click="copyText(sample.output)">复制</el-button>
            </div>
            <pre>{{ sample.output }}</pre>
          </div>
        </div>
      </div>
    </section>

    <aside class="panel meta-box">
      <div class="meta-title">
        <span class="eyebrow">题目编号</span>
        <strong>{{ displayNumber }}</strong>
      </div>
      <div class="meta-grid">
        <span>提交状态</span>
        <div v-if="statusImageSrc" class="status-image-wrap">
          <img class="status-image" :src="statusImageSrc" :alt="statusImageAlt" />
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
      <div class="tag-section">
        <span class="muted">标签</span>
        <div v-if="tags.length" class="tag-row">
          <el-tag v-for="tag in tags" :key="tag" size="small">{{ tag }}</el-tag>
        </div>
        <span v-else class="muted">暂无标签</span>
      </div>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed } from 'vue'
import type { Problem } from '../api/client'
import acStatusImage from '../assets/status/AC.png'
import uacStatusImage from '../assets/status/UAC.png'
import {
  difficultyFromTags,
  difficultyTagType,
  extractStatementSamples,
  problemDisplayCode,
  problemLimitLines,
  problemLimitText,
  tagList
} from '../features/problems/problemMeta'
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
  }>(),
  {
    showDifficulty: true
  }
)

const samples = computed(() => extractStatementSamples(props.problem.statement))
const tags = computed(() => tagList(props.problem.tags))
const difficulty = computed(() => difficultyFromTags(props.problem.tags))
const displayNumber = computed(() => props.problemNumber || problemDisplayCode(props.problem))
const statusImageSrc = computed(() => {
  if (props.statusImage === 'ac') return acStatusImage
  if (props.statusImage === 'uac') return uacStatusImage
  return ''
})
const statusImageAlt = computed(() => (props.statusImage === 'ac' ? '通过' : '未通过'))

async function copyText(value: string) {
  try {
    await navigator.clipboard.writeText(value)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败，请手动选择文本')
  }
}
</script>

<style scoped>
.problem-view-grid {
  display: grid;
  grid-template-columns: minmax(360px, 1fr) minmax(260px, 320px);
  gap: 14px;
  align-items: start;
}

.statement-box,
.meta-box {
  min-width: 0;
}

.statement-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.statement-head h3,
.meta-title strong {
  margin: 4px 0 0;
  color: var(--text);
}

.eyebrow {
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
}

.sample-section {
  display: grid;
  gap: 12px;
  margin-top: 18px;
}

.sample-pair {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.sample-block {
  min-width: 0;
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  background: color-mix(in srgb, var(--surface-strong) 72%, transparent);
}

.sample-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border);
}

.sample-block pre {
  min-height: 88px;
  max-height: 260px;
  overflow: auto;
  margin: 0;
  padding: 12px;
  color: #e2e8f0;
  background: #0f172a;
  white-space: pre;
}

.meta-title {
  display: grid;
  gap: 2px;
  margin-bottom: 14px;
}

.meta-title strong {
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
  min-height: 44px;
}

.status-image {
  width: min(128px, 100%);
  max-height: 64px;
  object-fit: contain;
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
  .problem-view-grid,
  .sample-pair {
    grid-template-columns: 1fr;
  }
}
</style>
