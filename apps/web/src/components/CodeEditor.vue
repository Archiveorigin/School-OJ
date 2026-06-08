<template>
  <div ref="host" class="monaco-host" />
</template>

<script setup lang="ts">
import type * as Monaco from 'monaco-editor'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps<{ modelValue: string; language: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const host = ref<HTMLElement | null>(null)
let monaco: typeof Monaco | null = null
let editor: Monaco.editor.IStandaloneCodeEditor | null = null
let disposed = false

onMounted(async () => {
  monaco = await import('monaco-editor')
  if (disposed || !host.value) return
  editor = monaco.editor.create(host.value, {
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
    if (model && monaco) monaco.editor.setModelLanguage(model, lang(value))
  }
)

watch(
  () => props.modelValue,
  (value) => {
    if (!editor || editor.getValue() === value) return
    editor.setValue(value)
  }
)

onBeforeUnmount(() => {
  disposed = true
  editor?.dispose()
  editor = null
})

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
  const formatted = language === 'python' ? formatPython(lines) : formatBraceLanguage(expandBraceLanguageLines(lines))
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

function expandBraceLanguageLines(lines: string[]) {
  const out: string[] = []
  for (const line of lines) {
    if (!line.trim()) {
      out.push('')
      continue
    }
    out.push(...splitBraceLanguageLine(line))
  }
  return out
}

function splitBraceLanguageLine(line: string) {
  const out: string[] = []
  let buffer = ''
  let parenDepth = 0
  let inString: '"' | "'" | null = null
  let escaped = false

  const flush = () => {
    const value = buffer.trim()
    if (value) out.push(value)
    buffer = ''
  }

  for (const char of line) {
    if (inString) {
      buffer += char
      if (escaped) {
        escaped = false
      } else if (char === '\\') {
        escaped = true
      } else if (char === inString) {
        inString = null
      }
      continue
    }
    if (char === '"' || char === "'") {
      inString = char
      buffer += char
      continue
    }
    if (char === '(') parenDepth += 1
    if (char === ')') parenDepth = Math.max(0, parenDepth - 1)
    if (char === '{' || char === '}') {
      flush()
      out.push(char)
      continue
    }
    buffer += char
    if (char === ';' && parenDepth === 0) flush()
  }
  flush()
  return out
}

function spaceCommonTokens(line: string) {
  return splitStringSegments(line)
    .map((part) => (part.string ? part.value : formatCodeSegment(part.value)))
    .join('')
    .replace(/\s+$/g, '')
}

function formatCodeSegment(value: string) {
  return value
    .replace(/\s*,\s*/g, ', ')
    .replace(/\b(if|for|while|switch|catch)\s*\(/g, '$1 (')
    .replace(/\s*(<<|>>)\s*/g, ' $1 ')
    .replace(/\s*(==|!=|<=|>=|&&|\|\|)\s*/g, ' $1 ')
    .replace(/\s*([+\-*/%])\s*/g, ' $1 ')
    .replace(/([^!<>=])\s*=\s*([^=])/g, '$1 = $2')
    .replace(/\s+/g, ' ')
    .replace(/\s+([;,)])/g, '$1')
    .replace(/([({])\s+/g, '$1')
}

function splitStringSegments(line: string) {
  const parts: Array<{ value: string; string: boolean }> = []
  let buffer = ''
  let inString: '"' | "'" | null = null
  let escaped = false

  const flush = (string: boolean) => {
    if (!buffer) return
    parts.push({ value: buffer, string })
    buffer = ''
  }

  for (const char of line) {
    if (inString) {
      buffer += char
      if (escaped) {
        escaped = false
      } else if (char === '\\') {
        escaped = true
      } else if (char === inString) {
        flush(true)
        inString = null
      }
      continue
    }
    if (char === '"' || char === "'") {
      flush(false)
      inString = char
      buffer += char
      continue
    }
    buffer += char
  }
  flush(Boolean(inString))
  return parts
}

defineExpose({ format })
</script>
