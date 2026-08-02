<template>
  <div class="problem-switcher-grid" role="listbox" aria-label="题目选择">
    <button
      v-for="(item, index) in items"
      :key="item.problem.id"
      type="button"
      role="option"
      :aria-selected="item.problem.id === modelValue"
      :title="`${displayLabel(item, index)} · ${item.problem.title}`"
      :class="{
        active: item.problem.id === modelValue,
        accepted: item.submission_status === 'accepted',
        attempted: Boolean(item.submission_status) && item.submission_status !== 'accepted'
      }"
      @click="emit('update:modelValue', item.problem.id)"
    >
      {{ displayLabel(item, index) }}
    </button>
  </div>
</template>

<script setup lang="ts">
type SwitcherItem = {
  problem: { id: number; title: string }
  label?: string
  submission_status?: string
}

defineProps<{ items: SwitcherItem[]; modelValue?: number | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: number] }>()

function displayLabel(item: SwitcherItem, index: number) {
  if (item.label?.trim()) return item.label.trim()
  index += 1
  let label = ''
  while (index > 0) {
    index -= 1
    label = String.fromCharCode(65 + (index % 26)) + label
    index = Math.floor(index / 26)
  }
  return label
}
</script>

<style scoped>
.problem-switcher-grid {
  display: grid;
  grid-template-columns: repeat(8, minmax(40px, 1fr));
  gap: 8px 6px;
  width: min(420px, 100%);
}

.problem-switcher-grid button {
  position: relative;
  min-width: 40px;
  height: 44px;
  padding: 0 7px;
  overflow: hidden;
  color: #28a9c8;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  font-family: Georgia, 'Times New Roman', serif;
  font-size: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
  transition: color .16s ease, border-color .16s ease, background .16s ease, transform .16s ease;
}

.problem-switcher-grid button:hover {
  color: var(--accent);
  border-color: color-mix(in srgb, var(--accent) 42%, transparent);
  background: color-mix(in srgb, var(--accent) 8%, transparent);
  transform: translateY(-1px);
}

.problem-switcher-grid button.active {
  color: #fff;
  border-color: #1473ff;
  background: #1473ff;
  box-shadow: 0 5px 14px rgba(20, 115, 255, .24);
  font-family: inherit;
}

.problem-switcher-grid button.accepted:not(.active) { color: #16a34a; }
.problem-switcher-grid button.attempted:not(.active) { color: #d97706; }

@media (max-width: 520px) {
  .problem-switcher-grid { grid-template-columns: repeat(6, minmax(38px, 1fr)); }
}
</style>
