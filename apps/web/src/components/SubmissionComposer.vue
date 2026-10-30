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
        <div><VerdictTag :status="status" /></div>
      </div>
      <slot name="options" />
      <div class="composer-actions">
        <el-button @click="editorRef?.format()">自动格式化</el-button>
        <el-button type="primary" :loading="submitting" :disabled="disabled || submitting" @click="submit">{{ submitLabel }}</el-button>
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
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import CodeEditor from './CodeEditor.vue'
import VerdictTag from './VerdictTag.vue'
import { clearSubmissionDraft, loadSubmissionDraft, saveSubmissionDraft, submissionDraftKey, type SubmissionDraftScope } from '../features/submissions/drafts'

const props = withDefaults(defineProps<{
  language: string
  source: string
  status?: string
  submitting?: boolean
  disabled?: boolean
  submitLabel?: string
  draftContext?: SubmissionDraftScope
}>(), { status: '', submitting: false, disabled: false, submitLabel: '提交评测' })

const emit = defineEmits<{
  'update:language': [value: string]
  'update:source': [value: string]
  submit: []
}>()
const editorRef = ref<InstanceType<typeof CodeEditor> | null>(null)
let pendingDraft: { scope: SubmissionDraftScope; language: string; source: string } | null = null
let draftTimer: ReturnType<typeof setTimeout> | undefined

function flushDraft() {
  clearTimeout(draftTimer)
  draftTimer = undefined
  const draft = pendingDraft
  pendingDraft = null
  if (draft) saveSubmissionDraft(draft.scope, draft.language, draft.source)
}

function onVisibilityChange() {
  if (document.visibilityState === 'hidden') flushDraft()
}

watch(() => props.draftContext ? submissionDraftKey(props.draftContext, props.language) : '', flushDraft, { flush: 'sync' })
onMounted(() => {
  window.addEventListener('pagehide', flushDraft)
  document.addEventListener('visibilitychange', onVisibilityChange)
})
onBeforeUnmount(() => {
  flushDraft()
  window.removeEventListener('pagehide', flushDraft)
  document.removeEventListener('visibilitychange', onVisibilityChange)
})

function updateSource(value: string) {
  emit('update:source', value)
  if (props.draftContext) {
    pendingDraft = { scope: { ...props.draftContext }, language: props.language, source: value }
    // At most one synchronous storage write per 300 ms, even while typing continuously.
    if (draftTimer === undefined) draftTimer = setTimeout(flushDraft, 300)
  }
}

function changeLanguage(value: string) {
  flushDraft()
  if (props.draftContext) {
    saveSubmissionDraft(props.draftContext, props.language, props.source)
    emit('update:source', loadSubmissionDraft(props.draftContext, value) || '')
  }
  emit('update:language', value)
}

function clearDraft() {
  clearTimeout(draftTimer)
  draftTimer = undefined
  pendingDraft = null
  if (props.draftContext) clearSubmissionDraft(props.draftContext, props.language)
}

function submit() {
  if (props.submitting || props.disabled) return
  flushDraft()
  emit('submit')
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
.composer-actions { display: flex; gap: 8px; margin-left: auto; }
@media (max-width: 680px) {
  .composer-toolbar { align-items: stretch; flex-direction: column; }
  .language-field :deep(.el-select) { width: 100%; }
  .composer-actions { display: grid; grid-template-columns: 1fr 1fr; margin-left: 0; }
}
</style>
