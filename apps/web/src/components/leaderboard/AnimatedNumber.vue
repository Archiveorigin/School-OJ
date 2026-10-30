<template>
  <span class="animated-number" :aria-label="String(value)"
    >{{ displayed }}<span v-if="max !== undefined">/{{ max }}</span></span
  >
</template>
<script setup lang="ts">
import { useAnimatedNumber } from "../../features/leaderboard/useAnimatedNumber";
const props = withDefaults(
  defineProps<{
    value: number;
    max?: number;
    enabled?: boolean;
    speed?: number;
  }>(),
  { enabled: true, speed: 1 },
);
const displayed = useAnimatedNumber(
  () => props.value,
  () => props.enabled,
  () => props.speed,
);
</script>
<style scoped>
.animated-number {
  font-variant-numeric: tabular-nums;
}
</style>
