<template>
  <section v-if="activeProblem" class="panel submit-panel">
    <div class="section-title">
      <h3>提交代码</h3>
      <span class="muted">{{ activeEntry?.label || `#${activeProblem.id}` }} {{ activeProblem.title }}</span>
    </div>
    <div class="toolbar editor-toolbar">
      <el-select :model-value="language" style="width: 130px" @update:model-value="emit('update:language', String($event))">
        <el-option label="C++17" value="cpp" />
        <el-option label="C" value="c" />
        <el-option label="Python" value="python" />
        <el-option label="Java" value="java" />
      </el-select>
      <el-button @click="formatSource">自动格式化</el-button>
      <el-button type="primary" :loading="submitting" :disabled="!detail.can_submit" @click="emit('submit')">提交</el-button>
    </div>
    <CodeEditor
      :key="activeProblem.id"
      ref="editorRef"
      :model-value="source"
      :language="language"
      @update:model-value="emit('update:source', String($event))"
    />
    <div v-if="live" class="live">
      <StatusBadge :status="live.status" />
      {{ live.status === 'pending_review' ? '等待教师评分' : `分数 ${live.score}，${live.message}` }}
    </div>
  </section>
  <div v-else class="panel empty-detail muted">请选择题目</div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { Problem } from '../../api/client'
import CodeEditor from '../../components/CodeEditor.vue'
import StatusBadge from '../../components/StatusBadge.vue'

defineProps<{
  detail: any
  activeEntry: { problem: Problem; score: number; label?: string; problem_id: number } | null
  activeProblem: Problem | null
  language: string
  source: string
  live: any
  submitting: boolean
}>()

const emit = defineEmits<{
  'update:language': [value: string]
  'update:source': [value: string]
  submit: []
}>()

const editorRef = ref<InstanceType<typeof CodeEditor> | null>(null)

function formatSource() {
  editorRef.value?.format()
}
</script>

<style scoped>
.submit-panel {
  display: grid;
  grid-template-rows: auto auto minmax(460px, 1fr) auto;
  gap: 10px;
}

.editor-toolbar {
  justify-content: flex-end;
}

.live {
  display: flex;
  align-items: center;
  gap: 10px;
}

.empty-detail {
  display: grid;
  min-height: 260px;
  place-items: center;
}
</style>
