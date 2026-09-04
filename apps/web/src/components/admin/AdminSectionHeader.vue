<template>
  <header class="admin-section-header">
    <component :is="headingTag">{{ title }}</component>
    <div v-if="$slots.default" class="admin-section-actions"><slot /></div>
  </header>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string
    headingTag?: 'h2' | 'h3'
  }>(),
  { headingTag: 'h2' },
)
</script>

<style scoped>
.admin-section-header {
  display: flex;
  min-height: 62px;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 12px 18px;
  border-bottom: 1px solid var(--admin-line-soft);
}
.admin-section-header h2,
.admin-section-header h3 {
  margin: 0;
  color: var(--admin-ink);
  font-size: 17px;
}
.admin-section-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
