<template>
  <span
    v-if="change"
    class="rank-delta"
    :class="change.delta < 0 ? 'is-down' : 'is-up'"
    :aria-label="
      change.isNew
        ? '新进入榜单'
        : `${change.delta > 0 ? '上升' : '下降'} ${Math.abs(change.delta)} 名`
    "
    >{{
      change.isNew
        ? "NEW"
        : `${change.delta > 0 ? "↑" : "↓"}${Math.abs(change.delta)}`
    }}</span
  >
</template>
<script setup lang="ts">
import type { RankDelta } from "../../features/leaderboard/useRankDelta";
defineProps<{ change?: RankDelta }>();
</script>
<style scoped>
.rank-delta {
  position: absolute;
  right: 1px;
  top: -2px;
  z-index: 4;
  font-size: 10px;
  font-weight: 800;
  line-height: 16px;
  padding: 0 3px;
  border-radius: 4px;
  background: var(--score-surface, var(--surface));
  color: var(--score-rank-up);
  animation: delta-enter 180ms ease-out;
  pointer-events: none;
}
.is-down {
  color: var(--score-rank-down);
}
@keyframes delta-enter {
  from {
    opacity: 0;
    transform: translateY(4px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}
@media (prefers-reduced-motion: reduce) {
  .rank-delta {
    animation: none;
  }
}
</style>
