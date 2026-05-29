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

function lang(value: string) {
  if (value === 'cpp') return 'cpp'
  if (value === 'c') return 'c'
  if (value === 'java') return 'java'
  return 'python'
}
</script>
