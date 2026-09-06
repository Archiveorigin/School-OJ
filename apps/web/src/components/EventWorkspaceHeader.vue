<template>
  <header class="event-header">
    <button v-if="backLabel" class="back-link" type="button" @click="$emit('back')">{{ backLabel }}</button>
    <div class="event-title-row">
      <div><h1>{{ title }}</h1><p v-if="subtitle">{{ subtitle }}</p></div>
      <div class="event-stats">
        <article><span>赛制</span><strong>{{ rule.toUpperCase() }}</strong></article>
        <article><span>状态</span><strong>{{ status }}</strong></article>
        <slot name="actions" />
      </div>
    </div>
    <nav class="event-nav" :aria-label="`${kind}导航`">
      <router-link :to="`${base}/description`">{{ kind }}说明</router-link>
      <router-link :to="`${base}/problems`">题目列表</router-link>
      <router-link v-if="showRanking && rule !== 'oi'" :to="`${base}/scoreboard`">排行榜</router-link>
    </nav>
  </header>
</template>

<script setup lang="ts">
defineProps<{ title: string; subtitle?: string; rule: string; status: string; kind: '比赛' | '考试'; base: string; showRanking: boolean; backLabel?: string }>()
defineEmits<{ back: [] }>()
</script>

<style scoped>
.back-link { padding: 0; color: var(--muted); border: 0; background: transparent; cursor: pointer; }
.event-title-row { display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 18px 0 22px; }
.event-title-row h1 { margin: 0 0 6px; font-size: clamp(23px, 3vw, 30px); overflow-wrap: anywhere; }
.event-title-row p { margin: 0; color: var(--muted); }
.event-stats { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; }
.event-stats article { min-width: 95px; padding: 8px 14px; text-align: center; border: 1px solid var(--border); border-radius: 8px; background: var(--surface-strong); }
.event-stats span, .event-stats strong { display: block; }
.event-stats span { color: var(--muted); font-size: 13px; }
.event-stats strong { margin-top: 4px; }
.event-nav { display: flex; border-bottom: 1px solid var(--border); }
.event-nav a { padding: 13px 22px; color: var(--muted); border-bottom: 3px solid transparent; }
.event-nav a.router-link-active { color: var(--accent); border-bottom-color: var(--accent); font-weight: 800; }
@media (max-width: 680px) { .event-title-row { align-items: stretch; flex-direction: column; gap: 12px; } .event-nav a { flex: 1; padding: 12px 6px; text-align: center; } }
</style>
