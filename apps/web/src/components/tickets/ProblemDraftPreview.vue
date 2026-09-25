<template>
  <section class="problem-draft-preview" aria-label="新题目实时预览">
    <header><span>实时预览</span><el-button link @click="expanded = true">展开预览</el-button></header>
    <h1>{{ title || '题目预览' }}</h1>
    <MarkdownRenderer :source="previewSource" :asset-urls="assetUrls" />
    <el-dialog v-model="expanded" title="题目预览" width="min(1100px, calc(100vw - 24px))" append-to-body destroy-on-close>
      <article class="expanded-preview"><h1>{{ title || '题目预览' }}</h1><MarkdownRenderer :source="previewSource" :asset-urls="assetUrls" /></article>
    </el-dialog>
  </section>
</template>
<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import MarkdownRenderer from '../MarkdownRenderer.vue'
const props = defineProps<{ title: string; source: string; assetUrls?: Record<string, string> }>()
const previewSource = ref(props.source), expanded = ref(false)
let timer: ReturnType<typeof setTimeout> | undefined
watch(() => props.source, value => { clearTimeout(timer); timer = setTimeout(() => { previewSource.value = value }, 150) })
onBeforeUnmount(() => clearTimeout(timer))
</script>
<style scoped>
.problem-draft-preview { min-width: 0; align-self: start; padding: clamp(16px, 2vw, 28px); border: 1px solid var(--border); border-radius: 10px; background: var(--surface); overflow-wrap: anywhere; }
header { display: flex; justify-content: space-between; align-items: center; gap: 12px; color: var(--muted); font-size: 12px; margin-bottom: 16px; }
h1 { margin: 0 0 24px; font-size: clamp(24px, 2vw, 32px); line-height: 1.4; }
.expanded-preview { max-width: 900px; margin: 0 auto; min-width: 0; overflow-wrap: anywhere; }
</style>
