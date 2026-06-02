<template>
  <div ref="host" class="monaco-host" />
</template>

<script setup lang="ts">
import * as monaco from 'monaco-editor'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps<{ modelValue: string; language: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const host = ref<HTMLElement | null>(null)
let editor: monaco.editor.IStandaloneCodeEditor | null = null

onMounted(() => {
  editor = monaco.editor.create(host.value!, {
    value: props.modelValue,
    language: lang(props.language),
    minimap: { enabled: false },
    automaticLayout: true,
    fontSize: 14,
    tabSize: 2
  })
  editor.onDidChangeModelContent(() => emit('update:modelValue', editor?.getValue() || ''))
})

watch(
  () => props.language,
  (value) => {
    const model = editor?.getModel()
    if (model) monaco.editor.setModelLanguage(model, lang(value))
  }
)

onBeforeUnmount(() => editor?.dispose())

async function format() {
  if (!editor) return
  const before = editor.getValue()
  try {
    await editor.getAction('editor.action.formatDocument')?.run()
  } catch {
    // Monaco does not ship formatters for every judge language.
  }
  if (editor.getValue() === before) {
    editor.setValue(lightweightFormat(before, props.language))
  }
}

function lang(value: string) {
  if (value === 'cpp') return 'cpp'
  if (value === 'c') return 'c'
  if (value === 'java') return 'java'
  return 'python'
}

function lightweightFormat(source: string, language: string) {
  const lines = source.replace(/\r\n/g, '\n').replace(/\r/g, '\n').replace(/\t/g, '  ').split('\n')
  const formatted = language === 'python' ? formatPython(lines) : formatBraceLanguage(lines)
  return `${formatted.join('\n').replace(/\s+$/g, '')}\n`
}

function formatPython(lines: string[]) {
  const out: string[] = []
  let indent = 0
  const dedent = /^(elif|else|except|finally)\b/
  for (const line of lines) {
    const trimmed = line.trim()
    if (!trimmed) {
      out.push('')
      continue
    }
    if (dedent.test(trimmed)) indent = Math.max(0, indent - 1)
    out.push(`${'  '.repeat(indent)}${trimmed}`)
    if (trimmed.endsWith(':')) indent += 1
  }
  return out
}

function formatBraceLanguage(lines: string[]) {
  const out: string[] = []
  let indent = 0
  for (const line of lines) {
    const trimmed = line.trim()
    if (!trimmed) {
      out.push('')
      continue
    }
    if (trimmed.startsWith('#')) {
      out.push(trimmed)
      continue
    }
    if (/^[}\])]/.test(trimmed)) indent = Math.max(0, indent - 1)
    out.push(`${'  '.repeat(indent)}${spaceCommonTokens(trimmed)}`)
    const opens = (trimmed.match(/[{\[]/g) || []).length
    const closes = (trimmed.match(/[}\]]/g) || []).length
    indent = Math.max(0, indent + opens - closes)
  }
  return out
}

function spaceCommonTokens(line: string) {
  return line
    .replace(/,\s*/g, ', ')
    .replace(/\b(if|for|while|switch|catch)\(/g, '$1 (')
    .replace(/\s+$/g, '')
}

defineExpose({ format })
</script>
