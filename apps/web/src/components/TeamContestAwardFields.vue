<template>
  <div class="award-allocation" :class="{ 'award-allocation--invalid': validationError }">
    <div class="award-fields">
      <label class="award-field award-field--gold">
        <span><i aria-hidden="true" />金奖</span>
        <el-input-number
          :model-value="goldAwardPercent"
          :min="0"
          :max="100"
          :precision="0"
          :step="1"
          controls-position="right"
          aria-label="金奖百分比"
          @update:model-value="$emit('update:goldAwardPercent', Number($event ?? 0))"
        />
      </label>
      <label class="award-field award-field--silver">
        <span><i aria-hidden="true" />银奖</span>
        <el-input-number
          :model-value="silverAwardPercent"
          :min="0"
          :max="100"
          :precision="0"
          :step="1"
          controls-position="right"
          aria-label="银奖百分比"
          @update:model-value="$emit('update:silverAwardPercent', Number($event ?? 0))"
        />
      </label>
      <label class="award-field award-field--bronze">
        <span><i aria-hidden="true" />铜奖</span>
        <el-input-number
          :model-value="bronzeAwardPercent"
          :min="0"
          :max="100"
          :precision="0"
          :step="1"
          controls-position="right"
          aria-label="铜奖百分比"
          @update:model-value="$emit('update:bronzeAwardPercent', Number($event ?? 0))"
        />
      </label>
    </div>
    <div class="award-summary" aria-live="polite">
      <div>
        <span>奖项占比合计</span>
        <strong>{{ total }}%</strong>
      </div>
      <p v-if="validationError" class="award-error">{{ validationError }}</p>
      <p v-else>排名按实际参赛人数计算，未覆盖的选手不显示奖牌。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { contestAwardTotal, contestAwardValidationError } from '../features/teams/contestAwards'

const props = defineProps<{
  goldAwardPercent: number
  silverAwardPercent: number
  bronzeAwardPercent: number
}>()

defineEmits<{
  'update:goldAwardPercent': [value: number]
  'update:silverAwardPercent': [value: number]
  'update:bronzeAwardPercent': [value: number]
}>()

const allocation = computed(() => ({
  gold_award_percent: props.goldAwardPercent,
  silver_award_percent: props.silverAwardPercent,
  bronze_award_percent: props.bronzeAwardPercent,
}))
const total = computed(() => contestAwardTotal(allocation.value))
const validationError = computed(() => contestAwardValidationError(allocation.value))
</script>

<style scoped>
.award-allocation { width: 100%; display: grid; gap: 12px; }
.award-fields { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; }
.award-field { display: grid; gap: 7px; padding: 12px; border: 1px solid var(--border); border-radius: 10px; background: color-mix(in srgb, var(--surface-strong) 88%, transparent); }
.award-field > span { display: flex; align-items: center; gap: 7px; color: var(--text); font-size: 13px; font-weight: 800; }
.award-field i { width: 9px; height: 9px; border-radius: 50%; background: var(--medal-color); box-shadow: 0 0 0 3px color-mix(in srgb, var(--medal-color) 16%, transparent); }
.award-field--gold { --medal-color: #d6a929; }
.award-field--silver { --medal-color: #8d99a8; }
.award-field--bronze { --medal-color: #b97842; }
.award-field :deep(.el-input-number) { width: 100%; }
.award-summary { padding: 11px 13px; border: 1px solid color-mix(in srgb, var(--accent) 20%, var(--border)); border-radius: 9px; background: color-mix(in srgb, var(--accent) 6%, var(--surface-strong)); }
.award-summary > div { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.award-summary strong { color: var(--accent); font-size: 18px; }
.award-summary p { margin: 4px 0 0; color: var(--muted); font-size: 12px; line-height: 1.5; }
.award-allocation--invalid .award-summary { border-color: color-mix(in srgb, #dc2626 55%, var(--border)); background: color-mix(in srgb, #dc2626 7%, var(--surface-strong)); }
.award-allocation--invalid .award-summary strong, .award-summary .award-error { color: #dc2626; }
@media (max-width: 560px) { .award-fields { grid-template-columns: 1fr; } .award-field { grid-template-columns: minmax(72px, 1fr) minmax(0, 1.5fr); align-items: center; } }
</style>
