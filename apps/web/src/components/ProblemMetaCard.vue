<template>
  <aside class="panel problem-meta-card">
    <header class="meta-card-header">
      <div>
        <span class="eyebrow">题目基本信息</span>
        <strong>{{ displayNumber }}</strong>
      </div>
      <el-tag v-if="showDifficulty" :type="difficultyTagType(difficulty)" effect="light">
        {{ difficulty || '未设置' }}
      </el-tag>
    </header>

    <dl class="meta-list">
      <template v-if="statusText">
        <div class="meta-item">
          <dt>提交状态</dt>
          <dd>
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
                <text
                  x="78"
                  y="30"
                  fill="white"
                  text-anchor="middle"
                  font-size="17"
                  font-weight="800"
                  font-family="Inter, Arial, sans-serif"
                >
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
                <text
                  x="78"
                  y="30"
                  fill="white"
                  text-anchor="middle"
                  font-size="17"
                  font-weight="800"
                  font-family="Inter, Arial, sans-serif"
                >
                  Unaccepted
                </text>
              </svg>
            </div>
            <el-tag v-else :type="statusType || 'info'" effect="light">{{ statusText }}</el-tag>
          </dd>
        </div>
      </template>
      <div v-if="score !== undefined && score !== null" class="meta-item">
        <dt>分值</dt>
        <dd>
          <strong>{{ score }}</strong>
        </dd>
      </div>
      <div class="meta-item">
        <dt>时间限制</dt>
        <dd>
          <strong>{{ problem.time_limit_ms ?? '-' }}</strong
          ><span> ms</span>
        </dd>
      </div>
      <div class="meta-item">
        <dt>内存限制</dt>
        <dd>
          <strong>{{ problem.memory_limit_mb ?? '-' }}</strong
          ><span> MB</span>
        </dd>
      </div>
      <div class="meta-item">
        <dt>输出限制</dt>
        <dd>
          <strong>{{ problem.output_limit_kb ?? '-' }}</strong
          ><span> KB</span>
        </dd>
      </div>
    </dl>

    <div class="tag-section">
      <span class="section-label">题目标签</span>
      <div v-if="tags.length" class="tag-row">
        <el-tag v-for="tag in tags" :key="tag" size="small" effect="plain">{{ tag }}</el-tag>
      </div>
      <span v-else class="muted">暂无标签</span>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Problem } from '../api/client'
import { difficultyTagType, problemDisplayCode, problemDifficulty, tagList } from '../features/problems/problemMeta'

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

const tags = computed(() => tagList(props.problem.tags))
const difficulty = computed(() => (props.showDifficulty ? problemDifficulty(props.problem) : ''))
const displayNumber = computed(() => props.problemNumber || problemDisplayCode(props.problem))
const statusImage = computed(() => props.statusImage || '')
const statusImageAlt = computed(() => {
  if (statusImage.value === 'ac') return '通过'
  if (statusImage.value === 'uac') return '未通过'
  return ''
})
</script>

<style scoped>
.problem-meta-card {
  min-width: 0;
  padding: 20px;
}

.meta-card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--border);
}

.meta-card-header > div {
  display: grid;
  min-width: 0;
  gap: 5px;
}

.eyebrow,
.section-label {
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
}

.meta-card-header strong {
  overflow: hidden;
  color: var(--text);
  font-size: 24px;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta-list {
  display: grid;
  margin: 0;
}

.meta-item {
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
  padding: 13px 0;
  border-bottom: 1px solid color-mix(in srgb, var(--border) 72%, transparent);
}

.meta-item dt {
  color: var(--muted);
  font-size: 13px;
}

.meta-item dd {
  min-width: 0;
  margin: 0;
  color: var(--text);
}

.meta-item dd > span {
  color: var(--muted);
  font-size: 12px;
}

.status-image-wrap {
  display: flex;
  align-items: center;
  min-height: 42px;
}

.status-icon {
  width: min(146px, 100%);
  height: auto;
}

.tag-section {
  display: grid;
  gap: 10px;
  padding-top: 18px;
}

.tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
}
</style>
