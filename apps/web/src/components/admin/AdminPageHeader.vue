<template>
  <header class="admin-page-header">
    <div class="admin-page-title">
      <span v-if="eyebrow">{{ eyebrow }}</span>
      <component :is="headingTag">{{ title }}</component>
    </div>
    <div v-if="$slots.default" class="admin-page-actions">
      <slot />
    </div>
  </header>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string
    eyebrow?: string
    headingTag?: 'h1' | 'h2'
  }>(),
  { eyebrow: '', headingTag: 'h1' },
)
</script>

<style scoped>
.admin-page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 18px;
}
.admin-page-title {
  min-width: 0;
}
.admin-page-title > span {
  color: var(--admin-accent);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.14em;
}
.admin-page-title h1,
.admin-page-title h2 {
  margin: 6px 0 0;
  color: var(--admin-ink);
  font-size: 30px;
  line-height: 1.2;
}
.admin-page-title h2 {
  font-size: 26px;
}
.admin-page-actions {
  display: flex;
  align-items: center;
  gap: 9px;
}
@media (max-width: 680px) {
  .admin-page-header {
    align-items: stretch;
    flex-direction: column;
    gap: 14px;
  }
  .admin-page-actions {
    flex-wrap: wrap;
  }
}
</style>
