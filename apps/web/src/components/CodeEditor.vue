<template>
  <div class="code-editor" :aria-busy="loading">
    <div ref="host" class="monaco-host" />
    <div v-if="loading" class="editor-placeholder" role="status">编辑器加载中…</div>
    <textarea v-if="failed" class="editor-fallback" aria-label="代码编辑区" :value="modelValue" @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)" />
  </div>
</template>

<script setup lang="ts">
import type * as Monaco from 'monaco-editor'
import { ElMessage } from 'element-plus'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { editorLanguage, formatIndentation } from '../features/editor/format'

const props = defineProps<{ modelValue: string; language: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const host = ref<HTMLElement | null>(null)
const loading = ref(true)
const failed = ref(false)
let monaco: typeof Monaco | null = null
let editor: Monaco.editor.IStandaloneCodeEditor | null = null
let model: Monaco.editor.ITextModel | null = null
let subscription: Monaco.IDisposable | null = null
let themeObserver: MutationObserver | null = null
let disposed = false
let syncing = false
let formatting = false
let lastValue = props.modelValue

function syncTheme() {
  monaco?.editor.setTheme(document.documentElement.classList.contains('dark') ? 'vs-dark' : 'vs')
}

onMounted(async () => {
  try {
    const loaded = await import('../features/editor/monaco')
    if (disposed || !host.value) return
    monaco = loaded.monaco
    lastValue = props.modelValue
    model = monaco.editor.createModel(lastValue, editorLanguage(props.language))
    editor = monaco.editor.create(host.value, {
      model,
      minimap: { enabled: false },
      automaticLayout: true,
      fontSize: 14,
      // Judge languages have no semantic highlight provider. Avoid cross-model scans
      // and Monaco 0.52's uncaught delayed highlight cancellation on rapid disposal.
      occurrencesHighlight: 'off',
    })
    model.updateOptions({ tabSize: 2, insertSpaces: true })
    subscription = model.onDidChangeContent(() => {
      if (syncing || !model) return
      lastValue = model.getValue()
      emit('update:modelValue', lastValue)
    })
    syncTheme()
    themeObserver = new MutationObserver(syncTheme)
    themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  } catch {
    releaseEditor()
    if (!disposed) {
      failed.value = true
      ElMessage.warning('编辑器加载失败，已切换为文本编辑模式')
    }
  } finally {
    if (!disposed) loading.value = false
  }
})

watch(() => props.language, (value) => {
  if (model && monaco) monaco.editor.setModelLanguage(model, editorLanguage(value))
})
watch(() => props.modelValue, (value) => {
  if (!editor || lastValue === value) return
  syncing = true
  try { editor.setValue(value); lastValue = editor.getValue() }
  finally { syncing = false }
})

function releaseEditor() {
  themeObserver?.disconnect()
  subscription?.dispose()
  editor?.dispose()
  model?.dispose()
  editor = null
  model = null
}
onBeforeUnmount(() => { disposed = true; releaseEditor() })

async function format() {
  if (!editor || !model || formatting) return
  formatting = true
  const activeEditor = editor
  const activeModel = model
  const language = props.language
  const version = activeModel.getVersionId()
  const before = lastValue
  try {
    const action = activeEditor.getAction('editor.action.formatDocument')
    if (action?.isSupported()) {
      await action.run()
      return
    }
    const formatted = formatIndentation(before, language)
    if (disposed || activeModel !== model || activeModel.getVersionId() !== version || props.language !== language) return
    if (formatted === before) {
      ElMessage.info(language === 'python' ? '当前未配置 Python 格式化器，已保留原有缩进' : '无需调整缩进；复杂语法保留原文')
      return
    }
    const state = activeEditor.saveViewState()
    activeEditor.pushUndoStop()
    activeEditor.executeEdits('safe-indent', [{ range: activeModel.getFullModelRange(), text: formatted }])
    activeEditor.pushUndoStop()
    if (state) activeEditor.restoreViewState(state)
  } catch {
    if (!disposed) ElMessage.warning('格式化未完成，请保留并检查原代码')
  } finally { formatting = false }
}

defineExpose({ format })
</script>

<style scoped>
.code-editor { position: relative; min-width: 0; }
.monaco-host { width: 100%; height: 420px; min-height: 240px; border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }
.editor-placeholder { position: absolute; inset: 0; display: grid; place-items: center; background: var(--surface); color: var(--muted); border-radius: 8px; }
.editor-fallback { position: absolute; inset: 0; width: 100%; height: 100%; padding: 12px; resize: none; font: 14px/1.6 monospace; background: var(--surface); color: var(--text); border: 1px solid var(--border); border-radius: 8px; }
</style>
