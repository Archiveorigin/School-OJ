<template>
  <div class="submission-composer">
    <div class="composer-toolbar">
      <div class="language-field">
        <span>提交语言</span>
        <el-select :model-value="language" aria-label="提交语言" @update:model-value="changeLanguage(String($event))">
          <el-option label="C++17" value="cpp" />
          <el-option label="C" value="c" />
          <el-option label="Python 3" value="python" />
          <el-option label="Java 21" value="java" />
        </el-select>
      </div>
      <div v-if="status" class="verdict-field" aria-live="polite">
        <span>判题结果</span>
        <div><VerdictTag :status="status" /><small v-if="message">{{ message }}</small></div>
      </div>
      <span v-if="scopeText" class="scope-text">{{ scopeText }}</span>
      <span v-if="draftContext" class="draft-state">草稿自动保存</span>
      <slot name="options" />
      <div class="composer-actions">
        <el-button @click="editorRef?.format()">自动格式化</el-button>
        <el-button type="primary" :loading="submitting" :disabled="disabled" @click="emit('submit')">{{ submitLabel }}</el-button>
      </div>
    </div>
    <CodeEditor
      ref="editorRef"
      :model-value="source"
      :language="language"
      @update:model-value="updateSource(String($event))"
    />
    <slot name="after-editor" />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import CodeEditor from './CodeEditor.vue'
import VerdictTag from './VerdictTag.vue'
import { clearSubmissionDraft, loadSubmissionDraft, saveSubmissionDraft, type SubmissionDraftScope } from '../features/submissions/drafts'

const props = withDefaults(defineProps<{
  language: string
  source: string
  status?: string
  message?: string
  scopeText?: string
  submitting?: boolean
  disabled?: boolean
  submitLabel?: string
  draftContext?: SubmissionDraftScope
}>(), { status: '', message: '', scopeText: '', submitting: false, disabled: false, submitLabel: '提交评测' })

const emit = defineEmits<{
  'update:language': [value: string]
  'update:source': [value: string]
  submit: []
}>()
const editorRef = ref<InstanceType<typeof CodeEditor> | null>(null)

function updateSource(value: string) {
  if (props.draftContext) saveSubmissionDraft(props.draftContext, props.language, value)
  emit('update:source', value)
}

function changeLanguage(value: string) {
  if (props.draftContext) {
    saveSubmissionDraft(props.draftContext, props.language, props.source)
    emit('update:source', loadSubmissionDraft(props.draftContext, value) || '')
  }
  emit('update:language', value)
}

function clearDraft() {
  if (props.draftContext) clearSubmissionDraft(props.draftContext, props.language)
}

defineExpose({ clearDraft })
</script>

<style scoped>
.submission-composer { display: grid; gap: 14px; min-width: 0; }
.composer-toolbar { display: flex; align-items: end; flex-wrap: wrap; gap: 12px; }
.language-field, .verdict-field { display: grid; gap: 6px; }
.language-field > span, .verdict-field > span { color: var(--muted); font-size: 12px; font-weight: 700; }
.language-field :deep(.el-select) { width: 140px; }
.verdict-field > div { display: flex; align-items: center; gap: 8px; min-height: 32px; }
.verdict-field small, .scope-text, .draft-state { color: var(--muted); font-size: 12px; }
.scope-text { align-self: center; }
.composer-actions { display: flex; gap: 8px; margin-left: auto; }
@media (max-width: 680px) {
  .composer-toolbar { align-items: stretch; flex-direction: column; }
  .language-field :deep(.el-select) { width: 100%; }
  .composer-actions { display: grid; grid-template-columns: 1fr 1fr; margin-left: 0; }
}
</style>
