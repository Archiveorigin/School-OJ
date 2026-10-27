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
    <TeamContestRankingRoster
      v-if="teamId"
      :team-id="teamId"
      :model-value="rankingUserIds ?? null"
      @update:model-value="emit('update:rankingUserIds', $event)"
      @count-change="participants = $event"
      @validity-change="rosterValid = $event"
    />
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
    <p v-if="validationError" role="alert" class="award-error">
      {{ validationError }}
    </p>
    <p v-else-if="unit === 'count'" class="award-note">
      按名单 {{ participants }} 人换算：金 {{ percents[0] }}% · 银
      {{ percents[1] }}% · 铜 {{ percents[2] }}%。
    </p>
    <p class="award-note">
      按排名名单人数计算累计名额并向上取整；同名次同奖，实际获奖人数可能增加。全设为
      0 可不评奖。
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import {
  awardCountToPercent,
  awardCountsFromPercents,
  contestAwardValidationError,
} from "../features/teams/contestAwards";
import TeamContestRankingRoster from "./TeamContestRankingRoster.vue";
const props = defineProps<{
  teamId?: number;
  rankingUserIds?: number[] | null;
  goldAwardPercent: number;
  silverAwardPercent: number;
  bronzeAwardPercent: number;
}>();
const emit = defineEmits<{
  "update:rankingUserIds": [value: number[] | null];
  "update:goldAwardPercent": [value: number];
  "update:silverAwardPercent": [value: number];
  "update:bronzeAwardPercent": [value: number];
  "validity-change": [valid: boolean];
}>();
const unit = ref<"percent" | "count">("percent");
const participants = ref(0);
const rosterValid = ref(!props.teamId);
const countDraft = ref([0, 0, 0]);
const basisReady = ref(false);
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
const counts = computed(() => countDraft.value);
const validationError = computed(() => {
  if (!rosterValid.value) return "请检查排名名单是否已加载且成员有效";
  if (unit.value === "count") {
    if (!validBasis.value) return "请至少选择一名排名人员，才能按人数设置奖项";
    if (
      counts.value.some((n) => !Number.isInteger(n) || n < 0) ||
      counts.value.reduce((a, b) => a + b, 0) > participants.value
    )
      return "获奖人数合计不能超过名单人数，请调整奖项人数";
  }
  return contestAwardValidationError(allocation.value);
});
watch(unit, (value) => {
  if (value === "count")
    countDraft.value = awardCountsFromPercents(
      allocation.value,
      participants.value,
    );
});
watch(participants, () => {
  if (!validBasis.value) {
    basisReady.value = false;
    return;
  }
  // The first valid basis can arrive after a failed member-list load. Derive
  // the counts from the stored percentages instead of persisting a zero draft.
  if (!basisReady.value) {
    basisReady.value = true;
    if (unit.value === "count")
      countDraft.value = awardCountsFromPercents(
        allocation.value,
        participants.value,
      );
  }
  if (unit.value === "count") emitCounts();
});
function emitCounts() {
  if (
    !validBasis.value ||
    counts.value.reduce((a, b) => a + b, 0) > participants.value
  )
    return;
  counts.value.forEach((count, i) =>
    emitValue(i, awardCountToPercent(count, participants.value)),
  );
}
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
  countDraft.value[index] = Number(value ?? 0);
  emitCounts();
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
.award-note {
  margin: 0;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.6;
}
.award-error {
  margin: 0;
  color: var(--el-color-danger);
  font-size: 13px;
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
