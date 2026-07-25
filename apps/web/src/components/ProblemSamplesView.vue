<template>
  <section v-if="samples.length" class="sample-section">
    <div class="section-title">
      <h3>输入输出样例</h3>
    </div>
    <div v-for="sample in samples" :key="sample.index" class="sample-card">
      <div class="sample-name">
        <strong>{{ sample.name || `样例 ${sample.index}` }}</strong>
      </div>
      <div class="sample-pair">
        <div class="sample-block">
          <div class="sample-head">
            <strong>输入</strong>
            <el-button size="small" text @click="copyText(sample.input)">复制</el-button>
          </div>
          <pre>{{ sample.input }}</pre>
        </div>
        <div class="sample-block">
          <div class="sample-head">
            <strong>输出</strong>
            <el-button size="small" text @click="copyText(sample.output)">复制</el-button>
          </div>
          <pre>{{ sample.output }}</pre>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { copyTextToClipboard } from '../features/clipboard'
import type { ProblemSample } from '../features/problems/problemMeta'

defineProps<{
  samples: ProblemSample[]
}>()

async function copyText(value: string) {
  try {
    await copyTextToClipboard(value)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败，请手动选择文本')
  }
}
</script>

<style scoped>
.sample-section {
  display: grid;
  gap: 12px;
  margin-top: 22px;
}

.section-title h3 {
  margin: 0;
}

.sample-card {
  display: grid;
  gap: 10px;
}

.sample-name {
  display: block;
}

.sample-name strong {
  color: var(--text);
  font-size: 15px;
}

.sample-pair {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 8px;
}

.sample-block {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: color-mix(in srgb, var(--surface-strong) 72%, transparent);
}

.sample-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 10px;
  border-bottom: 1px solid var(--border);
}

.sample-block pre {
  min-height: 48px;
  max-height: 160px;
  overflow: auto;
  margin: 0;
  padding: 9px 10px;
  color: #e2e8f0;
  background: #0f172a;
  white-space: pre;
}

</style>
