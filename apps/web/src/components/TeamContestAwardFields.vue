<template>
  <div
    class="award-allocation"
    :class="{ 'award-allocation--invalid': validationError }"
  >
    <div class="award-unit">
      <span>设置单位</span>
      <el-radio-group v-model="unit" size="small" aria-label="奖项设置单位">
        <el-radio-button value="percent">比例（%）</el-radio-button>
        <el-radio-button value="count">人数（人）</el-radio-button>
      </el-radio-group>
    </div>
    <div v-if="unit === 'count'" class="award-basis">
      <span>换算基数（预计参赛人数）</span>
      <el-input-number
        v-model="participants"
        :min="1"
        :max="1000000"
        :precision="0"
        controls-position="right"
        aria-label="预计参赛人数"
        placeholder="输入人数"
      />
    </div>
    <div class="award-fields">
      <div
        v-for="(medal, index) in medals"
        :key="medal.key"
        class="award-field"
        :class="`award-field--${medal.key}`"
      >
        <span><i aria-hidden="true" />{{ medal.label }}</span>
        <el-input-number
          :model-value="
            unit === 'percent'
              ? percents[index]
              : validBasis
                ? counts[index]
                : undefined
          "
          :disabled="unit === 'count' && !validBasis"
          :min="0"
          :max="unit === 'percent' ? 100 : participants"
          :precision="unit === 'count' ? 0 : undefined"
          :step="1"
          controls-position="right"
          :aria-label="`${medal.label}${unit === 'percent' ? '百分比' : '人数'}`"
          @update:model-value="update(index, $event)"
        />
      </div>
    </div>
    <div class="award-summary" aria-live="polite">
      <div>
        <span>奖项占比合计</span><strong>{{ total }}%</strong>
      </div>
      <p v-if="validationError" class="award-error">{{ validationError }}</p>
      <p v-else-if="unit === 'count'">
        按 {{ participants }} 人换算为比例：金 {{ percents[0] }}% · 银
        {{ percents[1] }}% · 铜 {{ percents[2] }}%。
      </p>
      <p>
        最终按实际参赛人数计算累计名额并向上取整；同名次同奖，实际获奖人数可能增加。
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import {
  awardCountToPercent,
  awardCountsFromPercents,
  contestAwardTotal,
  contestAwardValidationError,
} from "../features/teams/contestAwards";
const props = defineProps<{
  goldAwardPercent: number;
  silverAwardPercent: number;
  bronzeAwardPercent: number;
}>();
const emit = defineEmits<{
  "update:goldAwardPercent": [value: number];
  "update:silverAwardPercent": [value: number];
  "update:bronzeAwardPercent": [value: number];
  "validity-change": [valid: boolean];
}>();
const unit = ref<"percent" | "count">("percent");
const participants = ref<number>();
const medals = [
  { key: "gold", label: "金奖" },
  { key: "silver", label: "银奖" },
  { key: "bronze", label: "铜奖" },
];
const percents = computed(() => [
  props.goldAwardPercent,
  props.silverAwardPercent,
  props.bronzeAwardPercent,
]);
const allocation = computed(() => ({
  gold_award_percent: props.goldAwardPercent,
  silver_award_percent: props.silverAwardPercent,
  bronze_award_percent: props.bronzeAwardPercent,
}));
const validBasis = computed(
  () =>
    Number.isInteger(participants.value) &&
    participants.value! >= 1 &&
    participants.value! <= 1000000,
);
const counts = computed(() =>
  awardCountsFromPercents(
    allocation.value,
    validBasis.value ? participants.value! : 0,
  ),
);
const total = computed(() =>
  Number(contestAwardTotal(allocation.value).toFixed(6)),
);
const validationError = computed(() =>
  unit.value === "count" && !validBasis.value
    ? "请填写预计参赛人数作为换算基数"
    : contestAwardValidationError(allocation.value),
);
watch(validationError, (error) => emit("validity-change", !error), {
  immediate: true,
});
function emitValue(index: number, value: number) {
  if (index === 0) emit("update:goldAwardPercent", value);
  else if (index === 1) emit("update:silverAwardPercent", value);
  else emit("update:bronzeAwardPercent", value);
}
function update(index: number, value: number | undefined) {
  if (unit.value === "percent") {
    emitValue(index, Number(Number(value ?? 0).toFixed(6)));
    return;
  }
  if (!validBasis.value) return;
  const next = [...counts.value];
  next[index] = Number(value ?? 0);
  next.forEach((count, i) =>
    emitValue(i, awardCountToPercent(count, participants.value!)),
  );
}
</script>

<style scoped>
.award-unit,
.award-basis {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  color: var(--muted);
  font-size: 13px;
}
.award-allocation {
  width: 100%;
  display: grid;
  gap: 12px;
}
.award-fields {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}
.award-field {
  display: grid;
  gap: 7px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: color-mix(in srgb, var(--surface-strong) 88%, transparent);
}
.award-field > span {
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--text);
  font-size: 13px;
  font-weight: 800;
}
.award-field i {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--medal-color);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--medal-color) 16%, transparent);
}
.award-field--gold {
  --medal-color: #d6a929;
}
.award-field--silver {
  --medal-color: #8d99a8;
}
.award-field--bronze {
  --medal-color: #b97842;
}
.award-field :deep(.el-input-number) {
  width: 100%;
}
.award-summary {
  padding: 11px 13px;
  border: 1px solid color-mix(in srgb, var(--accent) 20%, var(--border));
  border-radius: 9px;
  background: color-mix(in srgb, var(--accent) 6%, var(--surface-strong));
}
.award-summary > div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.award-summary strong {
  color: var(--accent);
  font-size: 18px;
}
.award-summary p {
  margin: 4px 0 0;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.5;
}
.award-allocation--invalid .award-summary {
  border-color: color-mix(in srgb, #dc2626 55%, var(--border));
  background: color-mix(in srgb, #dc2626 7%, var(--surface-strong));
}
.award-allocation--invalid .award-summary strong,
.award-summary .award-error {
  color: #dc2626;
}
@media (max-width: 560px) {
  .award-fields {
    grid-template-columns: 1fr;
  }
  .award-field {
    grid-template-columns: minmax(72px, 1fr) minmax(0, 1.5fr);
    align-items: center;
  }
}
</style>
